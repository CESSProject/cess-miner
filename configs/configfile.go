package configs

type Confile struct {
	RpcAddr      string `toml:"RpcAddr"`
	MountedPath  string `toml:"MountedPath"`
	StorageSpace uint64 `toml:"StorageSpace"`
	ServiceAddr  string `toml:"ServiceAddr"`
	ServicePort  uint32 `toml:"ServicePort"`
	IncomeAcc    string `toml:"IncomeAcc"`
	SignaturePrk string `toml:"SignaturePrk"`
}

var C = new(Confile)
var ConfFilePath string

const ConfigFile_Templete = `# The rpc address of the chain node
RpcAddr      = ""
# Path to the mounted disk where the data is saved
MountedPath  = ""
# Total space used to store files, the unit is GB
StorageSpace = 1000
# The IP address of the machine's public network used by the mining program
ServiceAddr  = ""
# Port number monitored by the mining program
ServicePort  = 15001
# The address of income account
IncomeAcc    = ""
# phrase or seed of the signature account
SignaturePrk = ""`
