package gmail

import (
	"context"
	"encoding/base64"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	googlemail "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// ---------------------------------------------------------------------------
// EmailSender — the abstraction (Dependency Inversion Principle)
// ---------------------------------------------------------------------------

// EmailSender is the primary abstraction that high-level modules should
// depend on. It follows the Interface Segregation Principle by exposing
// only the single capability that consumers need: sending an email.
//
// By programming against this interface rather than the concrete GmailSender,
// application code gains:
//
//   - Testability: Unit tests can inject a mock or stub implementation
//     without hitting the Gmail API.
//   - Substitutability (Liskov): Any future provider (SendGrid, SES, SMTP)
//     can satisfy this contract without changing calling code.
//   - Loose coupling: The business layer never imports Google-specific types.
//
// The context.Context parameter allows callers to propagate deadlines,
// cancellation signals, and trace metadata into the send operation.
type EmailSender interface {
	Send(ctx context.Context, msg Message) error
}

// ---------------------------------------------------------------------------
// GmailSender — concrete implementation (Single Responsibility Principle)
// ---------------------------------------------------------------------------

// GmailSender sends emails through the Gmail REST API (v1) using OAuth2
// credentials. Its single responsibility is to translate a validated
// Message into a Gmail API call.
//
// It is intentionally NOT exported as a constructor return type — callers
// receive an EmailSender interface from NewGmailSender(), keeping their
// dependency on the abstraction rather than the concrete type. The struct
// itself is exported so that callers who truly need the concrete type
// (e.g. for type-assertions in advanced scenarios) can access it.
//
// Thread safety: GmailSender is safe for concurrent use. The underlying
// googlemail.Service uses an http.Client whose Transport is safe for
// concurrent requests.
type GmailSender struct {
	service     *googlemail.Service
	senderName  string
	senderEmail string
}

// NewGmailSender constructs a ready-to-use EmailSender backed by the
// Gmail REST API. It performs the following steps:
//
//  1. Validates the Config (fails fast on missing credentials).
//  2. Builds an OAuth2 config pointing at Google's token endpoint.
//  3. Creates an HTTP client that automatically refreshes the access
//     token using the provided refresh token.
//  4. Initialises the Gmail API service with that HTTP client.
//
// The returned EmailSender is safe for concurrent use and should be
// created once at application startup and shared across goroutines.
//
// Returns a wrapped error if:
//   - Config validation fails (sentinel errors from constants.go).
//   - The Gmail API service cannot be initialised (network / auth issue).
func NewGmailSender(cfg Config) (EmailSender, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("gmail: invalid config: %w", err)
	}

	oauthCfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  oauthRedirectURL,
		Scopes:       []string{googlemail.GmailSendScope},
		Endpoint:     google.Endpoint,
	}

	token := &oauth2.Token{RefreshToken: cfg.RefreshToken}
	httpClient := oauthCfg.Client(context.Background(), token)

	service, err := googlemail.NewService(
		context.Background(),
		option.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("gmail: failed to create gmail service: %w", err)
	}

	return &GmailSender{
		service:     service,
		senderName:  cfg.SenderName,
		senderEmail: cfg.SenderEmail,
	}, nil
}

// Send delivers a single email message through the Gmail API.
//
// It converts the Message into an RFC 2822 formatted string, base64url-
// encodes it (as required by the Gmail API), and calls the Gmail
// users.messages.send endpoint.
//
// The ctx parameter is forwarded to the Gmail API call, allowing callers
// to enforce deadlines or cancel long-running requests:
//
//	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
//	defer cancel()
//	err := sender.Send(ctx, msg)
//
// Errors from the Gmail API are wrapped with additional context to aid
// debugging (e.g. "gmail: failed to send email to user@example.com: ...").
func (s *GmailSender) Send(ctx context.Context, msg Message) error {
	raw := formatRawRFC2822(s.senderName, s.senderEmail, msg)

	gmailMsg := &googlemail.Message{
		Raw: base64.URLEncoding.EncodeToString([]byte(raw)),
	}

	_, err := s.service.Users.Messages.Send(gmailSpecialUser, gmailMsg).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("gmail: failed to send email to %s: %w", msg.To(), err)
	}

	return nil
}
