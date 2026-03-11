package errors

import (
	"fmt"
	"strings"
)

const defaultLanguage = "en"

var supportedLanguages = map[string]bool{
	"en": true,
	"th": true,
}

// translations maps error code -> language -> message template.
// Populated at startup via RegisterTranslations. Read-only after init.
var translations = map[string]map[string]string{}

// RegisterTranslations registers translation maps for error codes.
// Must be called during application startup before handling requests.
//
// Example:
//
//	errors.RegisterTranslations(map[string]map[string]string{
//	    "AUTH_LOGIN_INVALID_CREDENTIALS": {
//	        "en": "The email or password you entered is incorrect.",
//	        "th": "อีเมลหรือรหัสผ่านที่คุณกรอกไม่ถูกต้อง",
//	    },
//	})
func RegisterTranslations(t map[string]map[string]string) {
	for code, langs := range t {
		translations[code] = langs
	}
}

// ResolveLanguage parses the Accept-Language header value and returns
// the best matching supported language code. Returns "en" if the header
// is empty, unparseable, or contains no supported language.
//
// Supports formats: "th", "th-TH", "en-US,th;q=0.9", etc.
//
// Example:
//
//	lang := errors.ResolveLanguage("th-TH,en;q=0.5") // returns "th"
//	lang := errors.ResolveLanguage("")                // returns "en"
//	lang := errors.ResolveLanguage("fr")              // returns "en"
func ResolveLanguage(acceptLanguage string) string {
	if acceptLanguage == "" {
		return defaultLanguage
	}

	for _, tag := range strings.Split(acceptLanguage, ",") {
		// Remove quality value (e.g., ";q=0.9")
		tag = strings.TrimSpace(strings.SplitN(tag, ";", 2)[0])
		// Extract primary language subtag (e.g., "th" from "th-TH")
		lang := strings.ToLower(strings.TrimSpace(strings.SplitN(tag, "-", 2)[0]))

		if supportedLanguages[lang] {
			return lang
		}
	}

	return defaultLanguage
}

// ResolveMessage looks up the translated message for the given AppError
// and language. If Details is a map[string]interface{}, any {placeholder}
// in the message is replaced with the corresponding value.
//
// Fallback chain: requested lang -> "en" -> appErr.Message -> appErr.Code
//
// Example:
//
//	msg := errors.ResolveMessage(appErr, "th")
func ResolveMessage(appErr *AppError, lang string) string {
	var msg string

	if langs, ok := translations[appErr.Code]; ok {
		if translated, exists := langs[lang]; exists {
			msg = translated
		} else if fallback, exists := langs[defaultLanguage]; exists {
			msg = fallback
		}
	}

	// Fallback to AppError.Message or Code
	if msg == "" {
		if appErr.Message != "" {
			msg = appErr.Message
		} else {
			msg = appErr.Code
		}
	}

	// Replace {placeholder} from Details
	if m, ok := appErr.Details.(map[string]interface{}); ok {
		for k, v := range m {
			msg = strings.ReplaceAll(msg, "{"+k+"}", fmt.Sprintf("%v", v))
		}
	}

	return msg
}
