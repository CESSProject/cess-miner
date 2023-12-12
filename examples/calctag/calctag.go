package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	p2pgo "github.com/CESSProject/p2p-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const helpInfo string = `help information:
  --help      Show help information
  --port      Set listing port, default 4001
  --ip        Set public ip
  --cpu       Set cpu number
  --file      The file of calc tag
  --tee       Used tee addr
  --workspace Set work space, default .
`

func main() {
	var err error
	var n = node.New()
	var help bool
	var port int
	var cpu int
	var publicip string
	var workspace string
	var file string
	var tee string

	flag.BoolVar(&help, "help", false, "show help info")
	flag.IntVar(&port, "port", 4001, "listen port")
	flag.IntVar(&cpu, "cpu", 0, "use cpu cores")
	flag.StringVar(&publicip, "ip", "", "listen addr")
	flag.StringVar(&file, "file", "", "calc tag file")
	flag.StringVar(&tee, "tee", "", "tee addr")
	flag.StringVar(&workspace, "workspace", "", "work space")
	flag.Parse()

	if help {
		fmt.Printf("%v", helpInfo)
		os.Exit(0)
	}

	useCpu := configs.SysInit(uint8(cpu))
	log.Println("Use cpu: ", useCpu)

	if workspace == "" {
		workspace, _ = os.Getwd()
	}

	_, err = os.Stat(file)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	n.P2P, err = p2pgo.New(
		context.Background(),
		p2pgo.ListenPort(port),
		p2pgo.Workspace(workspace),
		//p2pgo.BootPeers(n.GetBootNodes()),
		p2pgo.PublicIpv4(publicip),
		p2pgo.ProtocolPrefix("/devnet"),
	)
	if err != nil {
		log.Println("[p2pgo.New] ", err)
		os.Exit(1)
	}
	buf, err := os.ReadFile(file)
	if err != nil {
		log.Println("[ReadFile] ", err)
		os.Exit(1)
	}
	hash, err := sutils.CalcSHA256(buf)
	if err != nil {
		log.Println("[CalcSHA256] ", err)
		os.Exit(1)
	}
	_, err = n.PoisServiceRequestGenTag(tee, buf, "", hash, "", time.Duration(time.Minute*10), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Println("[PoisServiceRequestGenTag] ", err)
		os.Exit(1)
	}

	log.Println("[PoisServiceRequestGenTag] suc")
}
