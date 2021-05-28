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
	"fmt"
	"time"

	"github.com/mitchellh/hashstructure"
	"github.com/sirupsen/logrus"
)

func appendIfMissing(slice []string, s string) []string {
	if len(slice) == 0 {
		return append(slice, s)
	}

	for _, ele := range slice {
		if ele == s {
			return slice
		}
	}
	return append(slice, s)
}

func removeItem(slice []string, s string) []string {
	newslice := []string{}

	for _, ele := range slice {
		if ele != s {
			newslice = append(newslice, ele)
		}
	}
	return newslice
}

// StringNotEmpty return the first not empty string
func StringNotEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

type stop struct {
	error
}

// Retry retries given function for given attempts times with given intervals
func Retry(attempts int, intervals time.Duration, fn func() error) error {
	if err := fn(); err != nil {
		if s, ok := err.(stop); ok {
			// Return the original error for later checking
			return s.error
		}

		if attempts--; attempts > 0 {
			time.Sleep(intervals)
			return Retry(attempts, intervals, fn)
		}
		return err
	}
	return nil
}

// GenerateChecksum generates hash value of given interface
func GenerateChecksum(v interface{}) string {
	hash, err := hashstructure.Hash(v, nil)
	if err != nil {
		logrus.Fatalf("Failed to generate hash: %v", err)
	}

	return fmt.Sprintf("%d", hash)
}

// TimeTrack tracks seconds since given start time
func TimeTrack(start time.Time) float64 {
	return time.Since(start).Seconds()
}
