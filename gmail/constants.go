// Package gmail provides an email-sending abstraction built on top of the
// Gmail REST API (v1). It follows the Dependency Inversion Principle by
// exposing an EmailSender interface that high-level modules depend on,
// while the concrete GmailSender implementation remains an internal detail.
//
// Typical usage:
//
//	cfg := gmail.Config{
//	    ClientID:     os.Getenv("GMAIL_CLIENT_ID"),
//	    ClientSecret: os.Getenv("GMAIL_CLIENT_SECRET"),
//	    RefreshToken: os.Getenv("GMAIL_REFRESH_TOKEN"),
//	    SenderName:   "OneTrust Support",
//	    SenderEmail:  "support@onetrust.com",
//	}
//
//	sender, err := gmail.NewGmailSender(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	msg := gmail.NewMessage("user@example.com", "Welcome!", "<h1>Hello</h1>")
//	if err := sender.Send(context.Background(), msg); err != nil {
//	    log.Println("failed to send email:", err)
//	}
package gmail

import "errors"

// oauthRedirectURL is the Google OAuth2 playground redirect URL used for
// server-to-server token refresh flows. It is intentionally unexported
// because callers should never need to change this value — it is a fixed
// constant dictated by the Google OAuth2 token exchange protocol when
// using refresh tokens obtained via the OAuth Playground.
const oauthRedirectURL = "https://developers.google.com/oauthplayground"

// gmailSpecialUser is the Gmail API sentinel that refers to the
// authenticated user's own mailbox. The Gmail API requires this literal
// string "me" in place of an email address when performing operations
// on the currently authenticated account.
const gmailSpecialUser = "me"

// Sentinel errors returned by this package. They are defined as package-level
// variables so callers can use errors.Is() for matching:
//
//	if errors.Is(err, gmail.ErrEmptyRecipient) { ... }
var (
	// ErrEmptyRecipient is returned when a Message is constructed with a
	// blank To address. Every email must have at least one recipient.
	ErrEmptyRecipient = errors.New("gmail: recipient (to) address must not be empty")

	// ErrEmptySubject is returned when a Message is constructed with a
	// blank Subject line. While the SMTP protocol technically allows empty
	// subjects, it is almost always a mistake in application code.
	ErrEmptySubject = errors.New("gmail: subject must not be empty")

	// ErrEmptyBody is returned when a Message is constructed with a blank
	// HTML body. An email with no content provides no value to the recipient.
	ErrEmptyBody = errors.New("gmail: html body must not be empty")

	// ErrEmptyClientID is returned when Config.ClientID is blank.
	ErrEmptyClientID = errors.New("gmail: client id must not be empty")

	// ErrEmptyClientSecret is returned when Config.ClientSecret is blank.
	ErrEmptyClientSecret = errors.New("gmail: client secret must not be empty")

	// ErrEmptyRefreshToken is returned when Config.RefreshToken is blank.
	ErrEmptyRefreshToken = errors.New("gmail: refresh token must not be empty")

	// ErrEmptySenderEmail is returned when Config.SenderEmail is blank.
	ErrEmptySenderEmail = errors.New("gmail: sender email must not be empty")
)
