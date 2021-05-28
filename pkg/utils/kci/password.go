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

	"github.com/kloeckner-i/can-haz-password/password"
	"github.com/sirupsen/logrus"
)

// GeneratePass generates secure password string
func GeneratePass() string {
	generator := password.NewGenerator(newDbPasswordRule())
	password, err := generator.Generate()
	if err != nil {
		logrus.Fatalf("can not generate password - %s", err)
	}
	return password
}

// Minimum length of 20 characters, maximum length of 30 characters.
// Varied composition including special characters and uppercase and lowercase letters.
// Excludes consecutive dashes (for hybris compatibility) and uses only url safe special characters.
type dbPasswordRule struct {
	invalid *regexp.Regexp
}

func newDbPasswordRule() *dbPasswordRule {
	return &dbPasswordRule{
		// Hybris does not support consecutive dashes.
		invalid: regexp.MustCompile(`[-]{2,}`),
	}
}

func (r *dbPasswordRule) Config() *password.Configuration {
	return &password.Configuration{
		Length: 20,
		CharacterClasses: []password.CharacterClassConfiguration{
			{Characters: password.LowercaseCharacters + password.UppercaseCharacters, Minimum: 10},
			{Characters: password.DigitCharacters, Minimum: 8},
			{Characters: password.URLSafeSpecialCharacters, Minimum: 2},
		},
	}
}

func (r *dbPasswordRule) Valid(password []rune) bool {
	return !r.invalid.MatchString(string(password))
}
