package chain

import "github.com/centrifuge/go-substrate-rpc-client/v4/types"

// cess chain module
const (
	State_Sminer      = "Sminer"
	State_SegmentBook = "SegmentBook"
	State_FileMap     = "FileMap"
)

// cess chain module method
const (
	Sminer_MinerItems          = "MinerItems"
	Sminer_MinerDetails        = "MinerDetails"
	SegmentBook_ConProofInfoA  = "ConProofInfoA"
	SegmentBook_ConProofInfoC  = "ConProofInfoC"
	SegmentBook_MinerHoldSlice = "MinerHoldSlice"
	SegmentBook_ChallengeMap   = "ChallengeMap"
)

// cess chain Transaction name
const (
	ChainTx_Sminer_Register              = "Sminer.regnstk"
	ChainTx_SegmentBook_IntentSubmit     = "SegmentBook.intent_submit"
	ChainTx_SegmentBook_IntentSubmitPost = "SegmentBook.intent_submit_po_st"
	ChainTx_SegmentBook_SubmitToVpa      = "SegmentBook.submit_to_vpa"
	ChainTx_SegmentBook_SubmitToVpb      = "SegmentBook.submit_to_vpb"
	ChainTx_SegmentBook_SubmitToVpc      = "SegmentBook.submit_to_vpc"
	ChainTx_SegmentBook_SubmitToVpd      = "SegmentBook.submit_to_vpd"
	ChainTx_Sminer_ExitMining            = "Sminer.exit_miner"
	ChainTx_Sminer_Withdraw              = "Sminer.withdraw"
	ChainTx_Sminer_Increase              = "Sminer.increase_collateral"
	FileMap_SchedulerInfo                = "SchedulerMap"
)

type CessChain_MinerInfo struct {
	MinerInfo1 Chain_MinerItems
	MinerInfo2 CessChain_MinerInfo2
}

type Chain_MinerItems struct {
	Peerid      types.U64       `json:"peerid"`
	Beneficiary types.AccountID `json:"beneficiary"`
	ServiceAddr types.Bytes     `json:"ip"`
	Collaterals types.U128      `json:"collaterals"`
	Earnings    types.U128      `json:"earnings"`
	Locked      types.U128      `json:"locked"`
	State       types.Bytes     `json:"state"`
}

type CessChain_MinerInfo2 struct {
	Address                           types.AccountID `json:"address"`
	Beneficiary                       types.AccountID `json:"beneficiary"`
	Power                             types.U128      `json:"power"`
	Space                             types.U128      `json:"space"`
	Total_reward                      types.U128      `json:"total_reward"`
	Total_rewards_currently_available types.U128      `json:"total_rewards_currently_available"`
	Totald_not_receive                types.U128      `json:"totald_not_receive"`
	Collaterals                       types.U128      `json:"collaterals"`
}

type ParamInfo struct {
	Peer_id    types.U64 `json:"peer_id"`
	Segment_id types.U64 `json:"segment_id"`
	Rand       types.U32 `json:"rand"`
}

type IpostParaInfo struct {
	Peer_id    types.U64   `json:"peer_id"`
	Segment_id types.U64   `json:"segment_id"`
	Sealed_cid types.Bytes `json:"sealed_cid"`
	Size_type  types.U128  `json:"size_type"`
}

type UnsealedCidInfo struct {
	Peer_id    types.U64     `json:"peer_id"`
	Segment_id types.U64     `json:"segment_id"`
	Uncid      []types.Bytes `json:"uncid"`
	Rand       types.U32     `json:"rand"`
	Shardhash  types.Bytes   `json:"shardhash"`
}

type FpostParaInfo struct {
	Peer_id    types.U64     `json:"peer_id"`
	Segment_id types.U64     `json:"segment_id"`
	Sealed_cid []types.Bytes `json:"sealed_cid"`
	Hash       types.Bytes   `json:"hash"`
	Size_type  types.U128    `json:"size_type"`
}

//---SchedulerInfo
type SchedulerInfo struct {
	Ip              types.Bytes
	Stash_user      types.AccountID
	Controller_user types.AccountID
}
type ChallengesInfo struct {
	File_size  types.U64
	File_type  types.U8
	Block_list []types.U32
	File_id    types.Bytes
	//48 bit random number
	Random []types.Bytes
}
