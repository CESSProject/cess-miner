/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import "log"

const OK = "✅"
const ERR = "❌"

func logOK(msg string) {
	log.Print(OK, " ", msg)
}

func logERR(msg string) {
	log.Print(ERR, " ", msg)
}
