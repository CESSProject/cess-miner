package configs

import "time"

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
	SIZE_1KiB          = 1024
	SIZE_1MiB          = SIZE_1KiB * 1024 // 1MB
	SIZE_1GiB          = SIZE_1MiB * 1024
	FillerSize         = 8 * SIZE_1MiB
	TimeToWaitEvents_S = 20             //The time to wait for the event, in seconds
	TokenAccuracy      = "000000000000" //Unit precision of CESS coins
	ExitColling        = 28800          //blocks
	BlockSize          = SIZE_1KiB * 2  //1MB
	//ScanBlockSize      = 512 * 1024     //512KB
	// the time to wait for the event, in seconds
	TimeToWaitEvents = time.Duration(time.Second * 15)
	//
	MaxProofData = 1
	//
	DefaultConfigFile = "./conf.yaml"
	//
	DirMode = 0644
)

const (
	// Maximum number of connections in the miner's certification space
	MAX_TCP_CONNECTION uint8 = 3
	// Tcp client connection interval
	TCP_Connection_Interval = time.Duration(time.Millisecond * 100)
	// Tcp message interval
	TCP_Message_Interval = time.Duration(time.Millisecond * 10)
	// Tcp short message waiting time
	TCP_Time_WaitNotification = time.Duration(time.Second * 10)
	// Tcp short message waiting time
	TCP_Time_WaitMsg = time.Duration(time.Minute)
	// Tcp short message waiting time
	TCP_FillerMessage_WaitingTime = time.Duration(time.Second * 150)
	// The slowest tcp transfers bytes per second
	TCP_Transmission_Slowest = SIZE_1KiB * 10
	// Number of tcp message caches
	TCP_Message_Send_Buffers = 10
	TCP_Message_Read_Buffers = 10
	//
	TCP_SendBuffer = 8192
	TCP_ReadBuffer = 12000
	TCP_TagBuffer  = 2012
	//
	Tcp_Dial_Timeout = time.Duration(time.Second * 5)
)

const (
	HELP_common = `Please check with the following help information:
    1.Check if the wallet balance is sufficient
    2.Block hash:`
	HELP_register = `    3.Check the Sminer_Registered transaction event result in the block hash above:
        If system.ExtrinsicFailed is prompted, it means failure;
        If system.ExtrinsicSuccess is prompted, it means success;`
	HELP_UpdateAddress = `    3.Check the Sminer_UpdataIp transaction event result in the block hash above:
        If system.ExtrinsicFailed is prompted, it means failure;
        If system.ExtrinsicSuccess is prompted, it means success;`
	HELP_UpdataBeneficiary = `    3.Check the Sminer_UpdataBeneficiary transaction event result in the block hash above:
        If system.ExtrinsicFailed is prompted, it means failure;
        If system.ExtrinsicSuccess is prompted, it means success;`
)

// Miner info
// updated at runtime
const (
	//data path
	DbDir    = "db"
	LogDir   = "log"
	SpaceDir = "space"
	FileDir  = "file"
	TmpDir   = "tmp"
)

var LogFiles = []string{
	"log",   //General log
	"panic", //Panic log
	"space",
	"report",
	"replace",
}
