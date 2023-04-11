package configs

import "time"

const (
	// Name is the name of the program
	Name = "BUCKET"
	// version
	Version = "v0.6.0 sprint4 dev"
	// Description is the description of the program
	Description = "A mining program provided by cess platform for storage miners."
	// NameSpace is the cached namespace
	NameSpace = Name
)

const (
	// BlockInterval is the time interval for generating blocks, in seconds
	BlockInterval = time.Second * time.Duration(6)
	//
	DirPermission = 0755
)
