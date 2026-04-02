package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rkeese/alex-backend/internal/database"
	"github.com/stretchr/testify/assert"
)

func TestHandleHealth(t *testing.T) {
	// We can pass nil for queries since handleHealth doesn't use it
	server := NewServer(nil, nil)

	req, err := http.NewRequest("GET", "/health", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.handleHealth)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "OK", rr.Body.String())
}

func TestRoutes(t *testing.T) {
	server := NewServer(&database.Queries{}, nil)
	handler := server.Routes()
	assert.NotNil(t, handler)
}
