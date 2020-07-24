package kci

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

// StringSanitize sanitizes and truncates a string to a fixed length using a hash function.
// useful for restricting the length and content of user supplied database identifiers.
func StringSanitize(s string, limit int) string {
	// use lowercase exclusively for identifiers.
	// https://dev.mysql.com/doc/refman/5.7/en/identifier-case-sensitivity.html
	s = strings.ToLower(s)

	// Strip out any unsupported characters.
	// https://dev.mysql.com/doc/refman/5.7/en/identifiers.html
	unsupportedChars := regexp.MustCompile(`[^0-9a-zA-Z$_]`)
	s = unsupportedChars.ReplaceAllString(s, "_")

	if len(s) <= limit {
		return s
	}

	if limit <= 9 {
		return fmt.Sprintf("%x", sha256.New().Sum([]byte(s)))[:limit]
	}

	hash := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%s_%x", s[:limit-9], hash[:4])
}
