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
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/disk"
	"storj.io/common/base58"
)

type mountpathInfo struct {
	Path  string
	Total uint64
	Free  uint64
}

type RespSpacetagInfo struct {
	FileId string       `json:"fileId"`
	T      api.FileTagT `json:"file_tag_t"`
	Sigmas [][]byte     `json:"sigmas"`
}

type RespSpacefileInfo struct {
	FileId     string `json:"fileId"`
	BlockTotal uint32 `json:"blockTotal"`
	BlockIndex uint32 `json:"blockIndex"`
	BlockData  []byte `json:"blockData"`
}

// Start the proof module
func Proof_Main() {
	go processingSpace()
	go processingChallenges()
	go processingInvalidFiles()
}

func getMountPathInfo(mountpath string) (mountpathInfo, error) {
	var mp mountpathInfo
	pss, err := disk.Partitions(false)
	if err != nil {
		return mp, errors.Wrap(err, "disk.Partitions err")
	}

	for _, ps := range pss {
		us, err := disk.Usage(ps.Mountpoint)
		if err != nil {
			continue
		}
		if us.Total < configs.Space_1GB {
			continue
		} else {
			if us.Path == mountpath {
				mp.Path = us.Path
				mp.Free = us.Free
				mp.Total = us.Total
				return mp, nil
			}
		}
	}
	return mp, errors.New("Mount path not found or total space less than 1TB")
}

func spaceReasonable() {
	var err error
	configs.MinerUseSpace, err = tools.DirSize(configs.BaseDir)
	if err != nil {
		Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
		os.Exit(1)
	}

	sspace := configs.C.StorageSpace * configs.Space_1GB
	mountP, err := getMountPathInfo(configs.C.MountedPath)
	if err != nil {
		Err.Sugar().Errorf("%v", err)
		os.Exit(1)
	}
	if mountP.Total < sspace {
		Err.Sugar().Errorf("[%v] The storage space cannot be greater than the total hard disk space", configs.MinerId_S)
		os.Exit(1)
	}
	if (sspace + configs.Space_1GB) < configs.MinerUseSpace {
		Err.Sugar().Errorf("[%v] You cannot reduce your storage space", configs.MinerId_S)
		os.Exit(1)
	}
	if sspace > configs.MinerUseSpace {
		enableSpace := sspace - configs.MinerUseSpace
		if (enableSpace > mountP.Free) || ((mountP.Free - enableSpace) < configs.Space_1GB*20) {
			Err.Sugar().Errorf("[%v] Please reserve at least 20GB of space for your disk", configs.MinerId_S)
			os.Exit(1)
		}
	}
}

// Calculate available space
func calcAvailableSpace() (uint64, error) {
	var err error
	configs.MinerUseSpace, err = tools.DirSize(configs.BaseDir)
	if err != nil {
		return 0, err
	}
	sspace := configs.C.StorageSpace * configs.Space_1GB
	mountP, err := getMountPathInfo(configs.C.MountedPath)
	if err != nil {
		return 0, err
	}
	if sspace <= configs.MinerUseSpace {
		return 0, nil
	}
	enableSpace := sspace - configs.MinerUseSpace
	if (enableSpace < mountP.Free) && ((mountP.Free - enableSpace) >= configs.Space_1GB*20) {
		return enableSpace, nil
	}
	return 0, nil
}

func getSegmentNumForTypeOne(segmentpath, segtype string) (uint32, error) {
	var (
		err   error
		count uint32
	)
	_, err = os.Stat(segmentpath)
	if err != nil {
		return 0, nil
	}
	fileInfoList, err := ioutil.ReadDir(segmentpath)
	if err != nil {
		return 0, err
	}
	for i := range fileInfoList {
		if fileInfoList[i].IsDir() {
			if fileInfoList[i].Name()[:1] == segtype {
				count++
			}
		}
	}
	return count, nil
}

// Delete failed data segment
func deleteFailedSegment(path string) {
	var (
		err error
	)
	dirs, _ := getChildDirs(path)
	for i := 0; i < len(dirs); i++ {
		_, err = os.Stat(dirs[i] + "/tmp")
		if err == nil {
			err = os.RemoveAll(dirs[i])
			if err == nil {
				Err.Sugar().Infof("Remove [%v] suc", dirs[i])
			}
		}
	}
}

func getChildDirs(filePath string) ([]string, error) {
	dirs := make([]string, 0)
	f, err := os.Stat(filePath)
	if err != nil {
		return dirs, err
	}
	if !f.IsDir() {
		return dirs, errors.New("Not a dir")
	}
	files, err := ioutil.ReadDir(filePath)
	if err != nil {
		return dirs, err
	} else {
		for _, v := range files {
			if v.IsDir() {
				path := filepath.Join(filePath, v.Name())
				dirs = append(dirs, path)
			}
		}
	}
	return dirs, nil
}

//
func processingSpace() {
	var (
		err     error
		count   uint8
		enableS uint64
		req     p.SpaceTagReq
		addr    string
	)
	defer func() {
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	addr, err = chain.GetAddressFromPrk(configs.C.SignaturePrk, tools.ChainCessTestPrefix)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
	prk, err := encryption.GetRSAPrivateKey(filepath.Join(configs.BaseDir, configs.PrivateKeyfile))
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
	sign, err := encryption.CalcSign([]byte(addr), prk)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	schds, _ := chain.GetSchedulerInfo()

	for {
		time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
		//deleteFailedSegment(configs.SpaceDir)
		count++
		if count%100 == 0 {
			count = 0
			schds, _ = chain.GetSchedulerInfo()
		}
		if len(schds) == 0 {
			continue
		}
		enableS, err = calcAvailableSpace()
		if err != nil {
			Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
			continue
		}
		if enableS > uint64(8*configs.Space_1MB) {
			req.SizeMb = 8
			req.WalletAddress = addr
			req.Fileid = ""
			req.BlockIndex = 0
			req.Sign = sign

			req_b, err := proto.Marshal(&req)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}

			var client *rpc.Client
			for i, schd := range schds {
				wsURL := "ws://" + string(base58.Decode(string(schd.Ip)))
				ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
				client, err = rpc.DialWebsocket(ctx, wsURL, "")
				if err != nil {
					Err.Sugar().Errorf("[%v] %v", wsURL, err)
					if (i + 1) == len(schds) {
						Err.Sugar().Errorf("All scheduler not working")
					}
				} else {
					break
				}
			}
			if client == nil {
				continue
			}
			resp, err := rpc.WriteData(client, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_Space, req_b)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}
			var respInfo RespSpacetagInfo
			err = json.Unmarshal(resp, &respInfo)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}
			var tagInfo pt.TagInfo
			tagfilepath := filepath.Join(configs.SpaceDir, respInfo.FileId)
			_, err = os.Stat(tagfilepath)
			if err != nil {
				err = os.MkdirAll(tagfilepath, os.ModeDir)
				if err != nil {
					Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
					continue
				}
			}
			tagfilename := respInfo.FileId + ".tag"
			tagfilefullpath := filepath.Join(tagfilepath, tagfilename)
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

			// req file
			req.SizeMb = 0
			req.WalletAddress = addr
			req.Fileid = respInfo.FileId
			req.BlockIndex = 0
			req.Sign = sign

			req_b, err = proto.Marshal(&req)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}
			resp, err = rpc.WriteData(client, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_Space, req_b)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}
			var respspacefile RespSpacefileInfo
			err = json.Unmarshal(resp, &respspacefile)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}
			spacefilename := respInfo.FileId + ".space"
			spacefilefullpath := filepath.Join(tagfilepath, spacefilename)
			spacefile, err := os.OpenFile(spacefilefullpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_APPEND, os.ModePerm)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				continue
			}
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
				respi, err := rpc.WriteData(client, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_Space, req_b)
				if err != nil {
					Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
					spacefile.Close()
					os.Remove(spacefilefullpath)
					break
				}
				var respspacefilei RespSpacefileInfo
				err = json.Unmarshal(respi, &respspacefilei)
				if err != nil {
					Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
					spacefile.Close()
					os.Remove(spacefilefullpath)
					break
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
			client.Close()
		}
	}
}

//
func processingChallenges() {
	var (
		err           error
		code          int
		fileid        string
		filedir       string
		filename      string
		tagfilename   string
		filetag       pt.TagInfo
		poDR2prove    api.PoDR2Prove
		proveResponse api.PoDR2ProveResponse
		puk           chain.Chain_SchedulerPuk
		chlng         []chain.ChallengesInfo
	)
	puk, _, err = chain.GetSchedulerPukFromChain()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
	for {
		time.Sleep(time.Minute * time.Duration(tools.RandomInRange(1, 5)))
		chlng, err = chain.GetChallengesById(configs.MinerId_I)
		if err != nil {
			continue
		}
		for i := 0; i < len(chlng); i++ {
			if chlng[i].File_type == 1 {
				filedir = filepath.Join(configs.SpaceDir, string(chlng[i].File_id))
				filename = string(chlng[i].File_id) + ".space"
				fileid = string(chlng[i].File_id)
			} else {
				fileid = strings.Split(string(chlng[i].File_id), ".")[0]
				filedir = filepath.Join(configs.FilesDir, fileid)
				filename = string(chlng[i].File_id)
			}
			tagfilename = string(chlng[i].File_id) + ".tag"
			_, err = os.Stat(filepath.Join(filedir, filename))
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", filedir, err)
				continue
			}
			tmp := make(map[int]*big.Int, len(chlng[i].Block_list))
			for j := 0; j < len(chlng[i].Block_list); j++ {
				tmp[int(chlng[i].Block_list[j])] = new(big.Int).SetBytes(chlng[i].Random[j])
			}

			qSlice, err := api.PoDR2ChallengeGenerateFromChain(tmp, string(puk.Shared_params))
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

			matrix, _, _, err := tools.Split(f, int64(chlng[i].Scan_size))
			if err != nil {
				f.Close()
				Err.Sugar().Errorf("[%v] %v", filename, err)
				continue
			}
			f.Close()
			poDR2prove.Matrix = matrix
			poDR2prove.S = int64(chlng[i].Scan_size)
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
				code, err = chain.PutProofToChain(configs.C.SignaturePrk, configs.MinerId_I, []byte(fileid), proveResponse.Sigma, proveResponse.MU)
				if err == nil {
					Out.Sugar().Infof("[%v] Proof submitted successfully", fileid)
					break
				}
				if time.Since(time.Unix(ts, 0)).Minutes() > 10.0 {
					Err.Sugar().Errorf("[%v] %v", filename, err)
					continue
				}
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 20)))
			}
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", filename, err)
			}
		}
	}
}

//
func processingInvalidFiles() {
	var (
		filename string
		fileid   string
	)
	for {
		time.Sleep(time.Minute * time.Duration(tools.RandomInRange(1, 5)))
		invalidFiles, _, err := chain.GetInvalidFileById(configs.MinerId_I)
		if err != nil {
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
