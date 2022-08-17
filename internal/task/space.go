package task

import (
	"cess-bucket/configs"
	"cess-bucket/internal/chain"
	. "cess-bucket/internal/logger"
	api "cess-bucket/internal/proof/apiv1"
	. "cess-bucket/internal/rpc"
	. "cess-bucket/internal/rpc/proto"
	"cess-bucket/tools"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

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

//The task_SpaceManagement task is to automatically allocate hard disk space.
//It will help you use your allocated hard drive space, until the size you set in the config file is reached.
//It keeps running as a subtask.
func task_SpaceManagement(ch chan bool) {
	var (
		err            error
		availableSpace uint64
		reconn         bool
		tSpace         time.Time
		reqspace       SpaceReq
		reqspacefile   SpaceFileReq
		tagInfo        api.TagInfo
		respspace      RespSpaceInfo
		client         *Client
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
		Flr.Sugar().Errorf("%v", err)
	} else {
		tSpace = time.Now()
	}

	reqspace.Publickey = configs.PublicKey

	kr, _ := keyring.FromURI(configs.C.SignatureAcc, keyring.NetSubstrate{})

	for {
		time.Sleep(time.Second)
		if client == nil || reconn {
			schds, err := chain.GetSchedulingNodes()
			if err != nil {
				Flr.Sugar().Errorf("%v", err)
				time.Sleep(time.Minute)
				continue
			}
			client, err = connectionScheduler(schds)
			if err != nil {
				Flr.Sugar().Errorf("--> All schedules unavailable")
				for i := 0; i < len(schds); i++ {
					Flr.Sugar().Errorf("   %v: %v", i, string(schds[i].Ip))
				}
				time.Sleep(time.Minute)
				continue
			}
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
			time.Sleep(time.Minute * time.Duration(tools.RandomInRange(10, 30)))
			continue
		}

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

		respCode, respBody, clo, err := WriteData(client, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_Space, req_b)
		reconn = clo
		if err != nil {
			Flr.Sugar().Errorf("%v", err)
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
			index := tools.RandomInRange(0, len(basefiller.MinerIp))
			var fillerurl string = "http://" + string(base58.Decode(basefiller.MinerIp[index])) + "/" + basefiller.FillerId
			var fillertagurl string = fillerurl + ".tag"
			fillerbody, err := getFiller(fillerurl)
			if err != nil {
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 6)))
				fillerbody, err = getFiller(fillerurl)
				if err != nil {
					Flr.Sugar().Errorf("%v", err)
					time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 10)))
					continue
				}
			}
			spacefilefullpath := filepath.Join(configs.SpaceDir, basefiller.FillerId)
			err = write_file(spacefilefullpath, fillerbody)
			if err != nil {
				os.Remove(spacefilefullpath)
				Flr.Sugar().Errorf("%v", err)
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 10)))
				continue
			}
			fillertagbody, err := getFiller(fillertagurl)
			if err != nil {
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 6)))
				fillertagbody, err = getFiller(fillertagurl)
				if err != nil {
					Flr.Sugar().Errorf("%v", err)
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
			req_back.Publickey = configs.PublicKey
			req_back.FileId = []byte(basefiller.FillerId)
			req_back.FileHash = []byte(hash)
			req_back_req, err := proto.Marshal(&req_back)
			if err != nil {
				Flr.Sugar().Errorf("%v", err)
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 10)))
				continue
			}

			_, _, clo, err := WriteData(client, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_FillerBack, req_back_req)
			reconn = clo
			if err != nil {
				if clo {
					schds, err := chain.GetSchedulingNodes()
					if err != nil {
						Flr.Sugar().Errorf("%v", err)
						time.Sleep(time.Minute)
						continue
					}
					client, err = connectionScheduler(schds)
					if err != nil {
						Flr.Sugar().Errorf("--> All schedules unavailable")
						for i := 0; i < len(schds); i++ {
							Flr.Sugar().Errorf("   %v: %v", i, string(schds[i].Ip))
						}
						time.Sleep(time.Minute)
						continue
					}
				}

				_, _, clo, err := WriteData(client, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_FillerBack, req_back_req)
				reconn = clo
				if err != nil {
					Flr.Sugar().Errorf(" %v", err)
					time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 10)))
				}
			}
			continue
		}

		if respCode != 200 {
			Flr.Sugar().Errorf("%v", respCode)
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(10, 30)))
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
			respCode, respBody, clo, err = WriteData(client, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_Spacefile, req_b)
			reconn = clo
			if err != nil {
				Flr.Sugar().Errorf(" %v", err)
				f.Close()
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

func connectionScheduler(schds []chain.SchedulerInfo) (*Client, error) {
	var (
		ok    bool
		err   error
		state = make(map[string]int32)
		cli   *Client
	)
	if len(schds) == 0 {
		return nil, errors.New("No scheduler service available")
	}
	var wsURL string
	for i := 0; i < len(schds); i++ {
		wsURL = "ws://" + string(base58.Decode(string(schds[i].Ip)))
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		cli, err = DialWebsocket(ctx, wsURL, "")
		if err != nil {
			continue
		}
		respCode, respBody, _, _ := WriteData(cli, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_State, nil)
		if respCode != 200 {
			cli.Close()
			continue
		}
		var resu int32
		if len(respBody) == 4 {
			resu += int32(respBody[0])
			resu = resu << 8
			resu += int32(respBody[1])
			resu = resu << 8
			resu += int32(respBody[2])
			resu = resu << 8
			resu += int32(respBody[3])
		}
		state[wsURL] = resu
		cli.Close()
	}
	tmpList := make([]kvpair, 0)
	for k, v := range state {
		tmpList = append(tmpList, kvpair{K: k, V: v})
	}
	sort.Slice(tmpList, func(i, j int) bool {
		return tmpList[i].V < tmpList[j].V
	})
	ok = false
	for _, pair := range tmpList {
		t := time.Duration(1)
		for i := 0; i < 3; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(t*time.Second))
			cli, err = DialWebsocket(ctx, pair.K, "")
			cancel()
			if err == nil {
				Flr.Sugar().Infof("Connect to %v", pair.K)
				ok = true
				break
			}
			t += 2
		}
		if ok {
			break
		}
	}
	return cli, err
}

func getFiller(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)

	client := &http.Client{
		Transport: globalTransport,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bo, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound || len(bo) < 20 {
		return nil, errors.New("Failed")
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
