package configs

type Confile struct {
	RpcAddr      string `toml:"RpcAddr"`
	MountedPath  string `toml:"MountedPath"`
	StorageSpace uint64 `toml:"StorageSpace"`
	ServiceIP    string `toml:"ServiceIP"`
	ServicePort  uint32 `toml:"ServicePort"`
	IncomeAcc    string `toml:"IncomeAcc"`
	SignatureAcc string `toml:"SignatureAcc"`
	DomainName   string `toml:"DomainName"`
}

var C = new(Confile)
var ConfFilePath string

const ConfigFile_Templete = `# The rpc address of the chain node
RpcAddr      = ""
# Path to the mounted disk where the data is saved
MountedPath  = ""
# Total space used to store files, the unit is GB
StorageSpace = 0
# The IP of the machine running the mining service
ServiceIP    = ""
# Port number monitored by the mining service
ServicePort  = 0
# The address of income account
IncomeAcc    = ""
# phrase of the signature account
SignatureAcc = ""
# If you don't have a public IP, you must set an access domain name
DomainName   = ""`
