package problem

import "strings"

const (
	// MessageTextLimit is the limit of the text of the judge result.
	//
	// This is used to limit the length of input, output, answer, checker stderr, etc.
	MessageTextLimit = 128
)

// TruncateMessage is a function to truncate the message of the judge result.
func TruncateMessage(s string) string {
	s = strings.TrimRight(s, "\n ")
	if len(s) > MessageTextLimit-3 {
		return s[:MessageTextLimit-3] + "..."
	}
	return s
}
