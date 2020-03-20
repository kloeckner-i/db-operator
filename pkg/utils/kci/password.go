package kci

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
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
	const (
		mininumLength = 8
		digits        = "0123456789"
		specials      = "-_" // include only uri safe special charactors
		uppers        = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lowers        = "abcdefghijklmnopqrstuvwxyz"
	)
	length := numLetters + numDigits + numSpecials

	if length < mininumLength {
		return "", fmt.Errorf("total length of the password should be at least bigger than %d", mininumLength)
	}

	if mixCase && (numLetters < 2) {
		return "", errors.New("can not mix case when the length of letters is smaller than 2")
	}

	rand.Seed(time.Now().UnixNano())
	var bufAll []byte

	if numDigits != 0 {
		bufDigits := randomElement(numDigits, digits)
		bufAll = append(bufAll, bufDigits...)
	}

	if numSpecials != 0 {
		bufSpecials := randomElement(numSpecials, specials)
		bufAll = append(bufAll, bufSpecials...)
	}

	numUppers := rand.Intn(numLetters)
	if numUppers == 0 && mixCase {
		numUppers = numUppers + 1
	}

	if numUppers != 0 {
		bufUppers := randomElement(numUppers, uppers)
		bufAll = append(bufAll, bufUppers...)
	}

	numLowers := numLetters - numUppers
	if numLowers != 0 {
		bufLowers := randomElement(numLowers, lowers)
		bufAll = append(bufAll, bufLowers...)
	}

	rand.Shuffle(len(bufAll), func(i, j int) {
		bufAll[i], bufAll[j] = bufAll[j], bufAll[i]
	})

	password := string(bufAll)

	return password, nil
}

func randomElement(length int, element string) []byte {
	buf := make([]byte, length)
	for i := 0; i < length; i++ {
		buf[i] = element[rand.Intn(len(element))]
	}

	return buf
}
