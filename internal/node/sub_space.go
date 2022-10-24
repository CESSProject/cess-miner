package node

import (
	"context"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/internal/chain"
	. "github.com/CESSProject/cess-bucket/internal/logger"
	"github.com/CESSProject/cess-bucket/internal/pattern"
	api "github.com/CESSProject/cess-bucket/internal/proof/apiv1"
	"github.com/CESSProject/cess-bucket/internal/rpc"
	"github.com/CESSProject/cess-bucket/tools"

	"github.com/CESSProject/go-keyring"
	"github.com/pkg/errors"
	"storj.io/common/base58"
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

	for {
		if pattern.GetMinerState() != pattern.M_Positive {
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

		if availableSpace < uint64(8*configs.Space_1MB) {
			Flr.Info("-------- Insufficient disk space --------")
			time.Sleep(time.Hour)
			continue
		}

		// Get all scheduler
		schds, err := chain.GetSchedulingNodes()
		if err != nil {
			Uld.Sugar().Infof("[%v] ", err)
			return
		}

		tools.RandSlice(schds)

		for i := 0; i < len(schds); i++ {
			time.Sleep(time.Second * 3)
			wsURL := string(base58.Decode(string(schds[i].Ip)))
			tcpAddr, err := net.ResolveTCPAddr("tcp", wsURL)
			if err != nil {
				Uld.Sugar().Infof("[%v] ", err)
				continue
			}

			conTcp, err := net.DialTCP("tcp", nil, tcpAddr)
			if err != nil {
				Uld.Sugar().Infof("[%v] ", err)
				continue
			}

			msg = tools.GetRandomcode(16)
			// sign message
			sign, err := kr.Sign(kr.SigningContext([]byte(msg)))
			if err != nil {
				conTcp.Close()
				time.Sleep(time.Second)
				continue
			}

			srv := n.NewClient(NewTcp(conTcp), configs.SpaceDir, nil)
			err = srv.RecvFiller(pattern.GetMinerAcc(), []byte(msg), sign[:])
			if err != nil {
				Uld.Sugar().Infof("[%v] ", err)
				continue
			}
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

	sspace := configs.C.StorageSpace * configs.Space_1GB
	mountP, err := tools.GetMountPathInfo(configs.C.MountedPath)
	if err != nil {
		return 0, err
	}

	if sspace <= usedSpace {
		return 0, nil
	}

	if mountP.Free > configs.Space_1MB*100 {
		if usedSpace < sspace {
			return sspace - usedSpace, nil
		}
	}
	return 0, nil
}

func connectionScheduler(schds []chain.SchedulerInfo) (*rpc.Client, error) {
	var (
		err   error
		resu  int32
		state = make(map[string]int32)
		cli   *rpc.Client
	)
	if len(schds) == 0 {
		return nil, errors.New("No scheduler service available")
	}
	var wsURL string
	for i := 0; i < len(schds); i++ {
		wsURL = "ws://" + string(base58.Decode(string(schds[i].Ip)))
		if pattern.IsInBlacklist(wsURL) {
			continue
		}
		accountinfo, err := chain.GetAccountInfo(schds[i].Controller_user[:])
		if err != nil {
			if err.Error() == chain.ERR_Empty {
				pattern.AddToBlacklist(wsURL)
			}
			continue
		}
		if accountinfo.Data.Free.CmpAbs(new(big.Int).SetUint64(2000000000000)) == -1 {
			pattern.AddToBlacklist(wsURL)
			continue
		}
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		cli, err = rpc.DialWebsocket(ctx, wsURL, "")
		if err != nil {
			continue
		}
		respCode, _, respBody, _, _ := rpc.WriteData(
			cli,
			rpc.RpcService_Scheduler,
			rpc.RpcMethod_Scheduler_State,
			time.Duration(time.Second*10),
			nil,
		)
		if respCode != 200 {
			cli.Close()
			continue
		}
		resu = 0
		if len(respBody) == 4 {
			resu += int32(respBody[0])
			resu = resu << 8
			resu += int32(respBody[1])
			resu = resu << 8
			resu += int32(respBody[2])
			resu = resu << 8
			resu += int32(respBody[3])
		}
		if resu < 10 {
			pattern.SetMinerRecentSche(wsURL)
			return cli, nil
		}
		state[wsURL] = resu
		cli.Close()
	}
	var ok = false
	var threshold int32 = 10
	for !ok {
		for k, v := range state {
			if (threshold-10) <= v && v < threshold {
				ctx, _ := context.WithTimeout(context.Background(), time.Duration(5*time.Second))
				cli, err = rpc.DialWebsocket(ctx, k, "")
				if err == nil {
					pattern.SetMinerRecentSche(k)
					ok = true
					break
				}
			}
		}
		if !ok {
			threshold += 5
		} else {
			break
		}
		if threshold >= 100 {
			return nil, errors.New("schedule busy")
		}
	}
	return cli, err
}

func getFiller(url string, t time.Duration) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)

	client := &http.Client{
		Timeout:   t,
		Transport: globalTransport,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Failed")
	}

	bo, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bo, nil
}

func write_file(fpath string, data []byte) error {
	ft, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer ft.Close()
	_, err = ft.Write(data)
	if err != nil {
		return err
	}
	return ft.Sync()
}

func ReConnect(url string) (*rpc.Client, error) {
	var (
		err error
		cli *rpc.Client
	)
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(5*time.Second))
	cli, err = rpc.DialWebsocket(ctx, url, "")
	if err != nil {
		Flr.Sugar().Infof("Reconnect to %v", url)
		return nil, errors.New("Failed")
	}
	return cli, nil
}
