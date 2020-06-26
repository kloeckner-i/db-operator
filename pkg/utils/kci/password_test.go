package kci

import (
	"regexp"
	"testing"

	"github.com/kloeckner-i/can-haz-password/password"
	"github.com/stretchr/testify/assert"
)

// Verify we generate a valid password based on the default rule.
func TestGeneratePass(t *testing.T) {
	generatedPassword := GeneratePass()

	if assert.NotEmpty(t, generatedPassword) {
		assert.True(t, len(generatedPassword) >= 20)
		assert.True(t, countOccurrences(generatedPassword, password.UppercaseCharacters+password.LowercaseCharacters) >= 10)
		assert.True(t, countOccurrences(generatedPassword, password.DigitCharacters) >= 8)
		assert.True(t, countOccurrences(generatedPassword, password.URLSafeSpecialCharacters) >= 2)
	}
}

// The number of occurrences of a rune/s in a string.
func countOccurrences(src string, runes string) int {
	re := regexp.MustCompile("[" + runes + "]")
	return len(re.FindAllString(src, -1))
}
