package task

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/internal/chain"
	. "github.com/CESSProject/cess-bucket/internal/logger"
	"github.com/CESSProject/cess-bucket/internal/pattern"
	api "github.com/CESSProject/cess-bucket/internal/proof/apiv1"
	"github.com/CESSProject/cess-bucket/internal/rpc"
	. "github.com/CESSProject/cess-bucket/internal/rpc/proto"
	"github.com/CESSProject/cess-bucket/tools"

	"github.com/CESSProject/go-keyring"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
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
func task_SpaceManagement(ch chan bool) {
	var (
		err error
		//availableSpace uint64
		reconn bool
		//tSpace         time.Time
		reqspace     SpaceReq
		reqspacefile SpaceFileReq
		tagInfo      api.TagInfo
		respspace    RespSpaceInfo
		client       *rpc.Client
	)
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
		ch <- true
	}()
	Flr.Info("-----> Start task_SpaceManagement <-----")

	// availableSpace, err = calcAvailableSpace()
	// if err != nil {
	// 	Flr.Sugar().Errorf("calcAvailableSpace: %v", err)
	// } else {
	// 	tSpace = time.Now()
	// }

	reqspace.Publickey = pattern.GetMinerAcc()

	kr, _ := keyring.FromURI(configs.C.SignatureAcc, keyring.NetSubstrate{})

	for {
		if pattern.GetMinerState() != pattern.M_Positive {
			if pattern.GetMinerState() == pattern.M_Pending {
				time.Sleep(time.Second * 3)
				continue
			}
			time.Sleep(time.Minute * time.Duration(tools.RandomInRange(1, 5)))
			continue
		}

		time.Sleep(time.Second)
		if client == nil || reconn {
			if client != nil {
				client.Close()
			}
			// schds, err := chain.GetSchedulingNodes()
			// if err != nil {
			// 	Flr.Sugar().Errorf("%v", err)
			// 	time.Sleep(time.Minute)
			// 	continue
			// }
			//for i := 0; i < len(schds); i++ {
			//wsURL := "ws://" + string(base58.Decode(string(schds[i].Ip)))
			wsURL := "ws://47.242.144.118:15000"
			ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
			client, err = rpc.DialWebsocket(ctx, wsURL, "")
			if err != nil {
				continue
			}
			Flr.Sugar().Infof("Connected to %v", wsURL)
			//}

			// client, err = connectionScheduler(schds)
			// if err != nil {
			// 	Flr.Sugar().Errorf("--> All schedules unavailable")
			// 	for i := 0; i < len(schds); i++ {
			// 		Flr.Sugar().Errorf("   %v: %v", i, string(schds[i].Ip))
			// 	}
			// 	time.Sleep(time.Minute)
			// 	continue
			// }
		}

		//Flr.Sugar().Infof("Connected to %v", pattern.GetMinerRecentSche())

		// if time.Since(tSpace).Minutes() >= 10 {
		// 	availableSpace, err = calcAvailableSpace()
		// 	if err != nil {
		// 		Flr.Sugar().Errorf("%v", err)
		// 	} else {
		// 		tSpace = time.Now()
		// 	}
		// }

		// if availableSpace < uint64(8*configs.Space_1MB) {
		// 	Flr.Info("-------- Insufficient disk space --------")
		// 	time.Sleep(time.Minute * time.Duration(tools.RandomInRange(10, 30)))
		// 	continue
		// }

		// sign message
		msg := []byte(fmt.Sprintf("%v", tools.RandomInRange(100000, 999999)))
		sig, _ := kr.Sign(kr.SigningContext(msg))
		reqspace.Msg = msg
		reqspace.Sign = sig[:]

		req_b, err := proto.Marshal(&reqspace)
		if err != nil {
			Flr.Sugar().Errorf("%v", err)
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 30)))
			continue
		}

		respCode, respMsg, respBody, clo, err := rpc.WriteData(
			client,
			rpc.RpcService_Scheduler,
			rpc.RpcMethod_Scheduler_Space,
			time.Duration(time.Second*30),
			req_b,
		)
		reconn = clo
		if err != nil {
			fail_sche := pattern.GetMinerRecentSche()
			Flr.Sugar().Errorf("Space: %v, code:%v msg:%v err:%v", fail_sche, respCode, respMsg, err)
			pattern.AddToBlacklist(fail_sche)
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 30)))
			continue
		}

		if respCode == 201 {
			var basefiller baseFiller
			err = json.Unmarshal(respBody, &basefiller)
			if err != nil {
				Flr.Sugar().Errorf(" %v", err)
				continue
			}
			mip := string(base58.Decode(basefiller.MinerIp[0]))
			var fillerurl string = "http://" + mip + "/" + basefiller.FillerId
			var fillertagurl string = fillerurl + ".tag"
			if pattern.IsInBlacklist(mip) {
				continue
			}
			Flr.Sugar().Infof("%v", fillerurl)
			fillerbody, err := getFiller(fillerurl, time.Duration(time.Second*90))
			if err != nil {
				Flr.Sugar().Errorf("%v", err)
				pattern.AddToBlacklist(mip)
				//
				var req_back FillerBackReq
				req_back.Publickey = pattern.GetMinerAcc()
				req_back.FileId = []byte(basefiller.FillerId)
				req_back.FileHash = nil
				req_back_req, err := proto.Marshal(&req_back)
				if err != nil {
					Flr.Sugar().Errorf("%v", err)
					time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 10)))
					continue
				}

				_, _, _, reconn, err = rpc.WriteData(
					client,
					rpc.RpcService_Scheduler,
					rpc.RpcMethod_Scheduler_FillerFall,
					time.Duration(time.Second*30),
					req_back_req,
				)
				if err != nil {
					Flr.Sugar().Errorf("%v", err)
				}
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 6)))
				continue
			}
			spacefilefullpath := filepath.Join(configs.SpaceDir, basefiller.FillerId)
			err = write_file(spacefilefullpath, fillerbody)
			if err != nil {
				os.Remove(spacefilefullpath)
				Flr.Sugar().Errorf("%v", err)
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 10)))
				continue
			}
			fillertagbody, err := getFiller(fillertagurl, time.Duration(time.Second*20))
			if err != nil {
				if err != nil {
					Flr.Sugar().Errorf("%v", err)
					pattern.AddToBlacklist(mip)
					os.Remove(spacefilefullpath)
					time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 6)))
					continue
				}
			}

			tagfilename := basefiller.FillerId + ".tag"
			tagfilefullpath := filepath.Join(configs.SpaceDir, tagfilename)
			err = write_file(tagfilefullpath, fillertagbody)
			if err != nil {
				os.Remove(tagfilefullpath)
				Flr.Sugar().Errorf("%v", err)
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 10)))
				continue
			}

			hash, err := tools.CalcFileHash(spacefilefullpath)
			if err != nil {
				os.Remove(tagfilefullpath)
				os.Remove(spacefilefullpath)
				Flr.Sugar().Errorf(" %v", err)
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 10)))
				continue
			}

			//
			var req_back FillerBackReq
			req_back.Publickey = pattern.GetMinerAcc()
			req_back.FileId = []byte(basefiller.FillerId)
			req_back.FileHash = []byte(hash)
			req_back_req, err := proto.Marshal(&req_back)
			if err != nil {
				Flr.Sugar().Errorf("%v", err)
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 10)))
				continue
			}

			respCode, _, _, reconn, err = rpc.WriteData(
				client,
				rpc.RpcService_Scheduler,
				rpc.RpcMethod_Scheduler_FillerBack,
				time.Duration(time.Second*20),
				req_back_req,
			)
			if respCode != 200 {
				if reconn {
					client, err = ReConnect(pattern.GetMinerRecentSche())
					if err != nil {
						Flr.Sugar().Errorf("%v", err)
						os.Remove(tagfilefullpath)
						os.Remove(spacefilefullpath)
						fail_sche := pattern.GetMinerRecentSche()
						pattern.AddToBlacklist(fail_sche)
						time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 10)))
						continue
					}
					respCode, _, _, reconn, err = rpc.WriteData(
						client,
						rpc.RpcService_Scheduler,
						rpc.RpcMethod_Scheduler_FillerBack,
						time.Duration(time.Second*20),
						req_back_req,
					)
					if respCode != 200 {
						Flr.Sugar().Errorf("%v", err)
						os.Remove(tagfilefullpath)
						os.Remove(spacefilefullpath)
						fail_sche := pattern.GetMinerRecentSche()
						pattern.AddToBlacklist(fail_sche)
						time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 10)))
						continue
					}
				} else {
					reconn = true
					Flr.Sugar().Errorf("%v", err)
					os.Remove(tagfilefullpath)
					os.Remove(spacefilefullpath)
					fail_sche := pattern.GetMinerRecentSche()
					pattern.AddToBlacklist(fail_sche)
					continue
				}
			}
			Flr.Sugar().Infof("C-filler: %v", basefiller.FillerId)
			continue
		}

		if respCode != 200 {
			fail_sche := pattern.GetMinerRecentSche()
			Flr.Sugar().Errorf("Call %v, code:%v msg:%v", fail_sche, respCode, respMsg)
			pattern.AddToBlacklist(fail_sche)
			time.Sleep(time.Second * 3)
			reconn = true
			continue
		}

		err = json.Unmarshal(respBody, &respspace)
		if err != nil {
			Flr.Sugar().Errorf("%v", err)
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 10)))
			continue
		}

		//save space file tag
		tagfilename := respspace.FileId + ".tag"
		tagfilefullpath := filepath.Join(configs.SpaceDir, tagfilename)
		tagInfo.T = respspace.T
		tagInfo.Sigmas = respspace.Sigmas
		tag, err := json.Marshal(tagInfo)
		if err != nil {
			Flr.Sugar().Errorf("%v", err)
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 10)))
			continue
		}
		err = write_file(tagfilefullpath, tag)
		if err != nil {
			os.Remove(tagfilefullpath)
			Flr.Sugar().Errorf("%v", err)
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 10)))
			continue
		}

		spacefilefullpath := filepath.Join(configs.SpaceDir, respspace.FileId)
		f, err := os.OpenFile(spacefilefullpath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
		if err != nil {
			os.Remove(tagfilefullpath)
			Flr.Sugar().Errorf("%v", err)
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 10)))
			continue
		}
		reqspacefile.Token = respspace.Token
		reqspacefile.Publickey = pattern.GetMinerAcc()
		for i := 0; i < 17; i++ {
			reqspacefile.BlockIndex = uint32(i)
			req_b, err = proto.Marshal(&reqspacefile)
			if err != nil {
				Flr.Sugar().Errorf("%v", err)
				f.Close()
				os.Remove(tagfilefullpath)
				os.Remove(spacefilefullpath)
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 10)))
				break
			}
			respCode, respMsg, respBody, reconn, err = rpc.WriteData(
				client,
				rpc.RpcService_Scheduler,
				rpc.RpcMethod_Scheduler_Spacefile,
				time.Duration(time.Second*60),
				req_b,
			)
			if err != nil {
				f.Close()
				fail_sche := pattern.GetMinerRecentSche()
				Flr.Sugar().Errorf("Spacefile: %v, code:%v msg:%v err:%v", fail_sche, respCode, respMsg, err)
				pattern.AddToBlacklist(fail_sche)
				os.Remove(tagfilefullpath)
				os.Remove(spacefilefullpath)
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 10)))
				break
			}
			if i < 16 {
				f.Write(respBody)
				if i == 15 {
					f.Close()
				}
			} else {
				Flr.Sugar().Infof("B-filler: %v", respspace.FileId)
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
