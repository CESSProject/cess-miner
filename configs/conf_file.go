package configs

type MinerInfoOnChain struct {
	CessChain CessChain `toml:"CessChain"`
	MinerData MinerData `toml:"MinerData"`
}

type CessChain struct {
	ChainAddr string `toml:"ChainAddr"`
}

type MinerData struct {
	StorageSpace   uint64 `toml:"StorageSpace"`
	MountedPath    string `toml:"MountedPath"`
	ServiceAddr    string `toml:"ServiceAddr"`
	ServicePort    uint32 `toml:"ServicePort"`
	RevenuePuK     string `toml:"RevenuePuK"`
	TransactionPrK string `toml:"TransactionPrK"`
}

var Confile = new(MinerInfoOnChain)
var ConfFilePath string

const ConfigFile_Templete = `[CessChain]
# CESS chain address
ChainAddr = ""

[MinerData]
# Total space used to store files, the unit is GB
StorageSpace           = 0
# Path to the mounted disk where the data is saved
MountedPath            = "/"
# The IP address of the machine's public network used by the mining program
ServiceAddr          = ""
# Port number monitored by the mining program
ServicePort           = 15000
# Public key of revenue account
RevenuePuK    = ""
# Phrase words or seeds for transaction account
TransactionPrK  = ""`
