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

package chain

// Pallert
const (
	state_FileBank    = "FileBank"
	state_FileMap     = "FileMap"
	state_Sminer      = "Sminer"
	state_SegmentBook = "SegmentBook"
	state_System      = "System"
)

// Chain state
const (
	// System
	system_Account = "Account"
	system_Events  = "Events"
	// Sminer
	sminer_MinerItems   = "MinerItems"
	sminer_MinerDetails = "MinerDetails"
	sminer_MinerLockIn  = "MinerLockIn"
	// SegmentBook
	segmentBook_MinerHoldSlice    = "MinerHoldSlice"
	segmentBook_ChallengeSnapshot = "ChallengeSnapshot"
	// FileMap
	fileMap_FileMetaInfo  = "File"
	fileMap_SchedulerPuk  = "SchedulerPuk"
	fileMap_SchedulerInfo = "SchedulerMap"
	// FileBank
	fileBank_FillerMap   = "FillerMap"
	fileBank_InvalidFile = "InvalidFile"
)

// Extrinsics
const (
	tx_Sminer_Register               = "Sminer.regnstk"
	ChainTx_SegmentBook_IntentSubmit = "SegmentBook.intent_submit"
	tx_Sminer_ExitMining             = "Sminer.exit_miner"
	tx_Sminer_Withdraw               = "Sminer.withdraw"
	tx_Sminer_UpdateIp               = "Sminer.update_ip"
	tx_Sminer_UpdateBeneficiary      = "Sminer.update_beneficiary"
	tx_Sminer_Increase               = "Sminer.increase_collateral"
	tx_SegmentBook_SubmitProve       = "SegmentBook.submit_prove"
	tx_FileBank_ClearInvalidFile     = "FileBank.clear_invalid_file"
	FileBank_ClearFiller             = "FileBank.clear_all_filler"
)

const (
	FILE_STATE_ACTIVE  = "active"
	FILE_STATE_PENDING = "pending"
)

const (
	MINER_STATE_POSITIVE = "positive"
	MINER_STATE_FROZEN   = "frozen"
	MINER_STATE_EXIT     = "exit"
)
