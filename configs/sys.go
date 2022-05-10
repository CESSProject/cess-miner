package configs

// type and version
const Version = "CESS-Bucket_V0.4.0"

// rpc service and method
const (
	RpcService_Local              = "mservice"
	RpcService_Scheduler          = "wservice"
	RpcMethod_Scheduler_Writefile = "writefile"
	RpcMethod_Scheduler_Readfile  = "readfile"
	RpcMethod_Scheduler_Space     = "space"
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
	Space_1MB          = 1024 * 1024    // 1MB
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
	FilesDir          = "files"
	Cache             = "cache"
	TmpltFileFolder   = "temp"
	TmpltFileName     = "template"
	PrivateKeyfile    = ".m_privateKey.pem"
	PublicKeyfile     = ".m_publicKey.pem"
)
