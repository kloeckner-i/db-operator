/*
 * Copyright 2021 kloeckner.i GmbH
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
