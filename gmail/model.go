package gmail

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Config — value object for Gmail OAuth2 credentials and sender identity
// ---------------------------------------------------------------------------

// Config holds the OAuth2 credentials and sender identity required to
// authenticate with the Gmail API. It is a pure value object with no
// behaviour beyond self-validation.
//
// Fields:
//   - ClientID:     The OAuth2 client ID issued by Google Cloud Console.
//   - ClientSecret: The corresponding OAuth2 client secret.
//   - RefreshToken: A long-lived refresh token used to obtain short-lived
//     access tokens without user interaction.
//   - SenderName:   The human-readable display name that appears in the
//     "From" header (e.g. "OneTrust Support"). Optional — if
//     empty, only the email address is shown.
//   - SenderEmail:  The email address that appears in the "From" header.
//     This must match the Gmail account that owns the refresh
//     token, otherwise the Gmail API will reject the request.
type Config struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	SenderName   string
	SenderEmail  string
}

// Validate checks that all required fields are present and returns the first
// validation error encountered. It uses the sentinel errors defined in
// constants.go so callers can match with errors.Is().
//
// SenderName is intentionally not validated — it is acceptable (though
// unusual) to send email without a display name.
func (c Config) Validate() error {
	if strings.TrimSpace(c.ClientID) == "" {
		return ErrEmptyClientID
	}
	if strings.TrimSpace(c.ClientSecret) == "" {
		return ErrEmptyClientSecret
	}
	if strings.TrimSpace(c.RefreshToken) == "" {
		return ErrEmptyRefreshToken
	}
	if strings.TrimSpace(c.SenderEmail) == "" {
		return ErrEmptySenderEmail
	}
	return nil
}

// ---------------------------------------------------------------------------
// Message — value object representing a single outbound email
// ---------------------------------------------------------------------------

// Message is an immutable value object that represents a single outbound
// email. Immutability is enforced by keeping all fields unexported and
// providing only getter methods. This guarantees that once a Message passes
// validation in NewMessage(), it can never be put into an invalid state.
//
// Design decision — Why unexported fields + getters instead of a plain struct?
//
//	A plain struct with exported fields would allow callers to create a
//	Message{} literal with empty fields, bypassing validation entirely.
//	By forcing construction through NewMessage(), every Message in the
//	system is guaranteed to be valid. The getter methods are trivial and
//	will be inlined by the Go compiler, so there is zero runtime cost.
type Message struct {
	to      string
	subject string
	html    string
}

// NewMessage constructs and validates a Message. It returns an error if any
// required field is blank. Whitespace-only strings are treated as blank.
//
// Parameters:
//   - to:      The recipient email address (e.g. "user@example.com").
//   - subject: The email subject line.
//   - html:    The email body as an HTML string.
//
// Returns:
//   - A valid, immutable Message on success.
//   - A sentinel error (ErrEmptyRecipient, ErrEmptySubject, or ErrEmptyBody)
//     on validation failure, so callers can match with errors.Is().
func NewMessage(to, subject, html string) (Message, error) {
	if strings.TrimSpace(to) == "" {
		return Message{}, ErrEmptyRecipient
	}
	if strings.TrimSpace(subject) == "" {
		return Message{}, ErrEmptySubject
	}
	if strings.TrimSpace(html) == "" {
		return Message{}, ErrEmptyBody
	}
	return Message{to: to, subject: subject, html: html}, nil
}

// To returns the recipient email address.
func (m Message) To() string { return m.to }

// Subject returns the email subject line.
func (m Message) Subject() string { return m.subject }

// HTML returns the email body as an HTML string.
func (m Message) HTML() string { return m.html }

// ---------------------------------------------------------------------------
// RFC 2822 raw message formatting
// ---------------------------------------------------------------------------

// formatRawRFC2822 builds a minimal RFC 2822 message string from the given
// sender identity and Message. The result is suitable for base64-encoding
// and passing to the Gmail API's messages.send endpoint.
//
// The format follows RFC 2822 §3.6 with the mandatory CRLF line endings:
//
//	From: "Display Name" <sender@example.com>\r\n
//	To: recipient@example.com\r\n
//	Subject: Hello\r\n
//	Content-Type: text/html; charset=utf-8\r\n
//	\r\n
//	<html>...</html>
//
// Parameters:
//   - senderName:  Display name for the From header. If empty, the From
//     header contains only the bare email address.
//   - senderEmail: The sender's email address.
//   - msg:         A validated Message value object.
func formatRawRFC2822(senderName, senderEmail string, msg Message) string {
	var from string
	if senderName != "" {
		from = fmt.Sprintf("\"%s\" <%s>", senderName, senderEmail)
	} else {
		from = senderEmail
	}

	return "From: " + from + "\r\n" +
		"To: " + msg.To() + "\r\n" +
		"Subject: " + msg.Subject() + "\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"\r\n" +
		msg.HTML()
}
