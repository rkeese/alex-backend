# Member Import API Documentation

## Import Members from CSV

**Endpoint:** `POST /api/v1/members/import`

**Description:**
Imports a list of members from a CSV file. The file should be uploaded as `multipart/form-data`.

**Headers:**
- `Content-Type`: `multipart/form-data`
- `Authorization`: Bearer `<token>` (Requires "Mitglieder" permission)
- `X-Club-ID`: `<club_id>` (Alternatively, provide `club_id` as a query parameter)

**Parameters:**
- `file`: The CSV file to upload.
- `club_id`: (Optional) Can be used as a query parameter instead of the `X-Club-ID` header (e.g., `/api/v1/members/import?club_id=<id>`).

**CSV Format:**
The CSV file must contain the following headers (German):
- `Mitgliedsnr.` (Required)
- `Vorname` (Required)
- `Nachname` (Required)
- `Anrede`
- `Titel`
- `Telefon`
- `Mobil`
- `E-Mail`
- `Strasse & Hausnr.`
- `PLZ`
- `Ort`
- `Land`
- `Geburtstag` (Format: D.M.YYYY)
- `Mitglied seit` (Format: D.M.YYYY)
- `Ehrenmitglied` (Values: "Ja", "Nein")
- `Status` (e.g., "Aktiv")
- `Mitglied bis` (Format: D.M.YYYY)
- `Beitrag (Bezeichnung)`
- `Beitrag (Typ)`
- `Beitrag (Betrag)` (Format: "12,00")
- `Beitrag (Zeitraum)`
- `Beitrag (Fälligkeit)` (Format: D.M.YYYY)
- `Notizen`
- `Geschlecht`
- `Familienstand`
- `Zahlungsart` (e.g., "Einzug")
- `IBAN`
- `Kontoinhaber`
- `SEPA-Mandat erteilt` (Values: "Einzug", "Ja" or Date)
- `Mandatsreferenz`
- `Art des Mandats`
- `Art der nächsten Lastschrift`
- `Mandat erteilt am` (Format: D.M.YYYY)
- `Letzte Verwendung` (Format: D.M.YYYY)

**Response:**

*   **200 OK**
    ```json
    {
      "success_count": 42,
      "errors": [
        "Line 5: First/Last name missing",
        "Line 10: Invalid BirthDate: parsing time \"...\""
      ]
    }
    ```

*   **400 Bad Request**
    *   Missing file
    *   File too large
    *   Invalid CSV format
    *   Missing required headers

*   **401 Unauthorized**
    *   Invalid or missing token

*   **500 Internal Server Error**
    *   Database error or other server-side issue.
