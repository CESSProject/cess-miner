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

// return code
const (
	//success
	Code_200 = 200
	//bad request
	Code_400 = 400
	//forbidden
	Code_403 = 403
	//not found
	Code_404 = 404
	//server internal error
	Code_500 = 500
	//The block was produced but the event was not resolved
	Code_600 = 600
)

// byte size
const (
	SIZE_1KiB  = 1024
	SIZE_1MiB  = 1024 * SIZE_1KiB
	SIZE_1GiB  = 1024 * SIZE_1MiB
	SIZE_SLICE = 512 * SIZE_1MiB
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
	//
	ClearFilesInterval = time.Duration(time.Minute * 5)
)

const (
	FillerSize         = 8 * SIZE_1MiB
	TimeToWaitEvents_S = 20             //The time to wait for the event, in seconds
	TokenAccuracy      = "000000000000" //Unit precision of CESS coins
	ExitColling        = 28800          //blocks
	BlockSize          = 1024 * 1024    //1MB
	ScanBlockSize      = 512 * 1024     //512KB
	// Time out waiting for transaction completion
	TimeOut_WaitBlock = time.Duration(time.Second * 15)
	// BlockInterval is the time interval for generating blocks, in seconds
	BlockInterval = time.Second * time.Duration(6)
	// Token length
	TokenLength = 32
)

const (
	// Maximum number of connections in the miner's certification space
	MAX_TCP_CONNECTION uint8 = 3
	// Tcp client connection interval
	TCP_Connection_Interval = time.Duration(time.Millisecond * 100)
	// Tcp message interval
	TCP_Message_Interval = time.Duration(time.Millisecond * 10)
	// Tcp short message waiting time
	TCP_Time_WaitNotification = time.Duration(time.Second * 10)
	// Tcp short message waiting time
	TCP_Time_WaitMsg = time.Duration(time.Minute)
	// Tcp short message waiting time
	TCP_FillerMessage_WaitingTime = time.Duration(time.Second * 150)
	// The slowest tcp transfers bytes per second
	TCP_Transmission_Slowest = SIZE_1KiB * 10
	// Number of tcp message caches
	TCP_Message_Send_Buffers = 10
	TCP_Message_Read_Buffers = 10
	//
	TCP_SendBuffer = 8192
	TCP_ReadBuffer = 12000
	TCP_TagBuffer  = 2012
	//
	TCP_MaxPacketSize = SIZE_1KiB * 32
	//
	Tcp_Dial_Timeout = time.Duration(time.Second * 5)
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
		"common",    //General log
		"panic",     //Panic log
		"upfile",    //Upload file log
		"challenge", //Challenge log
		"clear",     //Clear log
	}
)
