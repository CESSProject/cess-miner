package node

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/cache"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/sdk-go/core/client"
)

type Bucket interface {
	Run()
}

type Node struct {
	Cfg  confile.Confile
	Cli  client.Client
	Log  logger.Logger
	Cach cache.Cache
	//Handle   *gin.Engine
	FileDir  string
	TrackDir string
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
	log.Println("Service started successfully")

	for {
		time.Sleep(time.Second)
		// Accepts the next connection
		acceptTCP, err := listener.AcceptTCP()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				Out.Sugar().Error("[err] The port is closed and the service exits.")
				log.Println("[err] The port is closed and the service exits.")
				os.Exit(1)
			}
			continue
		}

		if !ConnectionFiltering(acceptTCP) {
			acceptTCP.Close()
			continue
		}

		remote = acceptTCP.RemoteAddr().String()
		Out.Sugar().Infof("received a conn: %v\n", remote)

		// Start the processing service of the new connection
		go New().NewServer(NewTcp(acceptTCP), configs.FilesDir).Start()
	}
}

func ConnectionFiltering(conn *net.TCPConn) bool {
	buf := make([]byte, len(HEAD_FILE))
	_, err := io.ReadAtLeast(conn, buf, len(HEAD_FILE))
	if err != nil {
		return false
	}
	return bytes.Equal(buf, HEAD_FILE) || bytes.Equal(buf, HEAD_FILLER)
}
