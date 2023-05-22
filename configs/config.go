/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package configs

import "time"

const (
	// the time to wait for the event, in seconds
	TimeToWaitEvent = time.Duration(time.Second * 12)
	// Default config file
	DefaultConfigFile = "./conf.yaml"
	//
	DefaultWorkspace = "/"
)

const (
	HELP_common = `Please check with the following help information:
    1.Check if the wallet balance is sufficient
    2.Block hash:`
	HELP_register = `    3.Check the Sminer_Registered transaction event result in the block hash above:
        If system.ExtrinsicFailed is prompted, it means failure;
        If system.ExtrinsicSuccess is prompted, it means success;`
	HELP_UpdateAddress = `    3.Check the Sminer_UpdataIp transaction event result in the block hash above:
        If system.ExtrinsicFailed is prompted, it means failure;
        If system.ExtrinsicSuccess is prompted, it means success;`
	HELP_UpdataBeneficiary = `    3.Check the Sminer_UpdataBeneficiary transaction event result in the block hash above:
        If system.ExtrinsicFailed is prompted, it means failure;
        If system.ExtrinsicSuccess is prompted, it means success;`
)

const (
	DbDir    = "db"
	LogDir   = "log"
	SpaceDir = "space"
)

var LogFiles = []string{
	"log",
	"panic",
	"space",
	"report",
	"replace",
	"challenge",
}
