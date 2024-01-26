/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package configs

import "time"

const (
	TokenTCESS = 1000000000000000000
	// the time to wait for the event, in seconds
	TimeToWaitEvent = time.Duration(time.Second * 30)
	// Default config file
	DefaultConfigFile = "conf.yaml"
	//
	DefaultWorkspace = "/"
	//
	DefaultServicePort = 4001
	//
	DefaultRpcAddr1 = "wss://testnet-rpc0.cess.cloud/ws/"
	DefaultRpcAddr2 = "wss://testnet-rpc1.cess.cloud/ws/"
	//
	DefaultBootNodeAddr = "_dnsaddr.boot-bucket-testnet.cess.cloud"
	//
	DefaultDeossAddr = "http://deoss-pub-gateway.cess.cloud/"
)

const (
	OrserState_CalcTag uint8 = 2
)

const (
	Err_tee_Busy         = "is being fully calculated"
	Err_ctx_exceeded     = "context deadline exceeded"
	Err_file_not_fount   = "no such file"
	Err_miner_not_exists = "the miner not exists"
)

const (
	DbDir            = "db"
	LogDir           = "log"
	SpaceDir         = "space"
	PoisDir          = "pois"
	AccDir           = "acc"
	RandomDir        = "random"
	PeersFile        = "peers"
	Podr2PubkeyFile  = ".podr2pubkey"
	IdleProofFile    = "idleproof"
	ServiceProofFile = "serviceproof"
)
