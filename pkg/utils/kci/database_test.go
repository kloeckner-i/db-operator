package kci

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStringSanitize(t *testing.T) {
	// Handles very short limits
	src := "HelloWorld"
	actual := StringSanitize(src, 6)
	assert.Equal(t, "936a18", actual)

	// Sanitizes successfully
	src = "Hello**World$"
	actual = StringSanitize(src, 32)
	assert.Equal(t, "hello__world$", actual)

	// Truncates very long values
	src = "The quick brown fox jumps over the lazy dog"
	actual = StringSanitize(src, 32)
	assert.Equal(t, "the_quick_brown_fox_jum_19e1ae50", actual)

	// Truncates very long values, avoid hash collision
	src = "The quick brown fox jumps over the lazy cat"
	actual = StringSanitize(src, 32)
	assert.NotEqual(t, "the_quick_brown_fox_jum_19e1ae50", actual)
}
