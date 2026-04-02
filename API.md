# REST API Documentation

## Authentication

### Register
*   **URL:** `/api/v1/auth/register`
*   **Method:** `POST`
*   **Request Body:**
    ```json
    {
        "email": "user@example.com",
        "password": "password"
    }
    ```
*   **Response:**
    ```json
    {
        "token": "jwt_token_string"
    }
    ```

### Login
*   **URL:** `/api/v1/auth/login`
*   **Method:** `POST`
*   **Request Body:**
    ```json
    {
        "email": "user@example.com",
        "password": "password"
    }
    ```
*   **Response:**
    ```json
    {
        "token": "jwt_token_string"
    }
    ```
*   **Error Responses:**
    *   `401 Unauthorized`: Invalid credentials.
    *   `403 Forbidden`: Account is blocked.
    *   `429 Too Many Requests`: Account temporarily locked after 5 failed login attempts (15 min) or IP rate limit exceeded (max 10 requests/min). Response includes `Retry-After` header (seconds).

> **Brute-Force-Schutz:**
> - **Rate Limiting:** Auth-Endpoints (`/register`, `/login`) sind auf max. 10 Requests pro Minute pro IP limitiert.
> - **Account-Lockout:** Nach 5 fehlgeschlagenen Login-Versuchen wird der Account automatisch für 15 Minuten gesperrt.
> - Bei temporärer Sperre enthält die Response den Header `Retry-After: <Sekunden>` und eine entsprechende Fehlermeldung.
> - Ein erfolgreicher Login setzt den Fehlversuchszähler zurück.
> - Ein Admin kann die Sperre über `PUT /api/v1/users/{id}` mit `{"is_blocked": false}` aufheben.

## User Management

### List Users
*   **URL:** `/api/v1/users`
*   **Method:** `GET`
*   **Headers:** `Authorization: Bearer <token>`
*   **Description:** Returns a list of all registered users (id, email). Useful for Admins to find User IDs for role assignment.
*   **Response:**
    ```json
    [
        {
            "id": "uuid",
            "email": "user@example.com",
            "is_blocked": false,
            "failed_login_attempts": 0,
            "locked_until": null,
            "created_at": "...",
            "updated_at": "..."
        },
        ...
    ]
    ```

### Get User Details
*   **URL:** `/api/v1/users/{id}`
*   **Method:** `GET`
*   **Headers:** `Authorization: Bearer <token>`
*   **Description:** Returns detailed user information, INCLUDING the list of assigned roles. Visible to System Admins, the user themselves, or Club Admins (filtered to their club's roles).
*   **Response:**
    ```json
    {
        "id": "uuid", // The User's ID
        "email": "user@example.com",
        "is_blocked": false,
        "roles": [
             { "role": "members_writer", "club_id": "uuid-of-club" },
             { "role": "admin", "club_id": "uuid-of-club" }
        ]
    }
    ```
*   **Error Responses:**
    *   `400 Bad Request`: Invalid User ID format.
    *   `403 Forbidden`: Insufficient permissions.
    *   `404 Not Found`: User not found.

### Update User
*   **URL:** `/api/v1/users/{id}`
*   **Method:** `PUT`
*   **Headers:** `Authorization: Bearer <token>`
*   **Description:** Updates user details such as password or status flags. Users can update themselves; only SysAdmins or Club Admins (for un/blocking) can update others or change the `is_blocked` status. Setting `is_blocked` to `false` also resets the brute-force lockout (failed attempts counter and `locked_until`).
*   **Request Body:**
    ```json
    {
        "password": "newPassword123", // Optional
        "must_change_password": false, // Optional
        "is_blocked": false // Optional (SysAdmin or Club Admin only)
    }
    ```
*   **Response:** `204 No Content`
*   **Error Responses:**
    *   `400 Bad Request`: Invalid ID or Body.
    *   `401 Unauthorized`: Not logged in.
    *   `403 Forbidden`: Trying to update another user without SysAdmin or Club Admin rights.
    *   `404 Not Found`: User not found.

### Admin Reset Password
*   **URL:** `/api/v1/users/{id}/reset-password`
*   **Method:** `POST`
*   **Headers:** `Authorization: Bearer <token>`
*   **Description:** Resets the password for a user to a randomly generated value. The user will be required to change their password on next login (`must_change_password` is set to `true`). Only accessible by SysAdmins or Club Admins.
*   **Request Body:** None
*   **Response:**
    ```json
    {
        "password": "a1b2c3d4e5f6g7h8i9j0"
    }
    ```
    The generated password is returned once. The admin should communicate it to the user securely.
*   **Error Responses:**
    *   `400 Bad Request`: Invalid User ID format.
    *   `401 Unauthorized`: Not logged in.
    *   `403 Forbidden`: Requires SysAdmin or Club Admin role.
    *   `404 Not Found`: User not found.

### List Available Roles
*   **URL:** `/api/v1/roles`
*   **Method:** `GET`
*   **Headers:** `Authorization: Bearer <token>`
*   **Description:** Returns a list of all role definitions available in the system (e.g., 'members_viewer', 'finance_manager'). Accessible to all authenticated users.
*   **Response:**
    ```json
    [
        {
            "id": "uuid",
            "name": "members_viewer",
            "created_at": "..."
        },
        ...
    ]
    ```

### Assign Role to User
*   **URL:** `/api/v1/users/roles`
*   **Method:** `POST`
*   **Headers:** `Authorization: Bearer <token>`
*   **Description:** Assigns a specific role to a user for a specific club. Requires System Administrator rights OR Club Administrator rights for the target club.
*   **Request Body:**
    ```json
    {
        "user_id": "uuid-of-user",
        "role_name": "finance_writer",
        "club_id": "uuid-of-club"
    }
    ```
*   **Response:** `200 OK`
*   **Error Responses:**
    *   `400 Bad Request`: Invalid User ID or Club ID.
    *   `403 Forbidden`: Insufficient permissions.
    *   `404 Not Found`: Role name not found.
    *   `409 Conflict`: Role already assigned to user.
    *   `500 Internal Server Error`: Database error.

### Remove Role from User
*   **URL:** `/api/v1/users/roles`
*   **Method:** `DELETE`
*   **Headers:** `Authorization: Bearer <token>`
*   **Description:** Removes a specific role from a user. Requires System Administrator rights OR Club Administrator rights for the target club.
*   **Query Parameters:**
    *   `user_id`: UUID of the user
    *   `role_name`: Name of the role (e.g. 'finance_writer')
    *   `club_id`: (Optional) UUID of the club context
*   **Response:** `200 OK` or `204 No Content`

## Clubs

### List Clubs
*   **URL:** `/api/v1/clubs`
*   **Method:** `GET`
*   **Headers:** `Authorization: Bearer <token>`
*   **Response:** Array of Club objects.

### Create Club
*   **URL:** `/api/v1/clubs`
*   **Method:** `POST`
*   **Headers:** `Authorization: Bearer <token>`
*   **Description:** Creates a new club. The user creating the club is automatically assigned the `admin` role for this club. Many fields are optional to allow quick registration.
*   **Request Body:**
    ```json
    {
        "registered_association": true,
        "name": "Club Name",
        "type": "sport_club", // Allowed: sport_club, music_club, social_club, environment_club, cultural_club, hobby_club, rescue_service
        "category": "Category", // Optional
        "number": "12345", // Optional
        "street_house_number": "Street 1", // Optional
        "postal_code": "12345", // Optional
        "city": "City", // Optional
        "name_extension": "e.V.", // Optional
        "address_extension": "c/o Chairman", // Optional
        "tax_office_name": "Finanzamt", // Optional
        "tax_office_tax_number": "123/456", // Optional
        "tax_office_assessment_period": "2024", // Optional
        "tax_office_purpose": "Charity", // Optional
        "tax_office_decision_date": "2024-01-01", // Optional
        "tax_office_decision_type": "Exemption" // Optional
    }
    ```
*   **Response:** Created Club object.
*   **Error Responses:**
    *   `400 Bad Request`: Invalid club type or format.
    *   `500 Internal Server Error`: Server error.

### Get Club
*   **URL:** `/api/v1/clubs/{id}`
*   **Method:** `GET`
*   **Headers:** `Authorization: Bearer <token>`
*   **Response:** Club object.

### Update Club
*   **URL:** `/api/v1/clubs/{id}`
*   **Method:** `PUT`
*   **Headers:** `Authorization: Bearer <token>`
*   **Request Body:** Same as Create Club.
*   **Response:** Updated Club object.

### Delete Club
*   **URL:** `/api/v1/clubs/{id}`
*   **Method:** `DELETE`
*   **Headers:** `Authorization: Bearer <token>`
*   **Description:** Deletes a club and all its associated data (members, biology, finance, etc.). This operation is irreversible.
*   **Response:** `204 No Content`
*   **Error Responses:**
    *   `400 Bad Request`: Invalid Club ID.
    *   `500 Internal Server Error`: Server error.

## Board of Management

### List Board Members
*   **URL:** `/api/v1/clubs/{club_id}/board-members`
*   **Method:** `GET`
*   **Headers:** `Authorization: Bearer <token>`
*   **Response:**
    ```json
    [
        {
            "id": "uuid",
            "club_id": "uuid",
            "member_id": "uuid",
            "user_id": "uuid",
            "position": "Chairman",
            "first_name": "John",
            "last_name": "Doe",
            "member_number": "M123",
            "email": "john@example.com"
        },
        ...
    ]
    ```

### Add Board Member
*   **URL:** `/api/v1/clubs/{club_id}/board-members`
*   **Method:** `POST`
*   **Headers:** `Authorization: Bearer <token>`
*   **Description:** Adds a member to the board. If the member is not a user, a user is created (pwd: Start123!, must change on login).
*   **Request Body:**
    ```json
    {
        "member_id": "uuid-of-member",
        "task": "Treasurer",
        "roles": ["finance_manager", "members_viewer"]
    }
    ```
*   **Response:** `201 Created` with BoardMember object.
*   **Error Responses:**
    *   `400 Bad Request`: Member not found/invalid.
    *   `403 Forbidden`: Admin only.
    *   `409 Conflict`: Already on board.

### Update Board Member
*   **URL:** `/api/v1/clubs/{club_id}/board-members/{id}`
*   **Method:** `PUT`
*   **Headers:** `Authorization: Bearer <token>`
*   **Description:** Updates board member task and roles.
*   **Request Body:**
    ```json
    {
        "task": "New Position",
        "roles": ["admin"]
    }
    ```
*   **Response:** `200 OK`
*   **Error Responses:**
    *   `400 Bad Request`: Invalid ID or Body.
    *   `403 Forbidden`: Admin only.
    *   `404 Not Found`: Board member not found.
    *   `500 Internal Server Error`: Server error.

### Remove Board Member
*   **URL:** `/api/v1/clubs/{club_id}/board-members/{id}`
*   **Method:** `DELETE`
*   **Headers:** `Authorization: Bearer <token>`
*   **Description:** Removes member from board (does not delete user/member).
*   **Response:** `204 No Content`
*   **Error Responses:**
    *   `400 Bad Request`: Invalid ID.
    *   `403 Forbidden`: Admin only.
    *   `404 Not Found`: Board member not found.

## Members

### Invite Member / Grant Access
*   **URL:** `/api/v1/clubs/{club_id}/members/{member_id}/invite`
*   **Method:** `POST`
*   **Headers:**
    *   `Authorization: Bearer <token>`
*   **Description:** Invites an existing club member to becoming a system user. 
    1.  **If the member's email is NOT yet registered as a User:**
        *   A new User account is created.
        *   A secure random password is generated.
        *   The User is linked to the Member.
        *   The User is assigned the `new_user` role (or a default member role).
        *   **Action:** An email MUST be sent to the member's email address containing:
            *   A welcome message.
            *   The generated initial password.
            *   A link to the application login page.
            *   Instructions to change the password after the first login.
    2.  **If a User with this email ALREADY exists:**
        *   The existing User is linked to the Member (if not already).
        *   The User is assigned the relevant role for this club.
        *   **Action:** An email MUST be sent to the user informing them that they have been granted access to the club management system for this club.
*   **Response:**
    ```json
    {
        "status": "invited",
        "email_sent": true
    }
    ```
*   **Error Responses:**
    *   `400 Bad Request`: Member has no email or IDs are invalid.
    *   `403 Forbidden`: User is not an admin.
    *   `404 Not Found`: Member not found.
    *   `409 Conflict`: Member is already linked to a user.

### List Members
*   **URL:** `/api/v1/members`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Response:** Array of Member objects.

### Create Member
*   **URL:** `/api/v1/members`
*   **Method:** `POST`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Request Body:**
    ```json
    {
        "member_number": "M001",
        "first_name": "John",
        "last_name": "Doe",
        "birth_date": "1990-01-01",
        "gender": "m", // m, f, d
        "street_house_number": "Street 1",
        "postal_code": "12345",
        "city": "City",
        "honorary": false,
        "status": "active", // active, passive, honorary, inactive
        "salutation": "mr", // mr, ms, div, company
        "marital_status": "single", // single, married, divorced, widowed
        "letter_salutation": "Dear John",
        "phone1": "123456",
        "email": "john@example.com",
        "joined_at": "2020-01-01",
        "iban": "DE1234...", 
        "account_holder": "John Doe", 
        "sepa_mandate_granted": true, 
        "mandate_reference": "REF123", 
        "mandate_granted_at": "2020-01-01", 
        "payment_method": "sepa", 
        "fee_label": "Standard", 
        "fee_type": "contribution", 
        "fee_assignment": "1_ideel", // Accounting Classification (1_ideel, 2_vermoegen, 3_zweckbetrieb, 4_wirtschaft)
        "creditor_account_id": "uuid-of-club-bank-account", // Optional: Specific Club Bank Account for money collection (Fee Level Override)
        "assigned_club_bank_id": "uuid-of-club-bank-account", // Optional: Default Club Bank Account for this member (Member Level)
        "fee_amount": 120.00, 
        "fee_period": "yearly", 
        "fee_maturity": "2026-03-01", 
        "fee_starts_at": "2026-01-01" 
    }
    ```
*   **Response:** Created Member object.

### Get Member
*   **URL:** `/api/v1/members/{id}`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Response:**
    ```json
    {
        "id": "uuid",
        "member_number": "M001",
        "first_name": "John",
        "last_name": "Doe",
        "birth_date": "1990-01-01",
        "gender": "m",
        "street_house_number": "Street 1",
        "postal_code": "12345",
        "city": "City",
        "honorary": false,
        "status": "active",
        "salutation": "mr",
        "marital_status": "single",
        "letter_salutation": "Dear John",
        "phone1": "123456",
        "email": "john@example.com",
        "joined_at": "2020-01-01",
        "member_until": "2025-12-31",
        "note": "Some note",
        "title": "Dr.",
        "iban": "DE1234...",
        "account_holder": "John Doe",
        "sepa_mandate_granted": true,
        "mandate_reference": "REF123",
        "mandate_granted_at": "2020-01-01",
        "payment_method": "sepa",
        "creditor_account_id": "uuid...",
        "assigned_club_bank_id": "uuid...",
        "fee_assignment": "1_ideel"
    }
    ```

### Update Member
*   **URL:** `/api/v1/members/{id}`
*   **Method:** `PUT`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Request Body:** Same as Create Member.
*   **Response:** Updated Member object.

### Delete Member
*   **URL:** `/api/v1/members/{id}`
*   **Method:** `DELETE`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Response:** 204 No Content (or success message).

### Get Member Statistics
*   **URL:** `/api/v1/members/statistics`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Query Parameters:**
    *   `year` (required): The year for which to generate statistics (e.g., `2024`).
*   **Description:** Returns member statistics grouped by birth year, including counts for male, female, divers, and total, for members active in the specified year.
*   **Response:**
    ```json
    [
        {
            "birth_year": 1990,
            "count_m": 5,
            "count_f": 3,
            "count_d": 0,
            "count_total": 8
        },
        ...
    ]
    ```

### Get Birthday List
*   **URL:** `/api/v1/members/birthdays`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Query Parameters:**
    *   `year` (optional): The target year (default: current year).
    *   `milestones` (optional): Comma-separated list of ages (default: "50,60,70,80,90,100").
*   **Description:** Returns a list of members celebrating specific milestone birthdays in the given year, sorted by date.
*   **Response:**
    ```json
    [
        {
            "first_name": "Max",
            "last_name": "Mustermann",
            "birth_date": "1974-05-20",
            "date": "2024-05-20",
            "new_age": 50
        }
    ]
    ```

### Get Birthday List (PDF)
*   **URL:** `/api/v1/members/birthdays/pdf`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Query Parameters:**
    *   `year` (optional): The target year (default: current year).
    *   `milestones` (optional): Comma-separated list of ages (default: "50,60,70,80,90,100").
*   **Description:** Downloads a PDF document listing members celebrating specific milestone birthdays.
*   **Response:** Binary PDF file.

### Get Anniversary List
*   **URL:** `/api/v1/members/anniversaries`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Query Parameters:**
    *   `year`: (Optional) The target year (default: current year).
    *   `years`: (Optional) Comma-separated list of anniversary years to include (default: "25,30,40,50,60").
*   **Description:** Returns a list of members celebrating specific anniversaries (based on joined_at) in the given year, sorted by date.
*   **Response:**
    ```json
    [
        {
            "FirstName": "John",
            "LastName": "Doe",
            "JoinedAt": "2000-01-01",
            "AnniversaryDate": "2025-01-01",
            "MembershipYears": 25
        }
    ]
    ```

### Get Anniversary List (PDF)
*   **URL:** `/api/v1/members/anniversaries/pdf`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Query Parameters:**
    *   `year`: (Optional) The target year (default: current year).
    *   `years`: (Optional) Comma-separated list of anniversary years to include (default: "25,30,40,50,60").
*   **Description:** Downloads a PDF document listing members celebrating specific anniversaries.
*   **Response:** Binary PDF file.

### Export Members CSV
*   **URL:** `/api/v1/members/export`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Description:** Downloads a CSV file containing complete member data including personal details, bank account, and fee information.
*   **Response:** CSV file download (`text/csv`).

## Departments

### List Departments
*   **URL:** `/api/v1/departments`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Response:** Array of Department objects.

### Get Department
*   **URL:** `/api/v1/departments/{id}`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Response:** Department object.

### Create Department
*   **URL:** `/api/v1/departments`
*   **Method:** `POST`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Request Body:**
    ```json
    {
        "name": "Football",
        "subdivision": "Youth",
        "parent_id": "optional-uuid"
    }
    ```
*   **Response:** Created Department object.

### Update Department
*   **URL:** `/api/v1/departments/{id}`
*   **Method:** `PUT`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Request Body:** Same as Create Department.
*   **Response:** Updated Department object.

### Delete Department
*   **URL:** `/api/v1/departments/{id}`
*   **Method:** `DELETE`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Response:** 204 No Content.

## Finance

### List Booking Accounts
*   **URL:** `/api/v1/finance/booking-accounts`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Response:** Array of Booking Account objects.
    ```json
    [
        {
            "id": "uuid",
            "club_id": "uuid",
            "majority_list": "1_ideel",
            "majority_list_description": "Ideeller Bereich",
            "minority_list": "List B",
            "created_at": "timestamp",
            "updated_at": "timestamp"
        }
    ]
    ```

### Create Booking Account
*   **URL:** `/api/v1/finance/booking-accounts`
*   **Method:** `POST`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Request Body:**
    ```json
    {
        "majority_list": "1_ideel",
        "minority_list": "List B"
    }
    ```
*   **Response:** Created Booking Account object.
    ```json
    {
        "id": "uuid",
        "club_id": "uuid",
        "majority_list": "1_ideel",
        "majority_list_description": "Ideeller Bereich",
        "minority_list": "List B",
        "created_at": "timestamp",
        "updated_at": "timestamp"
    }
    ```

### List Receipts
*   **URL:** `/api/v1/finance/receipts`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Response:** Array of Receipt objects.

### Create Receipt
*   **URL:** `/api/v1/finance/receipts`
*   **Method:** `POST`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Request Body:**
    ```json
    {
        "type": "income",
        "recipient": "Customer Name",
        "number": "INV-2026-001",
        "date": "2026-01-01",
        "position_assignment": "4_wirtschaft",
        "amount": 119.00,
        "is_booked": false,
        "note": "Services",
        "donor_id": null,
        "seller_name": "My Club",
        "seller_address": "Club Street 1, 12345 City",
        "buyer_name": "Customer Name",
        "buyer_address": "Customer Street 1, 54321 City",
        "seller_tax_id": "123/456/789",
        "seller_vat_id": "DE123456789",
        "delivery_date": "2026-01-01",
        "total_vat_amount": 19.00,
        "invoice_items": [
            {
               "description": "Consulting",
               "quantity": 1,
               "net_amount": 100.00,
               "tax_rate": 19.0,
               "vat_amount": 19.00,
               "gross_amount": 119.00
            }
        ]
    }
    ```
*   **Response:** Created Receipt object.

### Update Receipt
*   **URL:** `/api/v1/finance/receipts/{id}`
*   **Method:** `PUT`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Request Body:** Same as Create Receipt.
*   **Response:** Updated Receipt object.

### Delete Receipt
*   **URL:** `/api/v1/finance/receipts/{id}`
*   **Method:** `DELETE`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Response:** 204 No Content.

### Book Receipt (Transfer to Bookings)
*   **URL:** `/api/v1/finance/receipts/{id}/book`
*   **Method:** `POST`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Description:** Creates a new booking entry in the bookings table based on the receipt data, and sets the receipt's is_booked status to true.
*   **Response:**
    ```json
    {
        "message": "Receipt transferred to booking successfully"
    }
    ```
*   **Error Responses:**
    *   `400 Bad Request`: If receipt is already booked or invalid ID.
    *   `404 Not Found`: If receipt does not exist.

### List Bookings
*   **URL:** `/api/v1/finance/bookings`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Query Parameters:**
    *   `club_bank_account_id` (optional): Filter bookings by specific Bank Account (UUID).
    *   `start_date` (optional): Filter bookings on or after this date (`YYYY-MM-DD`). Used to calculate `start_amount`.
    *   `end_date` (optional): Filter bookings on or before this date (`YYYY-MM-DD`).
*   **Response:**
    *   Example:
    ```json
    {
        "bookings": [
            {
                "id": "uuid...",
                "booking_date": "2023-12-01",
                "valuta_date": "2023-12-01",
                "client_recipient": "Supplier Inc.",
                "booking_text": "Invoice 123",
                "purpose": "Hardware",
                "amount": -150.00,
                "currency": "EUR",
                "assigned_booking_account_id": "uuid-of-category",
                "club_bank_account_id": "uuid-of-bank-account",
                "payment_participant_iban": "DE1234...",
                "payment_participant_bic": "GENO...",
                "created_at": "..."
            }
        ],
        "start_amount": 1000.00, // Balance of the account BEFORE start_date
        "end_amount": 850.00     // Balance after applying all returned bookings
    }
    ```

### Update Booking (Link to Category)
*   **URL:** `/api/v1/finance/bookings/{id}`
*   **Method:** `PUT`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Description:** Updates the assignment of a booking to an internal booking account (Category).
*   **Request Body:**
    ```json
    {
        "assigned_booking_account_id": "uuid-of-category"
    }
    ```
*   **Response:** Updated Booking object.

### List Club Bank Accounts
*   **URL:** `/api/v1/finance/bank-accounts`
*   **Alternate URL for Dropdowns:** `/api/v1/clubs/{club_id}/banks`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Response:**
    ```json
    [
        {
            "id": "uuid",
            "name": "Sparkasse Checking",
            "account_holder": "My Club e.V.",
            "iban": "DE123...",
            "is_default": true
        },
        ...
    ]
    ```

### Create Club Bank Account
*   **URL:** `/api/v1/finance/bank-accounts`
*   **Method:** `POST`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Request Body:**
    ```json
    {
        "name": "Sparkasse Main",
        "account_holder": "Club e.V.",
        "creditor_id": "DE98ZZZ09999999999",
        "iban": "DE1234567890",
        "bic": "GENODEF1ABC",
        "is_default": true
    }
    ```
*   **Response:** Created object.

### Get Club Bank Account
*   **URL:** `/api/v1/finance/bank-accounts/{id}`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Response:** Bank Account object.

### Update Club Bank Account
*   **URL:** `/api/v1/finance/bank-accounts/{id}`
*   **Method:** `PUT`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Request Body:** Same as Create Club Bank Account.
*   **Response:** Updated Bank Account object.

### Delete Club Bank Account
*   **URL:** `/api/v1/finance/bank-accounts/{id}`
*   **Method:** `DELETE`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Response:** 204 No Content.

### List Fee Account Mappings
*   **URL:** `/api/v1/finance/fee-mappings`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Response:**
    ```json
    [
        {
            "id": "uuid",
            "fee_type": "Full Membership",
            "club_bank_account_id": "uuid",
            "bank_account_name": "Main Account",
            "iban": "DE..."
        },
        ...
    ]
    ```

### Create Fee Account Mapping
*   **URL:** `/api/v1/finance/fee-mappings`
*   **Method:** `POST`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Request Body:**
    ```json
    {
        "fee_type": "Youth Department",
        "club_bank_account_id": "uuid-of-bank-account"
    }
    ```
*   **Response:** Created Mapping object.

### Update Fee Account Mapping
*   **URL:** `/api/v1/finance/fee-mappings/{feeType}`
*   **Method:** `PUT`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Request Body:**
    ```json
    {
        "club_bank_account_id": "uuid-of-new-bank-account"
    }
    ```
*   **Response:** Updated Mapping object.

### Delete Fee Account Mapping
*   **URL:** `/api/v1/finance/fee-mappings/{feeType}`
*   **Method:** `DELETE`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Response:** 204 No Content.

### Get SEPA Due Members
*   **URL:** `/api/v1/finance/sepa-members`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Query Parameters:**
    *   `execution_date`: (Required) YYYY-MM-DD
*   **Response:** `200 OK`
    *   Returns a JSON list of members.
    *   Returns `[]` (empty array) if no due members are found.
    ```json
    [
        {
            "member_id": "uuid",
            "first_name": "John",
            "last_name": "Doe",
            "amount": 50.00,
            "fee_label": "Membership Fee 2023",
            "member_iban": "DE...",
            "member_bic": "GEN...",
            "mandate_reference": "MREF-001",
            "mandate_issued_at": "2022-01-01",
            "sequence_type": "RCUR",
            "target_account_holder": "Club Name",
            "target_iban": "DE..."
        }
    ]
    ```

### Generate SEPA XML / ZIP
*   **URL:** `/api/v1/finance/sepa-xml` OR `/api/v1/clubs/{club_id}/finance/sepa-xml`
*   **Method:** `POST`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Request Body:**
    ```json
    {
        "execution_date": "2023-02-01"
    }
    ```
*   **Response:**
    *   **Content-Type:** `application/zip` (Multi-file batch)
    *   **Content-Disposition:** `attachment; filename=sepa_files.zip`
    *   **Description:** Returns a ZIP archive containing individual XML files for each Club Bank Account involved (grouped by Target Creditor).

### Generate Donation Receipt PDF
*   **URL:** `/api/v1/finance/receipts/{id}/pdf`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Response:** PDF file download (`application/pdf`).

### Import Bank Bookings (CSV)
*   **URL:** `/api/v1/finance/import/bookings`
*   **Method:** `POST`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
    *   `Content-Type: multipart/form-data`
*   **Request Body:**
    *   `file`: The CSV file to upload (Sparkasse or Volksbank format).
*   **Response:** `200 OK`
    ```json
    {
        "message": "Imported 12 bookings to staging area using profile 'Sparkasse'. Errors: 2",
        "count": 12,
        "errors": [
             { "row": 3, "error": "Database error: ERROR: duplicate key value..." },
             { "row": 15, "error": "Parse error: row too short" }
        ]
    }
    ```
*   **Description:** Parses the uploaded CSV file and creates **pending** import entries (`bank_bookings_import`). Returns detailed error information for any rows that failed to import (e.g. duplicates, parsing errors).

### List Pending Imports
*   **URL:** `/api/v1/finance/import/bookings`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Response:** Array of pending import objects.

### Update Pending Import
*   **URL:** `/api/v1/finance/import/bookings/{id}`
*   **Method:** `PUT`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Request Body:**
    ```json
    {
        "club_bank_account_id": "uuid",
        "booking_date": "2023-01-01",
        "valuta_date": "2023-01-01",
        "amount": 10.50,
        "purpose": "Corrected purpose",
        "payment_participant_name": "Name",
        "payment_participant_iban": "DE123...",
        "payment_participant_bic": "GEN...",
        "status": "pending"
    }
    ```
*   **Response:** Updated import object.

### Delete Pending Import (Discard)
*   **URL:** `/api/v1/finance/import/bookings/{id}`
*   **Method:** `DELETE`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Response:** 204 No Content.

### Commit Pending Import (Create Booking)
*   **URL:** `/api/v1/finance/import/bookings/{id}/commit`
*   **Method:** `POST`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Request Body:** Same JSON structure as Update. Used to finalize details before committing.
*   **Description:** Creates a confirmed booking from the import record and removes the import record.
*   **Response:** Created Booking object.

## Calendar

### List Events
*   **URL:** `/api/v1/calendar/events`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Response:** Array of Event objects.

### Export Events PDF
*   **URL:** `/api/v1/calendar/events/pdf`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Query Parameters:**
    *   `year` (optional): The year to filter events (default: current year). Example: `2023`.
    *   `month` (optional): The month to filter events (1-12). If omitted, returns events for the whole year.
*   **Response:** PDF file download (`application/pdf`).

### Create Event
*   **URL:** `/api/v1/calendar/events`
*   **Method:** `POST`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Request Body:**
    ```json
    {
        "date": "2023-05-01",
        "time": "14:00",
        "description": "Annual Meeting"
    }
    ```
*   **Response:** Created Event object.

### Update Event
*   **URL:** `/api/v1/calendar/events/{id}`
*   **Method:** `PUT`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Request Body:** Same as Create Event.
*   **Response:** Updated Event object.

### Delete Event
*   **URL:** `/api/v1/calendar/events/{id}`
*   **Method:** `DELETE`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Response:** 204 No Content.

## Documents

### List Documents
*   **URL:** `/api/v1/documents`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
*   **Response:** Array of Document objects.

### Upload Document
*   **URL:** `/api/v1/documents`
*   **Method:** `POST`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`
    *   `Content-Type: multipart/form-data`
*   **Request Body:** Form data with file field named `file`.
*   **Response:** Created Document object.

### Download Document
*   **URL:** `/api/v1/documents/{id}/download`
*   **Method:** `GET`
*   **Headers:**
    *   `Authorization: Bearer <token>`
    *   `X-Club-ID: <club_id>`

## Finance Statements

### Generate/Create Finance Statement
*   **URL:** `/api/v1/finance/statements`
*   **Method:** `POST`
*   **Permissions:** `finance:write`
*   **Request Body:**
    ```json
    {
        "year": 2025,
        "initial_balances": {
             "uuid-of-bank-account": 1234.56,
             "cash": 50.00
        }
    }
    ```
    *   `initial_balances`: A map where keys are Bank Account UUIDs or the literal string `"cash"`. Values are the starting balances for the year.
    
    **Logic Notes:**
    *   **Cash Account:** If a starting balance for `"cash"` is provided, or if bookings without a valid Bank Account ID are found (e.g. "MasterDoor"), a specific row `"Kasse (Barbestand)"` is added to `bankBalances`.
    *   **Orphaned Bookings:** Bookings with no assigned bank account are automatically aggregated into the Cash account.
    *   **Category Fallback:** Bookings with missing or unknown categories are automatically grouped under `"Sammelposten"` in the `overview` and `details`.

*   **Response:**
    ```json
    {
        "id": "uuid",
        "club_id": "uuid",
        "year": 2025,
        "created_at": "2026-02-12T...",
        "data": {
            "clubName": "My Club",
            "year": 2025,
            "bankBalances": [...],
            "totalBankBalance": {...},
            "overview": [...],
            "totalOverview": {...},
            "details": {
                "Ideeller Bereich": [...]
            }
        }
    }
    ```

### List Finance Statements
*   **URL:** `/api/v1/finance/statements`
*   **Method:** `GET`
*   **Permissions:** `finance:read`
*   **Response:** List of statement summaries (without full data).

### Get Finance Statement
*   **URL:** `/api/v1/finance/statements/{id}`
*   **Method:** `GET`
*   **Permissions:** `finance:read`
*   **Response:** Full statement with data.

### Get Finance Statement PDF
*   **URL:** `/api/v1/finance/statements/{id}/pdf`
*   **Method:** `GET`
*   **Permissions:** `finance:read`
*   **Response:** Binary PDF data (`application/pdf`).

### Delete Finance Statement
*   **URL:** `/api/v1/finance/statements/{id}`
*   **Method:** `DELETE`
*   **Permissions:** `finance:write`
*   **Response:** 204 No Content

*   **Response:** File download (`application/octet-stream`).
