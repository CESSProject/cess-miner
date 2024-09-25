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
	ERR_FragmentHash        = "the fragment hash is wrong"
	ERR_FragmentNotMatchFid = "fragment does not match fid"
	ERR_NotFound            = "not found"
	ERR_HashLength          = "invalid fid or fragment"
	ERR_EmptyHashName       = "empty fid or fragment"

	// signature err
	ERR_InvalidSignature = "invalid signature"

	// range err
	ERR_InvalidRangeValue = "invalid range request"
	ERR_InvalidRangeTotal = "invalid range total"
	ERR_InvalidRangeStart = "invalid range start"
	ERR_InvalidRangeEnd   = "invalid range end"
)
