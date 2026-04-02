package api

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rkeese/alex-backend/internal/database"
	"github.com/rkeese/alex-backend/internal/email"
)

type Server struct {
	Queries     *database.Queries
	Pool        *pgxpool.Pool
	EmailSender email.Sender
	AuthLimiter *RateLimiter
}

func NewServer(queries *database.Queries, pool *pgxpool.Pool, emailSender email.Sender) *Server {
	return &Server{
		Queries:     queries,
		Pool:        pool,
		EmailSender: emailSender,
		AuthLimiter: NewRateLimiter(10, time.Minute),
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("POST /api/v1/auth/register", s.AuthLimiter.Middleware(http.HandlerFunc(s.handleRegister)))
	mux.Handle("POST /api/v1/auth/login", s.AuthLimiter.Middleware(http.HandlerFunc(s.handleLogin)))
	mux.HandleFunc("GET /health", s.handleHealth)

	// Protected routes
	mux.Handle("GET /api/v1/clubs", s.AuthMiddleware(http.HandlerFunc(s.handleListClubs)))
	mux.Handle("POST /api/v1/clubs", s.AuthMiddleware(s.RequireSysAdmin(http.HandlerFunc(s.handleCreateClub))))
	mux.Handle("GET /api/v1/clubs/{id}", s.AuthMiddleware(http.HandlerFunc(s.handleGetClub)))
	mux.Handle("PUT /api/v1/clubs/{id}", s.AuthMiddleware(s.RequireSysAdmin(http.HandlerFunc(s.handleUpdateClub))))
	mux.Handle("DELETE /api/v1/clubs/{id}", s.AuthMiddleware(s.RequireSysAdmin(http.HandlerFunc(s.handleDeleteClub))))

	// Board of Management
	mux.Handle("GET /api/v1/clubs/{club_id}/board-members", s.AuthMiddleware(s.ClubContextMiddleware(http.HandlerFunc(s.HandleGetBoardMembers))))
	mux.Handle("POST /api/v1/clubs/{club_id}/board-members", s.AuthMiddleware(s.ClubContextMiddleware(s.requireClubAdmin(http.HandlerFunc(s.HandleCreateBoardMember)))))
	mux.Handle("PUT /api/v1/clubs/{club_id}/board-members/{id}", s.AuthMiddleware(s.ClubContextMiddleware(s.requireClubAdmin(http.HandlerFunc(s.HandleUpdateBoardMember)))))
	mux.Handle("DELETE /api/v1/clubs/{club_id}/board-members/{id}", s.AuthMiddleware(s.ClubContextMiddleware(s.requireClubAdmin(http.HandlerFunc(s.HandleDeleteBoardMember)))))

	// Members
	mux.Handle("POST /api/v1/clubs/{club_id}/members/{member_id}/invite", s.AuthMiddleware(http.HandlerFunc(s.handleInviteMember)))
	mux.Handle("POST /api/v1/members", s.AuthMiddleware(s.RequirePermission("members:write", s.handleCreateMember)))
	mux.Handle("GET /api/v1/members", s.AuthMiddleware(s.RequirePermission("members:read", s.handleListMembers)))
	mux.Handle("GET /api/v1/members/statistics", s.AuthMiddleware(s.RequirePermission("members:read", s.handleGetMemberStatistics)))
	mux.Handle("GET /api/v1/members/birthdays", s.AuthMiddleware(s.RequirePermission("members:read", s.handleGetMemberBirthdays)))
	mux.Handle("GET /api/v1/members/birthdays/pdf", s.AuthMiddleware(s.RequirePermission("members:read", s.handleGetMemberBirthdaysPDF)))
	mux.Handle("GET /api/v1/members/anniversaries", s.AuthMiddleware(s.RequirePermission("members:read", s.handleGetMemberAnniversaries)))
	mux.Handle("GET /api/v1/members/anniversaries/pdf", s.AuthMiddleware(s.RequirePermission("members:read", s.handleGetMemberAnniversariesPDF)))
	mux.Handle("GET /api/v1/members/{id}", s.AuthMiddleware(s.RequirePermission("members:read", s.handleGetMember)))
	mux.Handle("PUT /api/v1/members/{id}", s.AuthMiddleware(s.RequirePermission("members:write", s.handleUpdateMember)))
	mux.Handle("DELETE /api/v1/members/{id}", s.AuthMiddleware(s.RequirePermission("members:delete", s.handleDeleteMember)))
	mux.Handle("POST /api/v1/members/import", s.AuthMiddleware(s.RequirePermission("members:write", s.handleImportMembers)))
	mux.Handle("GET /api/v1/members/export", s.AuthMiddleware(s.RequirePermission("members:read", s.handleExportMembersCSV)))

	// Departments
	mux.Handle("POST /api/v1/departments", s.AuthMiddleware(s.RequirePermission("departments:write", s.handleCreateDepartment)))
	mux.Handle("GET /api/v1/departments", s.AuthMiddleware(s.RequirePermission("departments:read", s.handleListDepartments)))
	mux.Handle("GET /api/v1/departments/{id}", s.AuthMiddleware(s.RequirePermission("departments:read", s.handleGetDepartment)))
	mux.Handle("PUT /api/v1/departments/{id}", s.AuthMiddleware(s.RequirePermission("departments:write", s.handleUpdateDepartment)))
	mux.Handle("DELETE /api/v1/departments/{id}", s.AuthMiddleware(s.RequirePermission("departments:delete", s.handleDeleteDepartment)))

	// Finance
	mux.Handle("POST /api/v1/finance/booking-accounts", s.AuthMiddleware(s.RequirePermission("finance:write", s.handleCreateBookingAccount)))
	mux.Handle("GET /api/v1/finance/booking-accounts", s.AuthMiddleware(s.RequirePermission("finance:read", s.handleListBookingAccounts)))

	mux.Handle("POST /api/v1/finance/receipts", s.AuthMiddleware(s.RequirePermission("finance:write", s.handleCreateReceipt)))
	mux.Handle("PUT /api/v1/finance/receipts/{id}", s.AuthMiddleware(s.RequirePermission("finance:write", s.handleUpdateReceipt)))
	mux.Handle("DELETE /api/v1/finance/receipts/{id}", s.AuthMiddleware(s.RequirePermission("finance:delete", s.handleDeleteReceipt)))
	mux.Handle("GET /api/v1/finance/receipts", s.AuthMiddleware(s.RequirePermission("finance:read", s.handleListReceipts)))
	mux.Handle("POST /api/v1/finance/receipts/{id}/book", s.AuthMiddleware(s.RequirePermission("finance:write", s.handleBookReceipt)))

	mux.Handle("GET /api/v1/finance/bookings", s.AuthMiddleware(s.RequirePermission("finance:read", s.handleListBookings)))
	mux.Handle("PUT /api/v1/finance/bookings/{id}", s.AuthMiddleware(s.RequirePermission("finance:write", s.handleUpdateBooking)))

	// Club Banks (Standard & Alias)
	mux.Handle("POST /api/v1/finance/bank-accounts", s.AuthMiddleware(s.RequirePermission("finance:write", s.handleCreateClubBankAccount)))
	mux.Handle("GET /api/v1/finance/bank-accounts", s.AuthMiddleware(s.RequirePermission("finance:read", s.handleListClubBankAccounts)))
	mux.Handle("GET /api/v1/clubs/{club_id}/banks", s.AuthMiddleware(s.RequirePermission("finance:read", s.handleListClubBankAccounts)))

	mux.Handle("GET /api/v1/finance/bank-accounts/{id}", s.AuthMiddleware(s.RequirePermission("finance:read", s.handleGetClubBankAccount)))
	mux.Handle("PUT /api/v1/finance/bank-accounts/{id}", s.AuthMiddleware(s.RequirePermission("finance:write", s.handleUpdateClubBankAccount)))
	mux.Handle("DELETE /api/v1/finance/bank-accounts/{id}", s.AuthMiddleware(s.RequirePermission("finance:write", s.handleDeleteClubBankAccount)))

	mux.Handle("POST /api/v1/finance/fee-mappings", s.AuthMiddleware(s.RequirePermission("finance:write", s.handleCreateFeeAccountMapping)))
	mux.Handle("GET /api/v1/finance/fee-mappings", s.AuthMiddleware(s.RequirePermission("finance:read", s.handleListFeeAccountMappings)))
	mux.Handle("PUT /api/v1/finance/fee-mappings/{feeType}", s.AuthMiddleware(s.RequirePermission("finance:write", s.handleUpdateFeeAccountMapping)))
	mux.Handle("DELETE /api/v1/finance/fee-mappings/{feeType}", s.AuthMiddleware(s.RequirePermission("finance:write", s.handleDeleteFeeAccountMapping)))

	mux.Handle("GET /api/v1/finance/sepa-members", s.AuthMiddleware(s.RequirePermission("finance:read", s.handleGetSEPAMembers)))
	mux.Handle("POST /api/v1/finance/sepa-xml", s.AuthMiddleware(s.RequirePermission("finance:write", s.handleGenerateSEPA)))
	mux.Handle("POST /api/v1/clubs/{club_id}/finance/sepa-xml", s.AuthMiddleware(s.RequirePermission("finance:write", s.handleGenerateSEPA)))
	mux.Handle("GET /api/v1/finance/receipts/{id}/pdf", s.AuthMiddleware(s.RequirePermission("finance:read", s.handleGenerateDonationReceipt)))

	// Finance Statements
	mux.Handle("POST /api/v1/finance/statements", s.AuthMiddleware(s.RequirePermission("finance:write", s.handleCreateFinanceStatement)))
	mux.Handle("GET /api/v1/finance/statements", s.AuthMiddleware(s.RequirePermission("finance:read", s.handleListFinanceStatements)))
	mux.Handle("GET /api/v1/finance/statements/{id}", s.AuthMiddleware(s.RequirePermission("finance:read", s.handleGetFinanceStatement)))
	mux.Handle("GET /api/v1/finance/statements/{id}/pdf", s.AuthMiddleware(s.RequirePermission("finance:read", s.handleGetFinanceStatementPDF)))
	mux.Handle("DELETE /api/v1/finance/statements/{id}", s.AuthMiddleware(s.RequirePermission("finance:write", s.handleDeleteFinanceStatement)))

	mux.Handle("POST /api/v1/finance/import/bookings", s.AuthMiddleware(s.RequirePermission("finance:write", s.handleImportBookings)))
	mux.Handle("GET /api/v1/finance/import/bookings", s.AuthMiddleware(s.RequirePermission("finance:read", s.handleListImportBookings)))
	mux.Handle("PUT /api/v1/finance/import/bookings/{id}", s.AuthMiddleware(s.RequirePermission("finance:write", s.handleUpdateImportBooking)))
	mux.Handle("DELETE /api/v1/finance/import/bookings/{id}", s.AuthMiddleware(s.RequirePermission("finance:write", s.handleDeleteImportBooking)))
	mux.Handle("POST /api/v1/finance/import/bookings/{id}/commit", s.AuthMiddleware(s.RequirePermission("finance:write", s.handleCommitImportBooking)))

	// Calendar
	mux.Handle("POST /api/v1/calendar/events", s.AuthMiddleware(s.RequirePermission("calendar:write", s.handleCreateEvent)))
	mux.Handle("GET /api/v1/calendar/events", s.AuthMiddleware(s.RequirePermission("calendar:read", s.handleListEvents)))
	mux.Handle("GET /api/v1/calendar/events/pdf", s.AuthMiddleware(s.RequirePermission("calendar:read", s.handleExportEventsPDF)))
	mux.Handle("PUT /api/v1/calendar/events/{id}", s.AuthMiddleware(s.RequirePermission("calendar:write", s.handleUpdateEvent)))
	mux.Handle("DELETE /api/v1/calendar/events/{id}", s.AuthMiddleware(s.RequirePermission("calendar:delete", s.handleDeleteEvent)))

	// Document Categories
	mux.Handle("POST /api/v1/document-categories", s.AuthMiddleware(s.RequirePermission("documents:write", s.handleCreateDocumentCategory)))
	mux.Handle("GET /api/v1/document-categories", s.AuthMiddleware(s.RequirePermission("documents:read", s.handleListDocumentCategories)))
	mux.Handle("PUT /api/v1/document-categories/{id}", s.AuthMiddleware(s.RequirePermission("documents:write", s.handleUpdateDocumentCategory)))
	mux.Handle("DELETE /api/v1/document-categories/{id}", s.AuthMiddleware(s.RequirePermission("documents:delete", s.handleDeleteDocumentCategory)))

	// Documents
	mux.Handle("POST /api/v1/documents", s.AuthMiddleware(s.RequirePermission("documents:write", s.handleUploadDocument)))
	mux.Handle("GET /api/v1/documents", s.AuthMiddleware(s.RequirePermission("documents:read", s.handleListDocuments)))
	mux.Handle("PUT /api/v1/documents/{id}", s.AuthMiddleware(s.RequirePermission("documents:write", s.handleUpdateDocument)))
	mux.Handle("GET /api/v1/documents/{id}/download", s.AuthMiddleware(s.RequirePermission("documents:read", s.handleDownloadDocument)))
	mux.Handle("DELETE /api/v1/documents/{id}", s.AuthMiddleware(s.RequirePermission("documents:delete", s.handleDeleteDocument)))

	// Users & Roles (System Administrator only)
	mux.Handle("GET /api/v1/roles", s.AuthMiddleware(http.HandlerFunc(s.handleListRoles)))
	mux.Handle("POST /api/v1/users/roles", s.AuthMiddleware(http.HandlerFunc(s.handleAssignRole)))
	mux.Handle("DELETE /api/v1/users/roles", s.AuthMiddleware(http.HandlerFunc(s.handleRemoveRole)))
	mux.Handle("GET /api/v1/users", s.AuthMiddleware(s.RequireSysAdmin(http.HandlerFunc(s.handleListUsers))))
	mux.Handle("GET /api/v1/users/{id}", s.AuthMiddleware(http.HandlerFunc(s.handleGetUser)))
	mux.Handle("PUT /api/v1/users/{id}", s.AuthMiddleware(http.HandlerFunc(s.handleUpdateUser)))
	mux.Handle("POST /api/v1/users/{id}/reset-password", s.AuthMiddleware(http.HandlerFunc(s.handleAdminResetPassword)))

	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func parseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}

func dateToPgDate(t time.Time) pgtype.Date {
	return pgtype.Date{
		Time:  t,
		Valid: true,
	}
}
