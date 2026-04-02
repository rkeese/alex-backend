# Club parameter configuration
As next I want to implement the interface for the club settings.
We have already respect the club settings within database tables but currently
it is not possible to setup the club settings from rest api point of view.
Please can you implement the neccessary REST endpoints to get access to the club
parameter after that please update the API.md rest interface document.
We have following database tables:
* club_bank_accounts
* clubs (this table has initally settings which you must takeover)
* departments

Frontend:
Can you implement the club parameter form for the available club parameter?
In the left hand sidebar should add a menue item which link the user to the form.
Following you find the description from backend implementation.

I have implemented the necessary REST endpoints for managing club settings, specifically for `club_bank_accounts`, and verified the existing endpoints for `clubs` and `departments`.

Here is a summary of the changes:

1.  **Database Queries (finance.sql)**:
    *   Added SQL queries for `GetClubBankAccountByID`, `UpdateClubBankAccount`, and `DeleteClubBankAccount` to enable full CRUD operations for club bank accounts.
    *   Regenerated Go code using `sqlc generate`.

2.  **API Handlers**:
    *   **Finance (finance_handler.go)**: Implemented `handleGetClubBankAccount`, `handleUpdateClubBankAccount`, and `handleDeleteClubBankAccount`.
    *   **Departments (departments_handler.go)**: Added `handleGetDepartment` to allow fetching a single department by ID.

3.  **Router (api.go)**:
    *   Registered the new routes:
        *   `GET /api/v1/finance/bank-accounts/{id}`
        *   `PUT /api/v1/finance/bank-accounts/{id}`
        *   `DELETE /api/v1/finance/bank-accounts/{id}`
        *   `GET /api/v1/departments/{id}`

4.  **Documentation (API.md)**:
    *   Updated the REST interface documentation to include the new endpoints for departments and club bank accounts.

You can now manage club settings (Clubs, Departments, Bank Accounts) fully via the REST API.



### Frontent implementatio:

I have implemented the Club Parameters form and the necessary frontend logic to manage Club Settings, Departments, and Bank Accounts.

### Summary of Changes

1.  **Created `ClubSettingsView.vue`**:
    *   Added a new view at ClubSettingsView.vue.
    *   Implements a **Tabbed Interface**:
        *   **Allgemein (General)**: Form to edit club details (Name, Address, Tax Office info, etc.).
        *   **Abteilungen (Departments)**: List with Add/Edit/Delete functionality for departments.
        *   **Bankkonten (Bank Accounts)**: List with Add/Edit/Delete functionality for bank accounts, including setting a default account.
    *   Uses PrimeVue components (`TabView`, `DataTable`, `Dialog`, `InputText`, `Checkbox`) for a consistent UI.

2.  **Updated api.ts**:
    *   Added missing methods for **Bank Accounts**: `getBankAccount` (by ID), `updateBankAccount`, `deleteBankAccount`.
    *   Added missing method for **Departments**: `getDepartment` (by ID).
    *   Ensured existing create/list methods are present.

3.  **Updated Router (index.ts)**:
    *   Added the route `/admin/club-settings` pointing to the new `ClubSettingsView`.
    *   Protected it with `club:write` permission.

4.  **Updated Sidebar (AppSidebar.vue)**:
    *   Added a **"Vereinsparameter"** (Club Parameters) menu item under the "Verwaltung" (Administration) section.
    *   It is visible if the user has `club:write` permission.

You can now access the club settings via the sidebar, edit the club's master data, and manage departments and bank accounts fully from the frontend.

Made changes.