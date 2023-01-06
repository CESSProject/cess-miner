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

import "github.com/centrifuge/go-substrate-rpc-client/v4/types"

// **************************************************************
// custom event type
// **************************************************************

// ------------------------SegmentBook----------------------------
type Event_PPBNoOnTimeSubmit struct {
	Phase     types.Phase
	Acc       types.AccountID
	SegmentId types.U64
	Topics    []types.Hash
}

type Event_PPDNoOnTimeSubmit struct {
	Phase     types.Phase
	Acc       types.AccountID
	SegmentId types.U64
	Topics    []types.Hash
}

type Event_SubmitReport struct {
	Phase  types.Phase
	Miner  types.AccountID
	Topics []types.Hash
}

type Event_VerifyProof struct {
	Phase  types.Phase
	Miner  types.AccountID
	Fileid types.Bytes
	Topics []types.Hash
}

type Event_OutstandingChallenges struct {
	Phase  types.Phase
	Miner  types.AccountID
	Fileid types.Bytes
	Topics []types.Hash
}

// ------------------------Sminer---------------------------------
type Event_Registered struct {
	Phase      types.Phase
	Acc        types.AccountID
	StakingVal types.U128
	Topics     []types.Hash
}

type Event_TimedTask struct {
	Phase  types.Phase
	Topics []types.Hash
}

type Event_DrawFaucetMoney struct {
	Phase  types.Phase
	Topics []types.Hash
}

type Event_FaucetTopUpMoney struct {
	Phase  types.Phase
	Acc    types.AccountID
	Topics []types.Hash
}

type Event_LessThan24Hours struct {
	Phase  types.Phase
	Last   types.U32
	Now    types.U32
	Topics []types.Hash
}
type Event_AlreadyFrozen struct {
	Phase  types.Phase
	Acc    types.AccountID
	Topics []types.Hash
}

type Event_MinerExit struct {
	Phase  types.Phase
	Acc    types.AccountID
	Topics []types.Hash
}

type Event_MinerClaim struct {
	Phase  types.Phase
	Acc    types.AccountID
	Topics []types.Hash
}

type Event_IncreaseCollateral struct {
	Phase   types.Phase
	Acc     types.AccountID
	Balance types.U128
	Topics  []types.Hash
}

type Event_Deposit struct {
	Phase   types.Phase
	Balance types.U128
	Topics  []types.Hash
}

type Event_Redeemed struct {
	Phase   types.Phase
	Acc     types.AccountID
	Deposit types.U128
	Topics  []types.Hash
}

type Event_Claimed struct {
	Phase   types.Phase
	Acc     types.AccountID
	Deposit types.U128
	Topics  []types.Hash
}

type Event_TimingStorageSpace struct {
	Phase  types.Phase
	Topics []types.Hash
}

type Event_UpdataBeneficiary struct {
	Phase  types.Phase
	Acc    types.AccountID
	New    types.AccountID
	Topics []types.Hash
}

type Event_UpdataIp struct {
	Phase  types.Phase
	Acc    types.AccountID
	Old    Ipv4Type
	New    Ipv4Type
	Topics []types.Hash
}

type Event_StartOfBufferPeriod struct {
	Phase  types.Phase
	When   types.U32
	Topics []types.Hash
}

type Event_EndOfBufferPeriod struct {
	Phase  types.Phase
	When   types.U32
	Topics []types.Hash
}

type Event_Receive struct {
	Phase  types.Phase
	Acc    types.AccountID
	Reward types.U128
	Topics []types.Hash
}

// ------------------------FileBank-------------------------------
type Event_DeleteFile struct {
	Phase  types.Phase
	Acc    types.AccountID
	Fileid FileHash
	Topics []types.Hash
}

type Event_BuySpace struct {
	Phase  types.Phase
	Acc    types.AccountID
	Size   types.U128
	Fee    types.U128
	Topics []types.Hash
}

type Event_FileUpload struct {
	Phase  types.Phase
	Acc    types.AccountID
	Topics []types.Hash
}

type Event_FileUpdate struct {
	Phase  types.Phase
	Acc    types.AccountID
	Fileid types.Bytes
	Topics []types.Hash
}

type Event_LeaseExpireIn24Hours struct {
	Phase  types.Phase
	Acc    types.AccountID
	Size   types.U128
	Topics []types.Hash
}

type Event_FileChangeState struct {
	Phase  types.Phase
	Acc    types.AccountID
	Fileid types.Bytes
	Topics []types.Hash
}

type Event_BuyFile struct {
	Phase  types.Phase
	Acc    types.AccountID
	Money  types.U128
	Fileid types.Bytes
	Topics []types.Hash
}

type Event_Purchased struct {
	Phase  types.Phase
	Acc    types.AccountID
	Fileid types.Bytes
	Topics []types.Hash
}

type Event_InsertFileSlice struct {
	Phase  types.Phase
	Fileid types.Bytes
	Topics []types.Hash
}

type Event_LeaseExpired struct {
	Phase  types.Phase
	Acc    types.AccountID
	Size   types.U128
	Topics []types.Hash
}

type Event_FillerUpload struct {
	Phase    types.Phase
	Acc      types.AccountID
	Filesize types.U64
	Topics   []types.Hash
}

type Event_UploadAutonomyFile struct {
	Phase     types.Phase
	User      types.AccountID
	File_hash FileHash
	File_size types.U64
	Topics    []types.Hash
}

type Event_ClearInvalidFile struct {
	Phase  types.Phase
	Acc    types.AccountID
	Fileid types.Bytes
	Topics []types.Hash
}

type Event_RecoverFile struct {
	Phase  types.Phase
	Acc    types.AccountID
	Fileid types.Bytes
	Topics []types.Hash
}

type Event_ReceiveSpace struct {
	Phase  types.Phase
	Acc    types.AccountID
	Topics []types.Hash
}

type Event_UploadDeclaration struct {
	Phase     types.Phase
	Acc       types.AccountID
	File_hash types.Bytes
	File_name types.Bytes
	Topics    []types.Hash
}
type Event_BuyPackage struct {
	Phase  types.Phase
	Acc    types.AccountID
	Size   types.U128
	Fee    types.U128
	Topics []types.Hash
}

type Event_PackageUpgrade struct {
	Phase    types.Phase
	Acc      types.AccountID
	Old_type types.U8
	New_type types.U8
	Topics   []types.Hash
}

type Event_PackageRenewal struct {
	Phase        types.Phase
	Acc          types.AccountID
	Package_type types.U8
	Topics       []types.Hash
}

// ------------------------FileMap--------------------------------
type Event_RegistrationScheduler struct {
	Phase  types.Phase
	Acc    types.AccountID
	Ip     types.Bytes
	Topics []types.Hash
}

type Event_UpdateScheduler struct {
	Phase    types.Phase
	Acc      types.AccountID
	Endpoint types.Bytes
	Topics   []types.Hash
}

// ------------------------other system---------------------------
type Event_UnsignedPhaseStarted struct {
	Phase  types.Phase
	Round  types.U32
	Topics []types.Hash
}

type Event_SignedPhaseStarted struct {
	Phase  types.Phase
	Round  types.U32
	Topics []types.Hash
}

type Event_SolutionStored struct {
	Phase            types.Phase
	Election_compute types.ElectionCompute
	Prev_ejected     types.Bool
	Topics           []types.Hash
}

type Event_Balances_Withdraw struct {
	Phase  types.Phase
	Who    types.AccountID
	Amount types.U128
	Topics []types.Hash
}

//**************************************************************

// All event types
type CessEventRecords struct {
	//system
	types.EventRecords
	//SegmentBook
	SegmentBook_PPBNoOnTimeSubmit     []Event_PPBNoOnTimeSubmit
	SegmentBook_PPDNoOnTimeSubmit     []Event_PPDNoOnTimeSubmit
	SegmentBook_SubmitReport          []Event_SubmitReport
	SegmentBook_VerifyProof           []Event_VerifyProof
	SegmentBook_OutstandingChallenges []Event_OutstandingChallenges
	//Sminer
	Sminer_Registered          []Event_Registered
	Sminer_TimedTask           []Event_TimedTask
	Sminer_DrawFaucetMoney     []Event_DrawFaucetMoney
	Sminer_FaucetTopUpMoney    []Event_FaucetTopUpMoney
	Sminer_LessThan24Hours     []Event_LessThan24Hours
	Sminer_AlreadyFrozen       []Event_AlreadyFrozen
	Sminer_MinerExit           []Event_MinerExit
	Sminer_MinerClaim          []Event_MinerClaim
	Sminer_IncreaseCollateral  []Event_IncreaseCollateral
	Sminer_Deposit             []Event_Deposit
	Sminer_Redeemed            []Event_Redeemed
	Sminer_Claimed             []Event_Claimed
	Sminer_TimingStorageSpace  []Event_TimingStorageSpace
	Sminer_UpdataBeneficiary   []Event_UpdataBeneficiary
	Sminer_UpdataIp            []Event_UpdataIp
	Sminer_StartOfBufferPeriod []Event_StartOfBufferPeriod
	Sminer_EndOfBufferPeriod   []Event_EndOfBufferPeriod
	Sminer_Receive             []Event_Receive
	//FileBank
	FileBank_DeleteFile           []Event_DeleteFile
	FileBank_BuySpace             []Event_BuySpace
	FileBank_FileUpload           []Event_FileUpload
	FileBank_FileUpdate           []Event_FileUpdate
	FileBank_LeaseExpireIn24Hours []Event_LeaseExpireIn24Hours
	FileBank_FileChangeState      []Event_FileChangeState
	FileBank_BuyFile              []Event_BuyFile
	FileBank_Purchased            []Event_Purchased
	FileBank_InsertFileSlice      []Event_InsertFileSlice
	FileBank_LeaseExpired         []Event_LeaseExpired
	FileBank_FillerUpload         []Event_FillerUpload
	FileBank_UploadAutonomyFile   []Event_UploadAutonomyFile
	FileBank_ClearInvalidFile     []Event_ClearInvalidFile
	FileBank_RecoverFile          []Event_RecoverFile
	FileBank_ReceiveSpace         []Event_ReceiveSpace
	FileBank_UploadDeclaration    []Event_UploadDeclaration
	FileBank_BuyPackage           []Event_BuyPackage
	FileBank_PackageUpgrade       []Event_PackageUpgrade
	FileBank_PackageRenewal       []Event_PackageRenewal
	//FileMap
	FileMap_RegistrationScheduler []Event_RegistrationScheduler
	FileMap_UpdateScheduler       []Event_UpdateScheduler
	//other system
	ElectionProviderMultiPhase_UnsignedPhaseStarted []Event_UnsignedPhaseStarted
	ElectionProviderMultiPhase_SignedPhaseStarted   []Event_SignedPhaseStarted
	ElectionProviderMultiPhase_SolutionStored       []Event_SolutionStored
	Balances_Withdraw                               []Event_Balances_Withdraw
}
