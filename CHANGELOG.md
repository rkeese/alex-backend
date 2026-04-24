# Changelog

## [Unreleased] - 2026-04-01

### Security
- **Brute-Force-Schutz implementiert:**
  - **Rate Limiting:** Auth-Endpoints (`/api/v1/auth/login`, `/api/v1/auth/register`) sind auf max. 10 Requests pro Minute pro IP limitiert. Bei Überschreitung wird HTTP 429 mit `Retry-After`-Header zurückgegeben.
    - `internal/api/ratelimit.go` – neuer IP-basierter Rate Limiter (Sliding Window)
    - `internal/api/api.go` – Auth-Routen mit Rate-Limiter-Middleware gewrappt
  - **Fehlversuch-Tracking:** Fehlgeschlagene Login-Versuche werden in der Datenbank gezählt (`failed_login_attempts`, `locked_until` Spalten).
    - `migrations/021_brute_force_protection.sql` – neue Spalten in `users`-Tabelle
    - `queries/users.sql` – neue Queries `IncrementFailedLoginAttempts`, `ResetFailedLoginAttempts`
  - **Automatische Account-Sperre:** Nach 5 fehlgeschlagenen Login-Versuchen wird der Account für 15 Minuten gesperrt. HTTP 429 mit `Retry-After`-Header und Fehlermeldung.
    - `internal/api/auth_handler.go` – `handleLogin()` prüft Lockout, zählt Fehlversuche, setzt bei Erfolg zurück
  - **Security-Logging:** Alle fehlgeschlagenen Login-Versuche werden mit E-Mail, IP-Adresse, Versuchsnummer und Lockout-Status geloggt.
    - `internal/api/auth_handler.go` – `log.Printf()` mit `SECURITY:`-Prefix
  - **Admin-Entsperrung:** Wenn ein Admin einen User entblockt (`is_blocked = false`), wird auch die Brute-Force-Sperre zurückgesetzt.
    - `internal/api/users_handler.go` – `handleUpdateUser()` ruft `ResetFailedLoginAttempts` auf
  - **ListUsers erweitert:** Admin-Endpoint `/api/v1/users` liefert nun auch `failed_login_attempts` und `locked_until` Felder.

### Added
- **Kontaktliste als PDF-Export:** Neuer Endpoint `GET /api/v1/members/contact-list/pdf` zum Download einer Kontaktliste aller Mitglieder als PDF (Querformat A4). Enthält Mitgliedsnummer, Name, Adresse, Telefon, Mobil und E-Mail.
  - `internal/pdf/contact_list.go` – PDF-Generator für die Kontaktliste
  - `internal/api/members_handler.go` – `handleExportMembersContactPDF()`
  - `internal/api/api.go` – Route registriert

### Fixed
- **Kontaktliste PDF: Mobilnummer wurde nicht angezeigt.** Das Frontend verwendet `phone2` für Mobilnummern, aber der PDF-Export las aus dem unbenutzten Feld `mobile`. Der PDF-Export liest nun `phone2` als Mobilnummer, sodass die im Frontend eingegebenen Mobilnummern korrekt im PDF erscheinen.
  - `internal/api/members_handler.go` – `handleExportMembersContactPDF()`: `m.Phone2.String` statt `m.Mobile.String`
- **Kontaktliste PDF: Textfelder-Encoding korrigiert.** Alle Textfelder werden nun korrekt über den Unicode-Translator (`tr()`) von gofpdf kodiert.
  - `internal/pdf/contact_list.go` – alle `CellFormat`-Aufrufe verwenden nun `tr()`
- **Login ist nicht mehr case-sensitiv:** E-Mail-Adressen werden bei Login, Registrierung, Einladung und Vorstandserstellung auf Kleinbuchstaben normalisiert (`strings.ToLower`). Damit funktioniert der Login jetzt unabhängig von Groß-/Kleinschreibung.
  - `internal/api/auth_handler.go` – `handleLogin()` und `handleRegister()`
  - `internal/api/invite_handler.go` – `handleInviteMember()`
  - `internal/api/board_handler.go` – `HandleCreateBoardMember()`
