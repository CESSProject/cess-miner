/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package logger

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLogs(t *testing.T) {
	log_files := make(map[string]string, 2)
	log_files["info"] = "./info.log"
	log_files["err"] = "./err.log"
	_, err := NewLogs(log_files)
	assert.NoError(t, err)
	os.Remove(log_files["info"])
	os.Remove(log_files["err"])
}
