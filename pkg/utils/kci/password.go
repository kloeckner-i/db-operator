package kci

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"github.com/sirupsen/logrus"
)

const (
	mininumLength = 8
	digits        = "0123456789"
	specials      = "-_" // include only uri safe special charactors
	uppers        = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowers        = "abcdefghijklmnopqrstuvwxyz"
)

// GeneratePass generates secure password string
func GeneratePass() string {
	password, err := generatePass(10, 8, 2, true)
	if err != nil {
		logrus.Fatalf("can not generate password - %s", err)
	}
	return password
}

func generatePass(numLetters, numDigits, numSpecials int, mixCase bool) (string, error) {
	length := numLetters + numDigits + numSpecials

	if length < mininumLength {
		return "", fmt.Errorf("total length of the password should be at least bigger than %d", mininumLength)
	}

	if mixCase && (numLetters < 2) {
		return "", errors.New("can not mix case when the length of letters is smaller than 2")
	}

	var bufAll []rune

	if numDigits != 0 {
		bufDigits, err := selectRandomCharacters(digits, numDigits)
		if err != nil {
			return "", fmt.Errorf("failed to select digits for password: %v", err)
		}
		bufAll = append(bufAll, bufDigits...)
	}

	if numSpecials != 0 {
		bufSpecials, err := selectRandomCharacters(specials, numSpecials)
		if err != nil {
			return "", fmt.Errorf("failed to select special characters for password: %v", err)
		}
		bufAll = append(bufAll, bufSpecials...)
	}

	minNumUppers := int64(0)
	maxNumUppers := int64(numLetters)
	if mixCase {
		minNumUppers = 1
		maxNumUppers = maxNumUppers - 1
	}

	numUppers, err := secureRandomIntWithinRange(minNumUppers, maxNumUppers)
	if err != nil {
		return "", err
	}

	if numUppers != 0 {
		bufUppers, err := selectRandomCharacters(uppers, int(numUppers))
		if err != nil {
			return "", fmt.Errorf("failed to select uppercase letters for password: %v", err)
		}
		bufAll = append(bufAll, bufUppers...)
	}

	numLowers := numLetters - int(numUppers)
	if numLowers != 0 {
		bufLowers, err := selectRandomCharacters(lowers, numLowers)
		if err != nil {
			return "", fmt.Errorf("failed to select lowercase letters for password: %v", err)
		}
		bufAll = append(bufAll, bufLowers...)
	}

	password, err := secureShuffle(string(bufAll))
	if err != nil {
		return "", fmt.Errorf("failed to shuffle password: %v", err)
	}

	return password, nil
}

// Select n random characters from the source string.
func selectRandomCharacters(src string, n int) ([]rune, error) {
	c := []rune(src)
	selection := make([]rune, n)

	for i := 0; i < n; i++ {
		r, err := secureRandomIntWithinRange(0, int64(len(c)))
		if err != nil {
			return nil, err
		}
		selection[i] = c[r]
	}

	return selection, nil
}

// Implementation of the Fisher–Yates shuffle using crypto/rand.
func secureShuffle(src string) (string, error) {
	c := []rune(src)
	n := int64(len(c))

	for i := int64(0); i < n; i++ {
		r, err := secureRandomIntWithinRange(i, n)
		if err != nil {
			return "", err
		}
		c[r], c[i] = c[i], c[r]
	}

	return string(c), nil
}

// Generate a secure random integer within the range min <= x < max.
func secureRandomIntWithinRange(min int64, max int64) (int64, error) {
	next, err := rand.Int(rand.Reader, new(big.Int).SetInt64(max-min))
	if err != nil {
		return -1, err
	}
	return min + next.Int64(), nil
}
