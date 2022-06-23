package chain

import "github.com/centrifuge/go-substrate-rpc-client/v4/types"

// cess chain module
const (
	State_Sminer      = "Sminer"
	State_SegmentBook = "SegmentBook"
	State_FileMap     = "FileMap"
	State_FileBank    = "FileBank"
)

// cess chain module method
const (
	Sminer_MinerItems          = "MinerItems"
	Sminer_MinerDetails        = "MinerDetails"
	SegmentBook_MinerHoldSlice = "MinerHoldSlice"
	SegmentBook_ChallengeMap   = "ChallengeMap"
	FileMap_SchedulerPuk       = "SchedulerPuk"
	FileBank_FillerMap         = "FillerMap"
	FileMap_SchedulerInfo      = "SchedulerMap"
	FileBank_InvalidFile       = "InvalidFile"
	Sminer_MinerColling        = "MinerColling"
)

// cess chain Transaction name
const (
	ChainTx_Sminer_Register          = "Sminer.regnstk"
	ChainTx_SegmentBook_IntentSubmit = "SegmentBook.intent_submit"
	ChainTx_Sminer_ExitMining        = "Sminer.exit_miner"
	ChainTx_Sminer_Withdraw          = "Sminer.withdraw"
	ChainTx_Sminer_Increase          = "Sminer.increase_collateral"
	SegmentBook_SubmitProve          = "SegmentBook.submit_challenge_prove"
	FileBank_ClearInvalidFile        = "FileBank.clear_invalid_file"
)

type MinerInfo struct {
	PeerId      types.U64
	IncomeAcc   types.AccountID
	Ip          types.Bytes
	Collaterals types.U128
	State       types.Bytes
	Power       types.U128
	Space       types.U128
	RewardInfo  RewardInfo
}

type RewardInfo struct {
	Total       types.U128
	Received    types.U128
	NotReceived types.U128
}

//---SchedulerInfo
type SchedulerInfo struct {
	Ip              types.Bytes
	Stash_user      types.AccountID
	Controller_user types.AccountID
}

type ChallengesInfo struct {
	File_size  types.U64
	Scan_size  types.U32
	File_type  types.U8
	Block_list types.Bytes
	File_id    types.Bytes
	//48 bit random number
	Random []types.Bytes
}

type Chain_SchedulerPuk struct {
	Spk           types.Bytes
	Shared_params types.Bytes
	Shared_g      types.Bytes
}

type SpaceFileInfo struct {
	MinerId   types.U64
	FileSize  types.U64
	BlockNum  types.U32
	ScanSize  types.U32
	Acc       types.AccountID
	BlockInfo []BlockInfo
	FileId    types.Bytes
	FileHash  types.Bytes
}
type BlockInfo struct {
	BlockIndex types.Bytes
	BlockSize  types.U32
}

//---Space Info
type UserSpaceInfo struct {
	PurchasedSpace types.U128
	UsedSpace      types.U128
	RemainingSpace types.U128
}

type ProveInfo struct {
	FileId  types.Bytes
	MinerId types.U64
	Cinfo   ChallengesInfo
	Mu      []types.Bytes
	Sigma   types.Bytes
}
