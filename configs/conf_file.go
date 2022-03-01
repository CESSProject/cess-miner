package configs

type MinerOnChain struct {
	CessChain CessChain `json:"cessChain"`
	MinerData MinerData `json:"minerData"`
}

type CessChain struct {
	RpcAddr string `json:"rpcAddr"`
}

type MinerData struct {
	StorageSpace          uint64 `json:"storageSpace"`
	MountedPath           string `json:"mountedPath"`
	ServiceIpAddr         string `json:"serviceIpAddr"`
	ServicePort           uint32 `json:"servicePort"`
	IncomeAccountPubkey   string `json:"incomeAccountPubkey"`
	IdAccountPhraseOrSeed string `json:"idAccountPhraseOrSeed"`
}

var Confile = new(MinerOnChain)
var ConfFilePath string

const ConfigFile_Templete = `[cessChain]
# RPC address of CES public chain
rpcAddr = ""

[minerData]
# Total space used to store files, the unit is GB.
storageSpace           = 1024
# Path to the mounted disk where the data is saved
mountedPath            = ""
# The IP address of the machine's public network used by the mining program.
serviceIpAddr          = ""
# Port number monitored by the mining program.
servicePort            = 15001
# Public key of income account.
incomeAccountPubkey    = ""
# Phrase words or seeds for identity accounts.
idAccountPhraseOrSeed  = ""`
