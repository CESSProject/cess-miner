package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	"github.com/CESSProject/cess-bucket/pkg/proof"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	p2pgo "github.com/CESSProject/p2p-go"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const helpInfo string = `help information:
  --help      Show help information
  --port      Set listing port, default 4001
  --ip        Set public ip
  --cpu       Set cpu number
  --tee       Used tee addr
  --pubkey    Set public key
  --workspace Put the file and the corresponding tag here, the tag file is saved as <filename>.tag
`

func main() {
	var err error
	var n = node.New()
	var help bool
	var port int
	var cpu int
	var publicip string
	var workspace string
	var key string
	var tee string

	flag.BoolVar(&help, "help", false, "show help info")
	flag.IntVar(&port, "port", 4001, "listen port")
	flag.IntVar(&cpu, "cpu", 0, "use cpu cores")
	flag.StringVar(&publicip, "ip", "", "listen addr")
	flag.StringVar(&tee, "tee", "", "tee addr")
	flag.StringVar(&key, "pubkey", "", "public key")
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

	pkey, err := hex.DecodeString(key)
	if err != nil {
		log.Println("[DecodeString key] ", err)
		os.Exit(1)
	}

	err = n.SetPublickey(pkey)
	if err != nil {
		log.Println("[SetPublickey] ", err)
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

	////////////////////////////////////
	var randomIndexList = []types.U32{26, 43, 1, 7, 2, 48, 65, 82, 160, 60}
	var randomList = []pattern.Random{
		{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200},
		{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200},
		{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200},
		{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200},
		{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200},
		{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200},
		{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200},
		{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200},
		{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200},
		{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200},
	}
	var sigma string
	var proveResponse proof.GenProofResponse
	var names = make([]string, 0)
	var us = make([]string, 0)
	var mus = make([]string, 0)
	var qslice = make([]proof.QElement, len(randomIndexList))
	for k, v := range randomIndexList {
		qslice[k].I = int64(v)
		var b = make([]byte, len(randomList[k]))
		for i := 0; i < len(randomList[k]); i++ {
			b[i] = byte(randomList[k][i])
		}
		qslice[k].V = new(big.Int).SetBytes(b).String()
	}

	serviceFiles, err := utils.DirFiles(workspace, 0)
	if err != nil {
		log.Println("DirFiles err: ", err)
		return
	}

	timeout := time.NewTicker(time.Duration(time.Minute))
	defer timeout.Stop()

	for {
		for i := 0; i < len(serviceFiles); i++ {
			if strings.Contains(serviceFiles[i], ".tag") {
				continue
			}
			log.Println("file: ", filepath.Base(serviceFiles[i]))
			serviceTagPath := serviceFiles[i] + ".tag"
			buf, err := os.ReadFile(serviceTagPath)
			if err != nil {
				log.Println("ReadFile: ", err)
				continue
			}
			var tag pb.Tag
			err = json.Unmarshal(buf, &tag)
			if err != nil {
				log.Println("Unmarshal tag: ", err)
				continue
			}

			matrix, _, err := proof.SplitByN(serviceFiles[i], int64(len(tag.T.Phi)))
			if err != nil {
				log.Println("SplitByN err: ", err)
				continue
			}

			proveResponseCh := n.GetPodr2Key().GenProof(qslice, nil, tag.T.Phi, matrix)
			timeout.Reset(time.Minute)
			select {
			case proveResponse = <-proveResponseCh:
			case <-timeout.C:
				proveResponse.StatueMsg.StatusCode = 0
			}

			if proveResponse.StatueMsg.StatusCode != proof.Success {
				log.Println("GenProof err code: ", proveResponse.StatueMsg.StatusCode)
				continue
			}

			sigmaTemp, ok := n.GetPodr2Key().AggrAppendProof(sigma, proveResponse.Sigma)
			if !ok {
				log.Println("AggrAppendProof failed")
				continue
			}
			sigma = sigmaTemp
			names = append(names, tag.T.Name)
			us = append(us, tag.T.U)
			mus = append(mus, proveResponse.MU)

			log.Println("Gen proof suc: ", filepath.Base(serviceFiles[i]))
		}

		// batch verify
		var randomIndexList_pb = make([]uint32, len(randomIndexList))
		for i := 0; i < len(randomIndexList); i++ {
			randomIndexList_pb[i] = uint32(randomIndexList[i])
		}
		var randomList_pb = make([][]byte, len(randomList))
		for i := 0; i < len(randomList); i++ {
			randomList_pb[i] = make([]byte, len(randomList[i]))
			for j := 0; j < len(randomList[i]); j++ {
				randomList_pb[i][j] = byte(randomList[i][j])
			}
		}

		var qslice_pb = &pb.RequestBatchVerify_Qslice{
			RandomIndexList: randomIndexList_pb,
			RandomList:      randomList_pb,
		}

		log.Println("req tee batch verify: ", tee)
		var batchVerifyParam = &pb.RequestBatchVerify_BatchVerifyParam{
			Names: names,
			Us:    us,
			Mus:   mus,
			Sigma: sigma,
		}

		n.Schal("info", fmt.Sprintf("req tee ip batch verify: %s", tee))
		var requestBatchVerify = &pb.RequestBatchVerify{
			AggProof:        batchVerifyParam,
			PeerId:          nil,
			MinerPbk:        nil,
			MinerPeerIdSign: nil,
			Qslices:         qslice_pb,
			USigs:           nil,
		}
		var dialOptions []grpc.DialOption
		if !strings.Contains(tee, "443") {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		} else {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
		}
		batchVerify, err := n.RequestBatchVerify(
			tee,
			requestBatchVerify,
			time.Duration(time.Minute*10),
			dialOptions,
			nil,
		)
		if err != nil {
			log.Println("RequestBatchVerify err: ", err)
			continue
		}
		log.Println("RequestBatchVerify result: ", batchVerify.BatchVerifyResult)
	}
}
