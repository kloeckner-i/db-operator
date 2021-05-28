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

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(s)))

	if limit <= 9 {
		return hash[:limit]
	}

	return fmt.Sprintf("%s_%s", s[:limit-9], hash[:8])
}
