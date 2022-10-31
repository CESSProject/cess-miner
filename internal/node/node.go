package node

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	. "github.com/CESSProject/cess-bucket/internal/logger"
)

type Scheduler interface {
	Run()
}

type Node struct {
	Conn *ConMgr
	// Confile   configfile.Configfiler
	// Chain     chain.Chainer
	// Logs      logger.Logger
	// Cache     db.Cacher
	// FileDir   string
	// TagDir    string
	// FillerDir string
}

// New is used to build a node instance
func New() *Node {
	return &Node{}
}

func (n *Node) Run() {
	var (
		err    error
		remote string
	)
	go n.CoroutineMgr()
	// Get an address of TCP end point
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", configs.C.ServicePort))
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// Listen for TCP networks
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	for {
		// Accepts the next connection
		acceptTCP, err := listener.AcceptTCP()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				Out.Sugar().Error("[err] The port is closed and the service exits.")
				os.Exit(1)
			}
			Out.Sugar().Infof("Accept tcp: %v\n", err)
			continue
		}

		remote = acceptTCP.RemoteAddr().String()
		Out.Sugar().Infof("received a conn: %v\n", remote)

		// Start the processing service of the new connection
		go New().NewServer(NewTcp(acceptTCP), configs.FilesDir).Start()
		time.Sleep(time.Millisecond * 100)
	}
}
