package chain

import "github.com/centrifuge/go-substrate-rpc-client/v4/types"

// **************************************************************
// custom event type
// **************************************************************

//------------------------SegmentBook----------------------------
type Event_ParamSet struct {
	Phase     types.Phase
	PeerId    types.U64
	SegmentId types.U64
	Random    types.U32
	Topics    []types.Hash
}

type Event_VPABCD_Submit_Verify struct {
	Phase     types.Phase
	PeerId    types.U64
	SegmentId types.U64
	Topics    []types.Hash
}

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

type Event_ChallengeProof struct {
	Phase  types.Phase
	PeerId types.U64
	Topics []types.Hash
}

type Event_VerifyProof struct {
	Phase  types.Phase
	PeerId types.U64
	Topics []types.Hash
}

//------------------------Sminer---------------------------------
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

//------------------------FileBank-------------------------------
type Event_DeleteFile struct {
	Phase  types.Phase
	Acc    types.AccountID
	Fileid types.Bytes
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

//------------------------FileMap--------------------------------
type Event_RegistrationScheduler struct {
	Phase  types.Phase
	Acc    types.AccountID
	Ip     types.Bytes
	Topics []types.Hash
}

//------------------------other system---------------------------
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
type MyEventRecords struct {
	//system
	types.EventRecords
	//SegmentBook
	SegmentBook_ParamSet          []Event_ParamSet
	SegmentBook_VPASubmitted      []Event_VPABCD_Submit_Verify
	SegmentBook_VPBSubmitted      []Event_VPABCD_Submit_Verify
	SegmentBook_VPCSubmitted      []Event_VPABCD_Submit_Verify
	SegmentBook_VPDSubmitted      []Event_VPABCD_Submit_Verify
	SegmentBook_VPAVerified       []Event_VPABCD_Submit_Verify
	SegmentBook_VPBVerified       []Event_VPABCD_Submit_Verify
	SegmentBook_VPCVerified       []Event_VPABCD_Submit_Verify
	SegmentBook_VPDVerified       []Event_VPABCD_Submit_Verify
	SegmentBook_PPBNoOnTimeSubmit []Event_PPBNoOnTimeSubmit
	SegmentBook_PPDNoOnTimeSubmit []Event_PPDNoOnTimeSubmit
	SegmentBook_ChallengeProof    []Event_ChallengeProof
	SegmentBook_VerifyProof       []Event_VerifyProof
	//Sminer
	Sminer_Registered         []Event_Registered
	Sminer_TimedTask          []Event_TimedTask
	Sminer_DrawFaucetMoney    []Event_DrawFaucetMoney
	Sminer_FaucetTopUpMoney   []Event_FaucetTopUpMoney
	Sminer_LessThan24Hours    []Event_LessThan24Hours
	Sminer_AlreadyFrozen      []Event_AlreadyFrozen
	Sminer_MinerExit          []Event_MinerExit
	Sminer_MinerClaim         []Event_MinerClaim
	Sminer_IncreaseCollateral []Event_IncreaseCollateral
	Sminer_Deposit            []Event_Deposit
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
	//FileMap
	FileMap_RegistrationScheduler []Event_RegistrationScheduler
	//other system
	ElectionProviderMultiPhase_UnsignedPhaseStarted []Event_UnsignedPhaseStarted
	ElectionProviderMultiPhase_SignedPhaseStarted   []Event_SignedPhaseStarted
	ElectionProviderMultiPhase_SolutionStored       []Event_SolutionStored
	Balances_Withdraw                               []Event_Balances_Withdraw
}
