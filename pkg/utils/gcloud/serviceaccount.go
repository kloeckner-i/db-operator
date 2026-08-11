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
	"context"
	"encoding/json"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// GetServiceAccount reads file which contains google service account credentials and parse it
func GetServiceAccount(ctx context.Context) ServiceAccount {
	var serviceaccount ServiceAccount

	log := log.FromContext(ctx)
	credentialPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	credentialValues, err := os.ReadFile(credentialPath)
	if err != nil {
		log.Error(err, "failed to open service account file")
	}

	// parse credentials.json file
	err = json.Unmarshal([]byte(credentialValues), &serviceaccount)
	if err != nil {
		log.Error(err, "failed to parse service account file")
	}

	return serviceaccount
}
