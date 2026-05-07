// Package stringutil provides small helpers for normalizing identifiers used
// across itestkit (project names, instance names, environment variable
// fragments). It is internal to itestkit and must not be imported elsewhere.
package stringutil

import "strings"

// NormalizeIdentifier converts an identifier into a lowercase, underscore-or-dash
// safe form. It lowercases ASCII letters, keeps digits, replaces any other
// character with '_', and collapses repeated separators. The result is suitable
// for use as a database name, container label, or network suffix.
//
// Examples:
//
//	NormalizeIdentifier("Wallet Service") -> "wallet_service"
//	NormalizeIdentifier("Order-Service!") -> "order-service_"
//	NormalizeIdentifier("CRM")            -> "crm"
func NormalizeIdentifier(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + 32)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

// EnvFragment converts an instance name into the uppercase form used in
// exported environment variables. Dashes become underscores so the resulting
// variable name is shell-safe.
//
// Examples:
//
//	EnvFragment("main")    -> "MAIN"
//	EnvFragment("main-db") -> "MAIN_DB"
//	EnvFragment("main_db") -> "MAIN_DB"
func EnvFragment(s string) string {
	if s == "" {
		return ""
	}
	upper := strings.ToUpper(s)
	return strings.ReplaceAll(upper, "-", "_")
}
