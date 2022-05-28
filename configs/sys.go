package configs

// type and version
const Version = "CESS-Bucket_V0.4.0"

// rpc service and method
const (
	RpcService_Local              = "mservice"
	RpcService_Scheduler          = "wservice"
	RpcMethod_Scheduler_Writefile = "writefile"
	RpcMethod_Scheduler_Readfile  = "readfile"
	RpcMethod_Scheduler_Spacefile = "spacefile"
	RpcMethod_Scheduler_Spacetag  = "spacetag"
	RpcMethod_Scheduler_Fileback  = "fileback"
	RpcFileBuffer                 = 64 * 1024 //64KB
)

// return code
const (
	Code_200 = 200
	Code_400 = 400
	Code_403 = 403
	Code_404 = 404
	Code_500 = 500
	Code_600 = 600 //The block was produced but the event was not resolved
)

const (
	PrivateKeyfile = ".m_privateKey.pem"
	PublicKeyfile  = ".m_publicKey.pem"
)

const (
	Space_1GB          = 1073741824     // 1GB
	Space_1MB          = 1024 * 1024    // 1MB
	ByteSize_1Kb       = 1024           // 1KB
	TimeToWaitEvents_S = 15             //The time to wait for the event, in seconds
	TokenAccuracy      = "000000000000" //Unit precision of CESS coins
	NewTestAddr        = true
	ExitColling        = 1200
)

// Miner info
// updated at runtime
var (
	MinerId_S string = ""
	MinerId_I uint64 = 0

	//data path
	BaseDir    = "bucket"
	LogfileDir = "/log"
	SpaceDir   = "space"
	FilesDir   = "files"
)
