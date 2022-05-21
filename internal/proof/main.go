package proof

import (
	"cess-bucket/configs"
	"cess-bucket/internal/chain"
	"cess-bucket/internal/encryption"
	. "cess-bucket/internal/logger"
	api "cess-bucket/internal/proof/apiv1"
	"cess-bucket/internal/pt"
	"cess-bucket/internal/rpc"
	p "cess-bucket/internal/rpc/proto"
	"cess-bucket/tools"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"storj.io/common/base58"
)

type RespSpacetagInfo struct {
	FileId string       `json:"fileId"`
	T      api.FileTagT `json:"file_tag_t"`
	Sigmas [][]byte     `json:"sigmas"`
}

type RespSpacefileInfo struct {
	FileId     string `json:"fileId"`
	FileHash   string `json:"fileHash"`
	BlockTotal uint32 `json:"blockTotal"`
	BlockIndex uint32 `json:"blockIndex"`
	BlockData  []byte `json:"blockData"`
}

// Start the proof module
func Proof_Main() {
	var (
		channel_1 = make(chan bool, 1)
		channel_2 = make(chan bool, 1)
		channel_3 = make(chan bool, 1)
	)
	go task_SpaceManagement(channel_1)
	go task_HandlingChallenges(channel_2)
	go task_RemoveInvalidFiles(channel_3)
	for {
		select {
		case <-channel_1:
			go task_SpaceManagement(channel_1)
		case <-channel_2:
			go task_HandlingChallenges(channel_2)
		case <-channel_3:
			go task_RemoveInvalidFiles(channel_3)
		}
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
		allsuc         bool
		addr           string
		filehash       string
		basedir        string
		tSpace         time.Time
		req            p.SpaceFileReq
		tagreq         p.SpaceTagReq
		fileback       p.FileBackReq
		tagInfo        pt.TagInfo
		respspacefile  RespSpacefileInfo
		client         *rpc.Client
	)
	defer func() {
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
		ch <- true
	}()
	Out.Info(">>>Start task_SpaceManagement task<<<")

	//Parse the account address through the phrase
	addr, err = chain.GetAddressFromPrk(configs.C.SignaturePrk, tools.SubstratePrefix)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	//Read RSA private key
	prk, err := encryption.GetRSAPrivateKey(filepath.Join(configs.BaseDir, configs.PrivateKeyfile))
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	//Calculate the signature
	sign, err := encryption.CalcSign([]byte(addr), prk)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	availableSpace, err = calcAvailableSpace()
	if err != nil {
		Err.Sugar().Errorf("[C%v] %v", configs.MinerId_S, err)
	} else {
		tSpace = time.Now()
	}

	req.Acc = addr
	req.Sign = sign
	tagreq.Acc = addr
	tagreq.Sign = sign
	fileback.Acc = addr
	fileback.Sign = sign
	allsuc = true

	for {
		if !allsuc {
			allsuc = true
			os.RemoveAll(basedir)
		}

		if client == nil || reconn {
			schds, _ := chain.GetSchedulerInfo()
			client, err = connectionScheduler(schds)
			if err != nil {
				Err.Sugar().Errorf("-->Err: All schedules unavailable")
				for i := 0; i < len(schds); i++ {
					Err.Sugar().Errorf("        %v", string(schds[i].Ip))
				}
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(10, 30)))
				continue
			}
		}

		if time.Since(tSpace).Minutes() >= 10 {
			availableSpace, err = calcAvailableSpace()
			if err != nil {
				Err.Sugar().Errorf("[C%v] %v", configs.MinerId_S, err)
			} else {
				tSpace = time.Now()
			}
		}

		if availableSpace >= uint64(8*configs.Space_1MB) {
			req.SizeMb = 8
			req.Fileid = ""
			req.BlockIndex = 0
			req_b, err := proto.Marshal(&req)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}

			respCode, respBody, clo, err := rpc.WriteData(client, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_Spacefile, req_b)
			reconn = clo
			if err != nil || respCode != configs.Code_200 {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}

			err = json.Unmarshal(respBody, &respspacefile)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}
			spacefilename := respspacefile.FileId + ".space"
			basedir = filepath.Join(configs.SpaceDir, respspacefile.FileId)
			_, err = os.Stat(basedir)
			if err != nil {
				err = os.MkdirAll(basedir, os.ModeDir)
				if err != nil {
					Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
					continue
				}
			}
			spacefilefullpath := filepath.Join(basedir, spacefilename)
			spacefile, err := os.OpenFile(spacefilefullpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_APPEND, os.ModePerm)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}
			allsuc = false
			req.Fileid = respspacefile.FileId
			spacefile.Write(respspacefile.BlockData)
			for j := 1; j < int(respspacefile.BlockTotal); j++ {
				req.BlockIndex = uint32(j)
				req_b, err = proto.Marshal(&req)
				if err != nil {
					Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
					spacefile.Close()
					os.Remove(spacefilefullpath)
					break
				}
				respCode, respBody, clo, err = rpc.WriteData(client, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_Spacefile, req_b)
				reconn = clo
				if err != nil || respCode != configs.Code_200 {
					Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
					spacefile.Close()
					os.Remove(spacefilefullpath)
					break
				}
				var respspacefilei RespSpacefileInfo
				err = json.Unmarshal(respBody, &respspacefilei)
				if err != nil {
					Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
					spacefile.Close()
					os.Remove(spacefilefullpath)
					break
				}
				if respspacefilei.FileHash != "" {
					filehash = respspacefilei.FileHash
				}
				spacefile.Write(respspacefilei.BlockData)
			}
			_, err = os.Stat(spacefilefullpath)
			if err != nil {
				continue
			}
			err = spacefile.Sync()
			if err != nil {
				spacefile.Close()
				os.Remove(spacefilefullpath)
				continue
			}
			spacefile.Close()
			hash, err := tools.CalcFileHash(spacefilefullpath)
			if err != nil {
				os.Remove(spacefilefullpath)
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}

			if filehash != hash {
				os.Remove(spacefilefullpath)
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}

			tagreq.Fileid = respspacefile.FileId

			req_b, err = proto.Marshal(&tagreq)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}
			respCode, respBody, clo, err = rpc.WriteData(client, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_Spacetag, req_b)
			reconn = clo
			if err != nil || respCode != configs.Code_200 {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}
			var respInfo RespSpacetagInfo
			err = json.Unmarshal(respBody, &respInfo)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}

			tagfilename := respInfo.FileId + ".tag"
			tagfilefullpath := filepath.Join(basedir, tagfilename)
			tagInfo.T = respInfo.T
			tagInfo.Sigmas = respInfo.Sigmas
			ft, err := os.OpenFile(tagfilefullpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}
			tag, err := json.Marshal(tagInfo)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				ft.Close()
				continue
			}
			ft.Write(tag)
			err = ft.Sync()
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				ft.Close()
				continue
			}
			ft.Close()

			fileback.Fileid = respspacefile.FileId
			fileback.Filehash = hash

			req_b, err = proto.Marshal(&fileback)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}
			respCode, respBody, clo, err = rpc.WriteData(client, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_Fileback, req_b)
			reconn = clo
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}
			if respCode == configs.Code_200 || len(respBody) > 0 {
				allsuc = true
				Out.Sugar().Infof(" %v store and upload to the chain successfully", respspacefile.FileId)
				continue
			}
			go func(path, fileid string) {
				var flag = false
				for i := 0; i < 3; i++ {
					time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
					_, code, _ := chain.GetFillerInfo(types.U64(configs.MinerId_I), fileid)
					if code == configs.Code_200 {
						flag = true
						return
					}
				}
				if !flag {
					os.RemoveAll(path)
					Err.Sugar().Errorf(" %v store and upload to the chain failed", respspacefile.FileId)
				}
			}(basedir, respspacefile.FileId)
			allsuc = true
		}
	}
}

//The task_HandlingChallenges task will automatically help you complete file challenges.
//Apart from human influence, it ensures that you submit your certificates in a timely manner.
//It keeps running as a subtask.
func task_HandlingChallenges(ch chan bool) {
	var (
		err           error
		code          int
		fileid        string
		filedir       string
		filename      string
		tagfilename   string
		blocksize     int64
		filetag       pt.TagInfo
		poDR2prove    api.PoDR2Prove
		proveResponse api.PoDR2ProveResponse
		puk           chain.Chain_SchedulerPuk
		chlng         []chain.ChallengesInfo
	)
	defer func() {
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
		ch <- true
	}()
	Out.Info(">>>Start task_HandlingChallenges<<<")

	//Get the scheduling service public key
	puk, _, err = chain.GetSchedulerPukFromChain()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	for {
		chlng, code, err = chain.GetChallengesById(configs.MinerId_I)
		if err != nil {
			if code == configs.Code_404 {
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(30, 120)))
			}
			continue
		}
		for i := 0; i < len(chlng); i++ {
			if chlng[i].File_type == 1 {
				//space file
				filedir = filepath.Join(configs.SpaceDir, string(chlng[i].File_id))
				filename = string(chlng[i].File_id) + ".space"
				fileid = string(chlng[i].File_id)
			} else {
				//user file
				fileid = strings.Split(string(chlng[i].File_id), ".")[0]
				filedir = filepath.Join(configs.FilesDir, fileid)
				filename = string(chlng[i].File_id)
			}
			tagfilename = string(chlng[i].File_id) + ".tag"
			fstat, err := os.Stat(filepath.Join(filedir, filename))
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", filedir, err)
				continue
			}
			if chlng[i].File_type == 1 {
				blocksize = configs.Space_1MB
			} else {
				blocksize, _ = calcFileBlockSizeAndScanSize(fstat.Size())
			}

			qSlice, err := api.PoDR2ChallengeGenerateFromChain(chlng[i].Block_list, chlng[i].Random)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", filedir, err)
				continue
			}
			ftag, err := ioutil.ReadFile(filepath.Join(filedir, tagfilename))
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", filename, err)
				continue
			}
			err = json.Unmarshal(ftag, &filetag)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", filename, err)
				continue
			}
			f, err := os.OpenFile(filepath.Join(filedir, filename), os.O_RDONLY, os.ModePerm)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", filename, err)
				continue
			}
			poDR2prove.QSlice = qSlice
			poDR2prove.T = filetag.T
			poDR2prove.Sigmas = filetag.Sigmas

			matrix, _, _, err := tools.Split(f, blocksize)
			if err != nil {
				f.Close()
				Err.Sugar().Errorf("[%v] %v", filename, err)
				continue
			}
			f.Close()
			poDR2prove.Matrix = matrix
			poDR2prove.S = blocksize
			proveResponseCh := poDR2prove.PoDR2ProofProve(puk.Spk, string(puk.Shared_params), puk.Shared_g, int64(chlng[i].Scan_size))
			select {
			case proveResponse = <-proveResponseCh:
			}
			if proveResponse.StatueMsg.StatusCode != api.Success {
				Err.Sugar().Errorf("[%v] %v", filename, err)
				continue
			}

			// proof up chain
			ts := time.Now().Unix()
			code = 0
			for code != int(configs.Code_200) && code != int(configs.Code_600) {
				code, err = chain.PutProofToChain(configs.C.SignaturePrk, configs.MinerId_I, []byte(chlng[i].File_id), proveResponse.Sigma, proveResponse.MU)
				if err == nil {
					Out.Sugar().Infof("[%v] Proof submitted successfully", fileid)
					break
				}
				if time.Since(time.Unix(ts, 0)).Minutes() > 10.0 {
					Err.Sugar().Errorf("[%v] %v", filename, err)
					break
				}
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 20)))
			}
		}
	}
}

//The task_RemoveInvalidFiles task automatically checks its own failed files and clears them.
//Delete from the local disk first, and then notify the chain to delete.
//It keeps running as a subtask.
func task_RemoveInvalidFiles(ch chan bool) {
	var (
		filename string
		fileid   string
	)
	defer func() {
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
		ch <- true
	}()
	Out.Info(">>>Start task_RemoveInvalidFiles task<<<")
	for {
		invalidFiles, code, err := chain.GetInvalidFileById(configs.MinerId_I)
		if err != nil {
			if code == configs.Code_404 {
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(30, 120)))
			}
			continue
		}
		for i := 0; i < len(invalidFiles); i++ {
			fileid = string(invalidFiles[i])
			filedir := filepath.Join(configs.BaseDir, configs.SpaceDir, fileid)
			_, err = os.Stat(filedir)
			if err == nil {
				filename = fileid + ".space"
				_, err = os.Stat(filepath.Join(filedir, filename))
				if err == nil {
					os.Remove(filepath.Join(filedir, filename))
					_, err = chain.ClearInvalidFileNoChain(configs.C.SignaturePrk, configs.MinerId_I, invalidFiles[i])
					if err == nil {
						Out.Sugar().Infof("%v", err)
					}
					continue
				}
			}
			strings.TrimRight(fileid, ".")
			tmp := strings.Split(fileid, ".")
			if tmp[len(tmp)-1][0] == 'd' {
				fileid = strings.TrimSuffix(fileid, tmp[len(tmp)-1])
				filedir = filepath.Join(configs.BaseDir, configs.SpaceDir, fileid)
				_, err = os.Stat(filedir)
				if err == nil {
					_, err = os.Stat(filepath.Join(filedir, string(invalidFiles[i])))
					if err == nil {
						os.Remove(filepath.Join(filedir, string(invalidFiles[i])))
						_, err = chain.ClearInvalidFileNoChain(configs.C.SignaturePrk, configs.MinerId_I, types.Bytes([]byte(fileid)))
						if err == nil {
							Out.Sugar().Infof("%v", err)
						}
					}
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
	mountP, err := pt.GetMountPathInfo(configs.C.MountedPath)
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

func calcFileBlockSizeAndScanSize(fsize int64) (int64, int64) {
	var (
		blockSize     int64
		scanBlockSize int64
	)
	if fsize < configs.ByteSize_1Kb {
		return fsize, fsize
	}
	if fsize > math.MaxUint32 {
		blockSize = math.MaxUint32
		scanBlockSize = blockSize / 8
		return blockSize, scanBlockSize
	}
	blockSize = fsize / 16
	scanBlockSize = blockSize / 8
	return blockSize, scanBlockSize
}

func connectionScheduler(schds []chain.SchedulerInfo) (*rpc.Client, error) {
	var (
		err error
		cli *rpc.Client
	)
	if len(schds) == 0 {
		return nil, errors.New("No scheduler service available")
	}
	var deduplication = make(map[int]struct{}, len(schds))
	var wsURL string
	for i := 0; i < len(schds); i++ {
		index := tools.RandomInRange(0, len(schds))
		_, ok := deduplication[index]
		if ok {
			continue
		}
		deduplication[index] = struct{}{}
		wsURL = "ws://" + string(base58.Decode(string(schds[index].Ip)))
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		cli, err = rpc.DialWebsocket(ctx, wsURL, "")
		if err != nil {
			if (i + 1) == len(schds) {
				return nil, errors.New("All schedules unavailable")
			}
			continue
		}
		break
	}
	return cli, err
}
