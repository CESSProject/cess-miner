package configs

type MinerOnChain struct {
	CessChain  CessChain  `json:"cessChain"`
	MinerData  MinerData  `json:"minerData"`
	FileSystem FileSystem `json:"fileSystem"`
}

type CessChain struct {
	RpcAddr string `json:"rpcAddr"`
}

type MinerData struct {
	PledgeTokens uint64 `json:"pledgeTokens"`
	//RenewalTokens         uint64 `json:"renewalTokens"`
	StorageSpace          uint64 `json:"storageSpace"`
	MountedPath           string `json:"mountedPath"`
	ServiceIpAddr         string `json:"serviceIpAddr"`
	ServicePort           uint32 `json:"servicePort"`
	FilePort              uint32 `json:"filePort"`
	IncomeAccountPubkey   string `json:"incomeAccountPubkey"`
	IdAccountPhraseOrSeed string `json:"idAccountPhraseOrSeed"`
}

type FileSystem struct {
	DfsInstallPath string `json:"dfsInstallPath"`
}

var Confile = new(MinerOnChain)

const ConfigFile_Templete = `[cessChain]
# RPC address of CES public chain
rpcAddr = ""

[minerData]
# The cess coin that the miner needs to pledge when registering, the unit is TCESS.
pledgeTokens           = 2000
# Total space used to store files, the unit is GB.
storageSpace           = 1024
# Path to the mounted disk where the data is saved
mountedPath            = ""
# The IP address of the machine's public network used by the mining program.
serviceIpAddr          = ""
# Port number monitored by the mining program.
servicePort            = 15001
# Port number for file service monitoring.
filePort               = 15002
# Public key of income account.
incomeAccountPubkey    = ""
# Phrase words or seeds for identity accounts.
idAccountPhraseOrSeed  = ""

[fileSystem]
# Installation path of Fastdfs 
dfsInstallPath = ""`
