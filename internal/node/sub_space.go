package node

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/internal/chain"
	. "github.com/CESSProject/cess-bucket/internal/logger"
	"github.com/CESSProject/cess-bucket/internal/pattern"
	api "github.com/CESSProject/cess-bucket/internal/proof/apiv1"
	"github.com/CESSProject/cess-bucket/tools"

	"github.com/CESSProject/go-keyring"
)

type kvpair struct {
	K string
	V int32
}

type baseFiller struct {
	MinerIp  []string `json:"minerIp"`
	FillerId string   `json:"fillerId"`
}

type RespSpaceInfo struct {
	FileId string `json:"fileId"`
	Token  string `json:"token"`
	T      api.FileTagT
	Sigmas [][]byte `json:"sigmas"`
}

var globalTransport *http.Transport

func init() {
	globalTransport = &http.Transport{
		DisableKeepAlives: true,
	}
}

// The task_SpaceManagement task is to automatically allocate hard disk space.
// It will help you use your allocated hard drive space, until the size you set in the config file is reached.
// It keeps running as a subtask.
func (n *Node) task_SpaceManagement(ch chan bool) {
	var (
		err            error
		msg            string
		availableSpace uint64
		tSpace         time.Time
	)
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
		ch <- true
	}()
	Flr.Info("-----> Start task_SpaceManagement <-----")

	availableSpace, err = calcAvailableSpace()
	if err != nil {
		Flr.Sugar().Errorf("calcAvailableSpace: %v", err)
	} else {
		tSpace = time.Now()
	}

	kr, _ := keyring.FromURI(configs.C.SignatureAcc, keyring.NetSubstrate{})
	time.Sleep(time.Second)
	for {
		if pattern.GetMinerState() != pattern.M_Positive {
			Flr.Sugar().Errorf("pattern.GetMinerState(): %s", pattern.GetMinerState())
			time.Sleep(time.Minute)
			continue
		}

		if time.Since(tSpace).Minutes() >= 10 {
			availableSpace, err = calcAvailableSpace()
			if err != nil {
				Flr.Sugar().Errorf("%v", err)
			} else {
				tSpace = time.Now()
			}
		}

		if availableSpace < uint64(configs.FillerSize) {
			Flr.Info("-------- Insufficient disk space --------")
			time.Sleep(time.Hour)
			continue
		}

		// Get all scheduler
		schds, err := chain.GetSchedulingNodes()
		if err != nil {
			Flr.Sugar().Errorf("GetSchedulingNodes: %v", err)
			time.Sleep(time.Second * 6)
			continue
		}

		tools.RandSlice(schds)

		for i := 0; i < len(schds); i++ {
			time.Sleep(time.Second * 3)
			wsURL := fmt.Sprintf("%d.%d.%d.%d:%d",
				schds[i].Ip.Value[0],
				schds[i].Ip.Value[1],
				schds[i].Ip.Value[2],
				schds[i].Ip.Value[3],
				schds[i].Ip.Port,
			)

			tcpAddr, err := net.ResolveTCPAddr("tcp", wsURL)
			if err != nil {
				Flr.Sugar().Infof("[%v] ", err)
				continue
			}
			dialer := net.Dialer{Timeout: time.Duration(time.Second * 5)}
			conn, err := dialer.Dial("tcp", tcpAddr.String())
			if err != nil {
				Flr.Sugar().Errorf("[%v] ", err)
				continue
			}
			conTcp, ok := conn.(*net.TCPConn)
			if !ok {
				Flr.Sugar().Errorf("[%v] ", err)
				continue
			}

			msg = tools.GetRandomcode(16)
			// sign message
			sign, err := kr.Sign(kr.SigningContext([]byte(msg)))
			if err != nil {
				conTcp.Close()
				conn = nil
				time.Sleep(time.Second)
				continue
			}
			Flr.Sugar().Infof("Request filler from [%v]", wsURL)
			srv := n.NewClient(NewTcp(conTcp), configs.SpaceDir, nil)
			err = srv.RecvFiller(pattern.GetMinerAcc(), []byte(msg), sign[:])
			if err != nil {
				Flr.Sugar().Errorf("[%v] ", err)
				continue
			}
			Flr.Sugar().Infof("Request filler from [%v] successfully ", wsURL)
		}
	}
}

// Calculate available space
func calcAvailableSpace() (uint64, error) {
	var err error

	usedSpace, err := tools.DirSize(configs.BaseDir)
	if err != nil {
		return 0, err
	}

	sspace := configs.C.StorageSpace * configs.SIZE_1GiB
	// mountP, err := tools.GetMountPathInfo(configs.C.MountedPath)
	// if err != nil {
	// 	return 0, err
	// }

	if sspace <= usedSpace {
		return 0, nil
	}

	// if mountP.Free > configs.SIZE_1MiB*100 {
	if usedSpace < sspace {
		return sspace - usedSpace, nil
	}
	// }
	return 0, nil
}
