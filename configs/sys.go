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

// return code
const (
	Code_200 = 200
	Code_400 = 400
	Code_403 = 403
	Code_404 = 404
	Code_500 = 500
	//The block was produced but the event was not resolved
	Code_600 = 600
)

const (
	PrivateKeyfile = ".m_privateKey.pem"
	PublicKeyfile  = ".m_publicKey.pem"
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
	MinerUseSpace    uint64 = 0
	MinerServiceAddr string = ""
	MinerServicePort int    = 0
	//data path
	BaseDir    = "bucket"
	LogfileDir = "/log"
	SpaceDir   = "space"
	FilesDir   = "files"

	//
	Cache           = "cache"
	TmpltFileFolder = "temp"
	TmpltFileName   = "template"
)
