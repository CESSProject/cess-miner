package configs

// type and version
const Version = "CESS-Bucket_V0.4.0"

// cess chain module
const (
	ChainModule_Sminer      = "Sminer"
	ChainModule_SegmentBook = "SegmentBook"
)

// cess chain module method
const (
	ChainModule_Sminer_MinerItems          = "MinerItems"
	ChainModule_Sminer_MinerDetails        = "MinerDetails"
	ChainModule_SegmentBook_ConProofInfoA  = "ConProofInfoA"
	ChainModule_SegmentBook_ConProofInfoC  = "ConProofInfoC"
	ChainModule_SegmentBook_MinerHoldSlice = "MinerHoldSlice"
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
)

// rpc service and method
const (
	RpcService_Local              = "mservice"
	RpcService_Scheduler          = "wservice"
	RpcMethod_Scheduler_Writefile = "writefile"
	RpcMethod_Scheduler_Readfile  = "readfile"
)

// data segment properties
const (
	SegMentType_Idle      uint8  = 1
	SegMentType_Service   uint8  = 2
	SegMentType_8M        uint8  = 1
	SegMentType_8M_post   uint8  = 6
	SegMentType_512M      uint8  = 2
	SegMentType_512M_post uint8  = 7
	SegMentType_8M_S      string = "1"
	SegMentType_512M_S    string = "2"
)

const (
	Space_1GB          = 1073741824     // 1GB
	TimeToWaitEvents_S = 15             //The time to wait for the event, in seconds
	TokenAccuracy      = "000000000000" //Unit precision of CESS coins
)

// Miner info
// updated at runtime
var (
	MinerId_S        string = ""
	MinerId_I        uint64 = 0
	MinerDataPath    string = "cessminer_c"
	MinerUseSpace    uint64 = 0
	MinerServiceAddr string = ""
	MinerServicePort int    = 0
	//data path
	LogfilePathPrefix = "/log"
	SpaceDir          = "space"
	ServiceDir        = "service"
	Cache             = "cache"
	TmpltFileFolder   = "temp"
	TmpltFileName     = "template"
	PrivateKeyfile    = ".m_privateKey.pem"
	PublicKeyfile     = ".m_publicKey.pem"
)
