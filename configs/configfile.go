package configs

type Confile struct {
	RpcAddr      string `toml:"Rpc_Address"`
	MountedPath  string `toml:"Mounted_Path"`
	StorageSpace uint64 `toml:"Storage_Space"`
	ServiceAddr  string `toml:"Service_IP"`
	ServicePort  uint32 `toml:"Service_Port"`
	IncomeAcc    string `toml:"Income_Acc"`
	SignaturePrk string `toml:"Signature_Acc"`
	DomainName   string `toml:"Domain_Name"`
}

var C = new(Confile)
var ConfFilePath string

const ConfigFile_Templete = `# The rpc address of the chain node
Rpc_Address      = ""
# Path to the mounted disk where the data is saved
Mounted_Path  = ""
# Total space used to store files, the unit is GB
Storage_Space = 0
# The IP of the machine running the mining service
Service_IP  = ""
# Port number monitored by the mining service
Service_Port  = 0
# The address of income account
Income_Acc    = ""
# phrase of the signature account
Signature_Acc = ""
# If you don't have a public IP, you must set an access domain name
Domain_Name   = ""`
