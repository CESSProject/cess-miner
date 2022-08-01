package configs

// type and version
const Version = "CESS-Bucket v0.4.3.220801.1626"

// rpc service and method
const (
	RpcService_Local               = "mservice"
	RpcService_Scheduler           = "wservice"
	RpcMethod_Scheduler_Writefile  = "writefile"
	RpcMethod_Scheduler_Readfile   = "readfile"
	RpcMethod_Scheduler_Space      = "space"
	RpcMethod_Scheduler_Spacefile  = "spacefile"
	RpcMethod_Scheduler_FillerBack = "fillerback"
	RpcMethod_Scheduler_State      = "state"
	RpcFileBuffer                  = 1024 * 1024 //1MB
)

// return code
const (
	//success
	Code_200 = 200
	//bad request
	Code_400 = 400
	//forbidden
	Code_403 = 403
	//not found
	Code_404 = 404
	//server internal error
	Code_500 = 500
	//The block was produced but the event was not resolved
	Code_600 = 600
)

const (
	Space_1GB          = 1024 * 1024 * 1024 // 1GB
	Space_1MB          = 1024 * 1024        // 1MB
	ByteSize_1Kb       = 1024               // 1KB
	TimeToWaitEvents_S = 20                 //The time to wait for the event, in seconds
	TokenAccuracy      = "000000000000"     //Unit precision of CESS coins
	ExitColling        = 57600              //blocks
	BlockSize          = 1024 * 1024        //1MB
	ScanBlockSize      = 512 * 1024         //512KB
)

// Miner info
// updated at runtime
var (
	// MinerId_S string = ""
	// MinerId_I uint64 = 0

	//data path
	BaseDir    = "bucket"
	LogfileDir = "/log"
	SpaceDir   = "space"
	FilesDir   = "files"
)
