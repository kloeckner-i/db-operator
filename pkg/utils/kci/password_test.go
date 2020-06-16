package kci

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Verify we generate a valid password based on the default rule.
func TestGeneratePass(t *testing.T) {
	generatedPassword := GeneratePass()

	if assert.NotEmpty(t, generatedPassword) {
		assert.Equal(t, 20, len(generatedPassword))
		assert.Equal(t, 10, countOccurrences(generatedPassword, uppers+lowers))
		assert.Equal(t, 8, countOccurrences(generatedPassword, digits))
		assert.Equal(t, 2, countOccurrences(generatedPassword, specials))
	}
}

func TestSelectRandomCharacters(t *testing.T) {
	src := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	freq := make(map[rune]int)
	for i := 0; i < 1000; i++ {
		res, err := selectRandomCharacters(src, 5)
		if err != nil {
			t.Error(err)
		}

		for _, c := range res {
			freq[c]++
		}
	}

	for rune, n := range freq {
		if n < 150 || n >= 250 {
			t.Errorf("Unexpected outlier: rune = %d, count = %d", rune, n)
		}
	}
}

func TestSecureShuffle(t *testing.T) {
	src := "The 0,u|ck brow/|/ f°? _|u^^ps °\\/er one3 lazj 1)°g$."

	s1, err := secureShuffle(src)
	if err != nil {
		t.Error(err)
	}

	s2, err := secureShuffle(src)
	if err != nil {
		t.Error(err)
	}

	assert.Len(t, s1, len(src))
	assert.Len(t, s2, len(src))
	assert.Equal(t, runeFrequency(src), runeFrequency(s1))
	assert.Equal(t, runeFrequency(src), runeFrequency(s2))
	assert.NotEqual(t, src, s1)
	assert.NotEqual(t, s1, s2)
}

// The number of occurances per rune in the source string.
func runeFrequency(src string) map[rune]int {
	freq := make(map[rune]int)
	for _, c := range src {
		freq[c]++
	}
	return freq
}

// The number of occurrences of a rune/s in a string.
func countOccurrences(src string, runes string) int {
	re := regexp.MustCompile("[" + runes + "]")
	return len(re.FindAllString(src, -1))
}
