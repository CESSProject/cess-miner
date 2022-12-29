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

package configs

import "time"

// byte size
const (
	SIZE_1KiB      = 1024
	SIZE_1MiB      = 1024 * SIZE_1KiB
	SIZE_1GiB      = 1024 * SIZE_1MiB
	SIZE_SLICE     = 512 * SIZE_1MiB
	SIZE_SLICE_KiB = 512 * SIZE_1KiB
)

// account
const (
	// CESS token precision
	CESSTokenPrecision = 1_000_000_000_000
	// MinimumBalance is the minimum balance required for the program to run
	// The unit is pico
	MinimumBalance = 2 * CESSTokenPrecision
	//
	DepositPerTiB = 2000
	//
	DirPermission = 755
	//
	ClearMemInterval = time.Duration(time.Minute * 10)
)

const (
	TokenAccuracy = "000000000000" //Unit precision of CESS coins
	ExitColling   = 28800          //blocks
	// Time out waiting for transaction completion
	TimeOut_WaitBlock = time.Duration(time.Second * 15)
	// BlockInterval is the time interval for generating blocks, in seconds
	BlockInterval = time.Second * time.Duration(6)
	// Token length
	TokenLength = 32
	//
	NumOfFillerSubmitted = 1
)

const (
	// Maximum number of connections in the miner's certification space
	MAX_TCP_CONNECTION uint8 = 3
	//
	TCP_Message_Read_Buffers = 10
	//
	TCP_MaxPacketSize = SIZE_1KiB * 1024
	//
	Tcp_Dial_Timeout    = time.Duration(time.Second * 5)
	ReplaceFileInterval = time.Duration(time.Minute * 5)
	TimeOut_WaitReport  = time.Duration(time.Second * 10)
	TimeOut_WaitTag     = time.Duration(time.Minute * 5)
)

const (
	URL_GetReport          = "http://localhost:80/get_report"
	URL_GetReport_Callback = "/report"
	URL_FillFile           = "http://localhost:80/fill_random_file"
	URL_GetTag             = "http://localhost:80/process_data"
	URL_GetTag_Callback    = "/tag"
	SgxMappingPath         = "/kaleido"
	SgxReportSuc           = 100000
	BlockSize              = SIZE_1KiB
)

const (
	HELP_Head = `Please check with the following help information:
    1.Check if the wallet balance is sufficient
    2.Block hash:`
	HELP_register          = `    3.Check the Sminer_Registered transaction event result in the block hash above:`
	HELP_UpdateAddress     = `    3.Check the Sminer_UpdataIp transaction event result in the block hash above:`
	HELP_UpdataBeneficiary = `    3.Check the Sminer_UpdataBeneficiary transaction event result in the block hash above:`
	HELP_MinerExit         = `    3.Check the Sminer_MinerExit transaction event result in the block hash above:`
	HELP_MinerIncrease     = `    3.Check the Sminer_IncreaseCollateral transaction event result in the block hash above:`
	HELP_MinerWithdraw     = `    3.Check the Sminer_Redeemed transaction event result in the block hash above:`
	HELP_Tail              = `		If system.ExtrinsicFailed is prompted, it means failure;
        If system.ExtrinsicSuccess is prompted, it means success;`
)

// log file
var (
	LogFiles = []string{
		"common",    //common log
		"panic",     //panic log
		"upfile",    //upload file log
		"challenge", //challenge log
		"replace",   //replace file log
		"space",     //space log
	}
)
