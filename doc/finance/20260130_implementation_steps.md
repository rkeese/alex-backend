# First step

I have completed the implementation of the Finance module core features.

Here is what I have done:

1.  **Requirements Update**: I updated 20260130_requirements.md to reflect our decision on Barges and Receipts.
2.  **SKR49 Standard Accounts**: I created a database migration 013_seed_skr49_accounts.sql that will automatically populate the booking accounts (Ideel, Zweckbetrieb, etc.) with standard SKR49 values when the app starts/migrates. This ensures imports have a valid category to assign to.
3.  **Bank Import Handler**: I implemented the CSV import logic in finance_import_handler.go.
    *   It automatically detects if the file is from **Sparkasse** or **Volksbank** (based on your provided samples).
    *   It parses German date formats (`30.12.25`) and number formats (`1.840,68 +`).
    *   It links bookings to the club's bank accounts if the IBAN matches.
4.  **API Registration**: I registered the new endpoint `POST /api/v1/finance/import/bookings`.
5.  **Documentation**: I updated API.md to include the new import endpoint.

**Next Steps for you:**
*   Restart your backend service to apply the new migration and register the routes.
*   You can now test the import by posting your CSV files to `/api/v1/finance/import/bookings`.

--------------------------------------------------------------

I have implemented the `ListBookings` functionality:

1.  **Backend Handler**: Added `handleListBookings` to finance_handler.go which queries the `bookings` table.
2.  **API Route**: Registered `GET /api/v1/finance/bookings` in api.go.
3.  **Documentation**: Updated API.md to include valid reference for the new endpoint.

The server has been restarted. You can now:
1.  **Import** CSV files via `POST /api/v1/finance/import/bookings`.
2.  **Verify & View** the imported data via `GET /api/v1/finance/bookings`.

This completes the loop of "Importing and Providing" the finance data as requested.

--------------------------------------------------------------

