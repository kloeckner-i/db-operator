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

package database

import "testing"

func TestDatabaseUser_SetAccessType(t *testing.T) {
	type fields struct {
		Username   string
		Password   string
		AccessType string
	}
	type args struct {
		accessType string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &DatabaseUser{
				Username:   tt.fields.Username,
				Password:   tt.fields.Password,
				AccessType: tt.fields.AccessType,
			}
			user.SetAccessType(tt.args.accessType)
		})
	}
}
