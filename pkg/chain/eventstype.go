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
type Event_SubmitReport struct {
	Phase  types.Phase
	Miner  types.AccountID
	Topics []types.Hash
}

type Event_ChallengeStart struct {
	Phase       types.Phase
	Total_power types.U128
	Reward      types.U128
	Topics      []types.Hash
}

type Event_ForceClearMiner struct {
	Phase  types.Phase
	Miner  types.AccountID
	Topics []types.Hash
}

type Event_PunishMiner struct {
	Phase  types.Phase
	Miner  types.AccountID
	Topics []types.Hash
}

// ------------------------Sminer---------------------------------
type Event_Registered struct {
	Phase      types.Phase
	Acc        types.AccountID
	StakingVal types.U128
	Topics     []types.Hash
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

type Event_MinerExit struct {
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

type Event_Receive struct {
	Phase  types.Phase
	Acc    types.AccountID
	Reward types.U128
	Topics []types.Hash
}

type Event_UpdateIasCert struct {
	Phase  types.Phase
	Acc    types.AccountID
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

type Event_LeaseExpireIn24Hours struct {
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

type Event_DeleteAutonomyFile struct {
	Phase     types.Phase
	User      types.AccountID
	File_hash FileHash
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

type Event_UploadDeal struct {
	Phase     types.Phase
	User      types.AccountID
	Assigned  types.AccountID
	File_hash FileHash
	Topics    []types.Hash
}

type Event_CreateBucket struct {
	Phase       types.Phase
	Acc         types.AccountID
	Owner       types.AccountID
	Bucket_name types.Bytes
	Topics      []types.Hash
}

type Event_DeleteBucket struct {
	Phase       types.Phase
	Acc         types.AccountID
	Owner       types.AccountID
	Bucket_name types.Bytes
	Topics      []types.Hash
}

type Event_UploadDealFailed struct {
	Phase     types.Phase
	User      types.AccountID
	File_hash FileHash
	Topics    []types.Hash
}

type Event_ReassignedDeal struct {
	Phase     types.Phase
	User      types.AccountID
	Assigned  types.AccountID
	File_hash FileHash
	Topics    []types.Hash
}

type Event_FlyUpload struct {
	Phase     types.Phase
	Operator  types.AccountID
	Owner     types.AccountID
	File_hash FileHash
	Topics    []types.Hash
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

// ------------------------oss---------------------------
type Event_OssRegister struct {
	Phase    types.Phase
	Acc      types.AccountID
	Endpoint Ipv4Type_Query
	Topics   []types.Hash
}

type Event_OssUpdate struct {
	Phase        types.Phase
	Acc          types.AccountID
	New_endpoint Ipv4Type_Query
	Topics       []types.Hash
}

type Event_OssDestroy struct {
	Phase  types.Phase
	Acc    types.AccountID
	Topics []types.Hash
}

type Event_Authorize struct {
	Phase    types.Phase
	Acc      types.AccountID
	operator types.AccountID
	Topics   []types.Hash
}

type Event_CancelAuthorize struct {
	Phase  types.Phase
	Acc    types.AccountID
	Topics []types.Hash
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
	SegmentBook_SubmitReport    []Event_SubmitReport
	SegmentBook_ChallengeStart  []Event_ChallengeStart
	SegmentBook_ForceClearMiner []Event_ForceClearMiner
	SegmentBook_PunishMiner     []Event_PunishMiner
	//Sminer
	Sminer_Registered         []Event_Registered
	Sminer_DrawFaucetMoney    []Event_DrawFaucetMoney
	Sminer_FaucetTopUpMoney   []Event_FaucetTopUpMoney
	Sminer_LessThan24Hours    []Event_LessThan24Hours
	Sminer_IncreaseCollateral []Event_IncreaseCollateral
	Sminer_Deposit            []Event_Deposit
	Sminer_TimingStorageSpace []Event_TimingStorageSpace
	Sminer_UpdataBeneficiary  []Event_UpdataBeneficiary
	Sminer_UpdataIp           []Event_UpdataIp
	Sminer_Receive            []Event_Receive
	Sminer_UpdateIasCert      []Event_UpdateIasCert
	//FileBank
	FileBank_DeleteFile           []Event_DeleteFile
	FileBank_BuySpace             []Event_BuySpace
	FileBank_FileUpload           []Event_FileUpload
	FileBank_LeaseExpireIn24Hours []Event_LeaseExpireIn24Hours
	FileBank_FillerUpload         []Event_FillerUpload
	FileBank_UploadAutonomyFile   []Event_UploadAutonomyFile
	FileBank_DeleteAutonomyFile   []Event_DeleteAutonomyFile
	FileBank_ClearInvalidFile     []Event_ClearInvalidFile
	FileBank_RecoverFile          []Event_RecoverFile
	FileBank_UploadDeal           []Event_UploadDeal
	FileBank_MinerExit            []Event_MinerExit
	FileBank_CreateBucket         []Event_CreateBucket
	FileBank_DeleteBucket         []Event_DeleteBucket
	FileBank_UploadDealFailed     []Event_UploadDealFailed
	FileBank_ReassignedDeal       []Event_ReassignedDeal
	FileBank_FlyUpload            []Event_FlyUpload
	//FileMap
	FileMap_RegistrationScheduler []Event_RegistrationScheduler
	FileMap_UpdateScheduler       []Event_UpdateScheduler
	//OSS
	Oss_OssRegister     []Event_OssRegister
	Oss_OssUpdate       []Event_OssUpdate
	Oss_OssDestroy      []Event_OssDestroy
	Oss_Authorize       []Event_Authorize
	Oss_CancelAuthorize []Event_CancelAuthorize
	//other system
	ElectionProviderMultiPhase_UnsignedPhaseStarted []Event_UnsignedPhaseStarted
	ElectionProviderMultiPhase_SignedPhaseStarted   []Event_SignedPhaseStarted
	ElectionProviderMultiPhase_SolutionStored       []Event_SolutionStored
	Balances_Withdraw                               []Event_Balances_Withdraw
}
