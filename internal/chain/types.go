package chain

import "github.com/centrifuge/go-substrate-rpc-client/v4/types"

// DOT is "." character
const DOT = "."

// Pallets
const (
	// SMINER is a module about storage miners
	SMINER = "Sminer"
	// AUDIT is a module on data challenges
	AUDIT = "Audit"
	// TEEWOEKER is a module about TEE
	TEEWORKER = "TeeWorker"
	// FILEBANK is a module about data metadata, bucket info, etc.
	FILEBANK = "FileBank"
	// STORAGEHD is a module about storage space
	STORAGEHD = "StorageHandler"
)

// Chain state
const (
	// SMINER
	ALLMINER    = "AllMiner"
	MINERITEMS  = "MinerItems"
	MINERLOCKIN = "MinerLockIn"

	// AUDIT
	CHALLENGEMAP = "ChallengeMap"

	// TEEWORKER
	SCHEDULERMAP = "SchedulerMap"

	// FILEBANK
	INVALIDFILE = "InvalidFile"
)

// Extrinsics
const (
	// SMINER
	TX_SMINER_REG         = SMINER + DOT + "regnstk"
	TX_SMINER_EXIT        = SMINER + DOT + "exit_miner"
	TX_SMINER_WITHDRAW    = SMINER + DOT + "withdraw"
	TX_SMINER_UPDATEADDR  = SMINER + DOT + "update_ip"
	TX_SMINER_UPDATEACC   = SMINER + DOT + "update_beneficiary"
	TX_SMINER_PLEDGETOKEN = SMINER + DOT + "increase_collateral"

	// AUDIT
	TX_AUDIT_REPORTPROOF = AUDIT + DOT + "submit_challenge_prove"

	// FILEBANK
	TX_FILEBANK_DELFILE      = FILEBANK + DOT + "clear_invalid_file"
	TX_FILEBANK_DELALLFILLER = FILEBANK + DOT + "clear_all_filler"
)

type FileHash [64]types.U8
type FileBlockId [68]types.U8

// Storage Miner Information Structure
type MinerInfo struct {
	PeerId      types.U64
	IncomeAcc   types.AccountID
	Ip          Ipv4Type
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

// Scheduling Node Information Structure
type SchedulerInfo struct {
	Ip              Ipv4Type
	Stash_user      types.AccountID
	Controller_user types.AccountID
}

// Challenge information structure
type ChallengesInfo struct {
	File_size  types.U64
	File_type  types.U8
	Block_list []types.U32
	File_id    FileHash
	Shard_id   FileBlockId
	Random     []types.Bytes
}

// Scheduling node public key information structure
type Chain_SchedulerPuk struct {
	Spk           [128]types.U8
	Shared_params types.Bytes
	Shared_g      [128]types.U8
}

// Proof information structure
type ProveInfo struct {
	FileId      FileHash
	MinerAcc    types.AccountID
	Cinfo       ChallengesInfo
	U           types.Bytes
	Mu          types.Bytes
	Sigma       types.Bytes
	Omega       types.Bytes
	SigRootHash types.Bytes
	HashMi      []types.Bytes
}

type IpAddress struct {
	IPv4 Ipv4Type
	IPv6 Ipv6Type
}
type Ipv4Type struct {
	Index types.U8
	Value [4]types.U8
	Port  types.U16
}
type Ipv6Type struct {
	Index types.U8
	Value [8]types.U16
	Port  types.U16
}

const (
	ERR_Failed  = "Failed"
	ERR_Timeout = "Timeout"
	ERR_Empty   = "Empty"
)
