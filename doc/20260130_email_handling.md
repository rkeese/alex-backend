##### Email handling in the backend

I have analyzed the backend implementation and updated it to support sending real emails.

### What was going on in the backend?
Previously, the backend **Mocked** the email sending process. The invite_handler.go file contained code that simply printed the email content (including the generated password) to the server logs (stdout) instead of actually sending an email via SMTP.

### What I have implemented
I have replaced the mock logic with a real SMTP email sender.

1.  **Email Service**: Created a new package email with an SMTP sender implementation.
2.  **Configuration**:
    *   Updated config.go to support SMTP settings.
    *   Updated config.json with new fields (`smtp_host`, `smtp_port`, `smtp_username`, etc.).
3.  **Server Logic**:
    *   Updated api.go and main.go to initialize and inject the email sender into the API server.
    *   Updated invite_handler.go to use this service to send actual emails.
        *   **New User**: Sends "Welcome" email with the generated temporary password.
        *   **Existing User**: Sends "Access Granted" email.

### Action Required
The config.json file now contains **placeholder values** for the SMTP server. You must update this file with your actual SMTP credentials for emails to be sent successfully.

```json
    "smtp_host": "smtp.example.com",
    "smtp_port": "587",
    "smtp_username": "user",
    "smtp_password": "password",
    "smtp_from": "noreply@example.com"
```

The backend is now fully capable of sending invites as requested.