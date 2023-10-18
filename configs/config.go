/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package configs

import "time"

const (
	//
	TimeFormat = "2006-01-02 15:04:05"
	// the time to wait for the event, in seconds
	TimeToWaitEvent = time.Duration(time.Second * 12)
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
	DefaultBootNodeAddr = "_dnsaddr.boot-kldr-devnet.cess.cloud"
	//
	DefaultDeossAddr = ""
)

const (
	DbDir            = "db"
	LogDir           = "log"
	SpaceDir         = "space"
	PoisDir          = "pois"
	AccDir           = "acc"
	RandomDir        = "random"
	PeersFile        = "peers"
	IdleProofFile    = "idleproof"
	ServiceProofFile = "serviceproof"
)
