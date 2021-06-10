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

package gcloud

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
)

// GetServiceAccount reads file which contains google service account credentials and parse it
func GetServiceAccount() ServiceAccount {
	var serviceaccount ServiceAccount

	credentialPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	credentialFile, err := os.Open(credentialPath)
	if err != nil {
		logrus.Fatalf("failed to open service account file - %s", err)
	}
	// defer the closing of our jsonFile so that we can parse it later on
	defer credentialFile.Close()

	credentialValues, _ := ioutil.ReadAll(credentialFile)

	// parse credentials.json file
	json.Unmarshal([]byte(credentialValues), &serviceaccount)

	return serviceaccount
}
