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

package dbinstance

import "errors"

var (
	// ErrAlreadyExists is thrown when db instance already exists
	ErrAlreadyExists = errors.New("instance already exists")
	// ErrNotExists is thrown when db instance does not exists
	ErrNotExists = errors.New("instance does not exists")
	// ErrInstanceNotReady is throw is gsql instance is still not marked as Ready
	ErrInstanceNotReady = errors.New("instance is not ready")
)
