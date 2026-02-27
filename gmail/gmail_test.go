package gmail

import (
	"bytes"
	"context"
	"html/template"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// To run this test, set the following environment variables:
//
//	GMAIL_CLIENT_ID
//	GMAIL_CLIENT_SECRET
//	GMAIL_REFRESH_TOKEN
//	GMAIL_SENDER_NAME       — display name (e.g. "OneTrust Support")
//	GMAIL_SENDER_EMAIL      — sender address (e.g. "noreply@onetrust.com")
//	GMAIL_TEST_RECIPIENT    — recipient address to send the test email to
//	GMAIL_TEST_TEMPLATE     — absolute path to a template file
//	                          (e.g. templates/email/login-verify-otp.html)
//
// Example:
//
//	go test ./third_party/gmail/... -v -run TestSendEmail
func TestSendEmail(t *testing.T) {
	clientID := os.Getenv("GMAIL_CLIENT_ID")
	clientSecret := os.Getenv("GMAIL_CLIENT_SECRET")
	refreshToken := os.Getenv("GMAIL_REFRESH_TOKEN")
	senderName := os.Getenv("GMAIL_SENDER_NAME")
	senderEmail := os.Getenv("GMAIL_SENDER_EMAIL")
	recipient := os.Getenv("GMAIL_TEST_RECIPIENT")
	templatePath := os.Getenv("GMAIL_TEST_TEMPLATE")

	if clientID == "" || clientSecret == "" || refreshToken == "" {
		t.Skip("skipping: GMAIL_CLIENT_ID, GMAIL_CLIENT_SECRET, GMAIL_REFRESH_TOKEN are required")
	}
	if senderEmail == "" || recipient == "" || templatePath == "" {
		t.Skip("skipping: GMAIL_SENDER_EMAIL, GMAIL_TEST_RECIPIENT, GMAIL_TEST_TEMPLATE are required")
	}

	// Parse template together with layout
	layoutPath := filepath.Join(filepath.Dir(templatePath), "layout.html")
	tmpl, err := template.ParseFiles(templatePath, layoutPath)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]interface{}{
		"OTP":           "482916",
		"ExpireMinutes": 5,
		"Year":          time.Now().Year(),
	})
	if err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}

	sender, err := NewGmailSender(Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RefreshToken: refreshToken,
		SenderName:   senderName,
		SenderEmail:  senderEmail,
	})
	if err != nil {
		t.Fatalf("failed to create sender: %v", err)
	}

	msg, err := NewMessage(recipient, "Test Email from gmail package", buf.String())
	if err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	if err := sender.Send(context.Background(), msg); err != nil {
		t.Fatalf("failed to send email: %v", err)
	}

	t.Logf("email sent successfully to %s", recipient)
}
