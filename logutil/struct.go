// Package logutil provides lightweight logging helpers for debugging
// and inspecting data structures during development. It serializes
// Go values to JSON and prints them to standard output with a
// "PAYLOAD" prefix for easy identification in log output.
package logutil

import (
	"encoding/json"
	"fmt"
)

// Payloader is a utility type for serializing Go values to JSON strings
// and printing them to stdout. It is designed for quick debugging and
// payload inspection during development.
//
// Usage:
//
//	p := &logutil.Payloader{}
//	p.Print(myStruct)  // prints: PAYLOAD {"field":"value",...}
type Payloader struct{}

// Marshal serializes any Go value to a JSON string using encoding/json.
// If the value cannot be marshalled (e.g., it contains channels or
// functions), it returns an empty string instead of propagating the error.
// This makes it safe to use in logging contexts where you don't want
// serialization failures to interrupt program flow.
//
// Example:
//
//	p := &logutil.Payloader{}
//	jsonStr := p.Marshal(map[string]int{"count": 42})
//	// jsonStr: `{"count":42}`
func (j *Payloader) Marshal(v any) string {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(jsonBytes)
}

// Print serializes the given value to JSON and prints it to stdout
// with a "PAYLOAD " prefix. This is a convenience method that combines
// Marshal and fmt.Println for quick debugging of request/response
// payloads, database results, or any other data structure.
//
// Example:
//
//	p := &logutil.Payloader{}
//	p.Print(user)  // Output: PAYLOAD {"id":"abc","name":"Alice"}
func (j *Payloader) Print(v any) {
	fmt.Println("PAYLOAD " + j.Marshal(v))
}
