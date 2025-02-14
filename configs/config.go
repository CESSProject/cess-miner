/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package configs

import (
	"net/http"
	"time"
)

const (
	//
	TokenTCESS = 1000000000000000000
	// the time to wait for the event, in seconds
	TimeToWaitEvent = time.Duration(time.Second * 30)
	// default config file
	DefaultConfigFile = "conf.yaml"
	//
	DefaultWorkspace = "/"
	//
	DefaultServicePort = 15001
	//
	DefaultRpcAddr = "wss://testnet-rpc.cess.network/ws/"
	//
	MinTagFileSize = 600000
	//
	FileMode = 0755
)

const (
	Err_tee_Busy         = "is being fully calculated"
	Err_ctx_exceeded     = "context deadline exceeded"
	Err_file_not_fount   = "no such file"
	Err_miner_not_exists = "the miner not exists"
)

const (
	DevNet  = "devnet"
	TestNet = "testnet"
	MainNet = "mainnet"
)

const (
	DefaultGW1 = "https://deoss-sgp.cess.network"
	DefaultGW2 = "https://deoss-sv.cess.network"
	DefaultGW3 = "https://deoss-fra.cess.network"
)

var GlobalTransport = &http.Transport{
	DisableKeepAlives: true,
}
