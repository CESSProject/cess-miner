package node

import (
	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/CESSProject/cess-bucket/pkg/db"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/serve"
)

type Bucket interface {
	Run()
}

type Node struct {
	Ser       serve.IServer
	Cfile     confile.IConfile
	Chn       chain.IChain
	Logs      logger.ILog
	Cach      db.ICache
	FillerDir string
	FileDir   string
	TmpDir    string
}

// New is used to build a node instance
func New() *Node {
	return &Node{}
}

func (n *Node) Run() {
	// Start the subtask manager
	go n.CoroutineMgr()
	// Start Service
	n.Ser.Serve()
}
