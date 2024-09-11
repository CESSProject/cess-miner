/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package common

const (
	// ok
	OK = "ok"

	// server err
	ERR_SystemErr = "system error"

	// rpc err
	ERR_RPCConnection = "failed to connect to rpc, please try again later."

	// client err
	ERR_FragmentSize        = "the fragment size is wrong"
	ERR_FragmentNotMatchFid = "fragment does not match fid"
)
