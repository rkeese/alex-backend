#### Implementation of sepa direct debit

I would like to be able to collect the membership fees via pain.008.001.002 direct debits.
This means that I want the "pain.008.001.002" XML file to be generated on the basis of a corresponding xsd file.
The information from which bank a member's membership fee is to be collected can be found in the database.
It should be possible to trigger the collection of membership fees via the rest endpoints.
Can you implement this functionality and document the extension of the rest endpoints in the api.md file?
In the directory ./doc/sepa there are further documents that describe the requirements for described direct debit procedures and also the xsd files.


I have implemented the SEPA Direct Debit collection functionality using `pain.008.001.02` format.

Here is a summary of the changes:

1.  **Database Query Update**: Modified fees.sql to include the `next_direct_debit_type` (Sequence Type: `FRST` or `RCUR`) from the member's bank account. This is required for correct SEPA grouping.
2.  **SEPA Library Update**: Updated sepa.go to support the `pain.008.001.02` schema fully.
    *   Added `CreditorSchemeId` structs (missing in the original implementation).
    *   Added `SequenceType` support (`SeqTp`) to `PaymentGroup` and `PaymentType`.
    *   Updated `GenerateSEPA` to populate these fields in the XML.
3.  **API Handler Update**: Updated finance_handler.go to group transactions not just by Target IBAN, but also by **Sequence Type** (Recurrent vs First), ensuring `pain.008.001.02` compliance.
4.  **Documentation**: Updated API.md to specify that the `POST /api/v1/finance/sepa-xml` endpoint generates files in `pain.008.001.02` format.

You can now trigger the collection via `POST /api/v1/finance/sepa-xml` with an `execution_date`, and it will return the correct XML file.

Made changes.

--------------------------------------------------------------------------

Most recently, you implemented the "Generate SEPA XML" function. In this context, I would like to display the complete list of members with the contribution and bank information on the frontend. Can you provide me with a corresponding list via the rest interface? Can you please document the extension of the rest endpoints in the api.md document? Furthermore, I would like to be able to export the Sepa Xml direct debit file from the frontend, so that I can transfer it to the bank via a bank app.


I have implemented the changes you requested.

### 1. New REST Endpoint for SEPA Member List
I have added a new REST endpoint `GET /api/v1/finance/sepa-members` to retrieving the list of members with their contribution and bank data, ready for the frontend display.
*   **Endpoint:** `GET /api/v1/finance/sepa-members`
*   **Query Param:** `execution_date` (required)
*   **Returns:** A JSON list of members including IBAN, BIC, Amount, Mandate Reference, and Sequence Type.

### 2. SEPA XML Export
The existing endpoint `POST /api/v1/finance/sepa-xml` is configured to return the generated XML file as a downloadable attachment (`Content-Disposition: attachment; filename=sepa.xml`). You can call this endpoint directly from the frontend to trigger the file download for your banking app.

### 3. Documentation
I have updated API.md to include the documentation for the new `GET` endpoint.

**Summary of Changes:**
*   Modified finance_handler.go: Added `handleGetSEPAMembers` and `SEPAMemberResponse` struct.
*   Modified api.go: Registered the new route.
*   Modified API.md: Documented the new endpoint.

You can now use the `GET` endpoint to show the preview table in the frontend and the `POST` endpoint to download the final XML file.

Made changes.

-----------------------------------------------------------------------

Where is it depend on to get a valid list:

Based on the backend code analysis, the list returned by `api/v1/finance/sepa-members` depends on the SQL query `GetDueMembershipFees` defined in fees.sql.

Here are the specific constraints that must be fulfilled for a record to appear in the list:

1.  **Club Association**: The member must belong to the club identified by the current user's token (`m.club_id = $1`).
2.  **Payment Method**: The fee record in `membership_fees` must have `payment_method = 'sepa_direct_debit'`.
3.  **Maturity Date**: The fee's `maturity_date` must be **on or before** the `execution_date` you provided in the request query parameter (`mf.maturity_date <= $2`).
4.  **Active Fee**: The fee must be currently valid (`mf.ends_at` is `NULL` OR `ids >= execution_date`).
5.  **SEPA Mandate**: The member's bank account must have a valid SEPA mandate (`mba.sepa_mandate_available = TRUE`).

-----------------------------------------------------------------------

