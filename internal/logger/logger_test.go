/*
   Copyright 2022 CESS (Cumulus Encrypted Storage System) authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLogs(t *testing.T) {
	log_files := make(map[string]string, 2)
	log_files["info"] = "./info.log"
	log_files["err"] = "./err.log"
	_, err := NewLogs(log_files)
	assert.NoError(t, err)
}
