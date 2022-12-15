package configs

// type and version
const Version = "cess-bucket v0.6.0 dev"

const (
	// Name is the name of the program
	Name = "cess-bucket"
	// Description is the description of the program
	Description = "The storage miner implementation of the CESS platform"
	// NameSpace is the cached namespace
	NameSpace = "bucket"
)

const (
	// BaseDir is the base directory where data is stored
	BaseDir = NameSpace
	// Data directory
	LogDir    = "log"
	CacheDir  = "cache"
	FileDir   = "file"
	FillerDir = "filler"
)
