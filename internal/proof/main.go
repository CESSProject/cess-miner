package proof

import (
	"cess-bucket/configs"
	"cess-bucket/internal/chain"
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

	keyring "github.com/CESSProject/go-keyring"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"storj.io/common/base58"
)

type RespSpaceInfo struct {
	FileId string `json:"fileId"`
	Token  string `json:"token"`
	T      api.FileTagT
	Sigmas [][]byte `json:"sigmas"`
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
		availableSpace uint64
		reconn         bool
		tSpace         time.Time
		reqspace       p.SpaceReq
		reqspacefile   p.SpaceFileReq
		tagInfo        pt.TagInfo
		respspace      RespSpaceInfo
		client         *rpc.Client
	)
	defer func() {
		if err := recover(); err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
		ch <- true
	}()
	Out.Info(">>>Start task_SpaceManagement <<<")

	pubkey, err := chain.GetPublickeyFromPrk(configs.C.SignaturePrk)
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

	reqspace.Publickey = pubkey

	kr, _ := keyring.FromURI(configs.C.SignaturePrk, keyring.NetSubstrate{})

	for {
		if client == nil || reconn {
			_, schds, _ := chain.GetAllSchedulerInfo()
			client, err = connectionScheduler(schds)
			if err != nil {
				Err.Sugar().Errorf("-->Err: All schedules unavailable")
				for i := 0; i < len(schds); i++ {
					Err.Sugar().Errorf("   %v", string(schds[i].Ip))
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

		if availableSpace < uint64(8*configs.Space_1MB) {
			time.Sleep(time.Minute * time.Duration(tools.RandomInRange(1, 10)))
			continue
		}

		// sign message
		msg := []byte(fmt.Sprintf("%v", tools.RandomInRange(100000, 999999)))
		sig, _ := kr.Sign(kr.SigningContext(msg))
		reqspace.Msg = msg
		reqspace.Sign = sig[:]

		req_b, err := proto.Marshal(&reqspace)
		if err != nil {
			Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
			continue
		}

		respCode, respBody, clo, err := rpc.WriteData(client, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_Space, req_b)
		reconn = clo
		if err != nil || respCode != configs.Code_200 {
			Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
			continue
		}

		err = json.Unmarshal(respBody, &respspace)
		if err != nil {
			Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
			continue
		}

		//save space file tag
		tagfilename := respspace.FileId + ".tag"
		tagfilefullpath := filepath.Join(configs.SpaceDir, tagfilename)
		tagInfo.T = respspace.T
		tagInfo.Sigmas = respspace.Sigmas
		tag, err := json.Marshal(tagInfo)
		if err != nil {
			Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
			continue
		}
		err = genFileTag(tagfilefullpath, tag)
		if err != nil {
			os.Remove(tagfilefullpath)
			Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
			continue
		}

		spacefilefullpath := filepath.Join(configs.SpaceDir, respspace.FileId)
		f, err := os.OpenFile(spacefilefullpath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
		if err != nil {
			os.Remove(tagfilefullpath)
			Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
			continue
		}
		reqspacefile.Token = respspace.Token

		for i := 0; i < 17; i++ {
			reqspacefile.BlockIndex = uint32(i)
			req_b, err = proto.Marshal(&reqspacefile)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				f.Close()
				os.Remove(tagfilefullpath)
				os.Remove(spacefilefullpath)
				break
			}
			respCode, respBody, clo, err = rpc.WriteData(client, configs.RpcService_Scheduler, configs.RpcMethod_Scheduler_Spacefile, req_b)
			reconn = clo
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", configs.MinerId_S, err)
				f.Close()
				os.Remove(tagfilefullpath)
				os.Remove(spacefilefullpath)
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

//The task_HandlingChallenges task will automatically help you complete file challenges.
//Apart from human influence, it ensures that you submit your certificates in a timely manner.
//It keeps running as a subtask.
func task_HandlingChallenges(ch chan bool) {
	var (
		err             error
		code            int
		fileid          string
		fileFullPath    string
		fileTagFullPath string
		blocksize       int64
		filetag         pt.TagInfo
		poDR2prove      api.PoDR2Prove
		proveResponse   api.PoDR2ProveResponse
		puk             chain.Chain_SchedulerPuk
		chlng           []chain.ChallengesInfo
	)
	defer func() {
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
		ch <- true
	}()
	Out.Info(">>>>> Start task_HandlingChallenges <<<<<")

	//Get the scheduling service public key
	for {
		puk, _, err = chain.GetSchedulerPukFromChain()
		if err != nil {
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 30)))
			continue
		}
		Out.Info(">>>>> Get puk ok <<<<<")
		break
	}

	miner_pubkey, err := chain.GetPublickeyFromPrk(configs.C.SignaturePrk)
	if err != nil {
		Err.Sugar().Errorf("GetPublickeyFromPrk err: %v", err)
		os.Exit(1)
	}

	for {
		chlng, code, err = chain.GetChallengesById(miner_pubkey)
		if err != nil {
			if code != configs.Code_404 {
				Out.Sugar().Infof("[ERR] %v", err)
			}
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 30)))
			continue
		}

		if len(chlng) == 0 {
			continue
		}

		Out.Sugar().Infof("---> Prepare to generate challenges [%v]\n", len(chlng))
		for x := 0; x < len(chlng); x++ {
			Out.Sugar().Infof("     %v: %s\n", x, string(chlng[x].File_id))
		}
		var proveInfos = make([]chain.ProveInfo, 0)
		for i := 0; i < len(chlng); i++ {
			if len(proveInfos) > 80 {
				break
			}
			fileid = string(chlng[i].File_id)
			if chlng[i].File_type == 1 {
				//space file
				fileFullPath = filepath.Join(configs.SpaceDir, fileid)
				fileTagFullPath = filepath.Join(configs.SpaceDir, fileid+".tag")
			} else {
				//user file
				fileFullPath = filepath.Join(configs.FilesDir, fileid)
				fileTagFullPath = filepath.Join(configs.FilesDir, fileid+".tag")
			}

			fstat, err := os.Stat(fileFullPath)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", fileid, err)
				continue
			}
			if chlng[i].File_type == 1 {
				blocksize = configs.Space_1MB
			} else {
				blocksize, _ = calcFileBlockSizeAndScanSize(fstat.Size())
			}

			qSlice, err := api.PoDR2ChallengeGenerateFromChain(chlng[i].Block_list, chlng[i].Random)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", fileid, err)
				continue
			}
			ftag, err := ioutil.ReadFile(fileTagFullPath)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", fileid, err)
				continue
			}
			err = json.Unmarshal(ftag, &filetag)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", fileid, err)
				continue
			}

			poDR2prove.QSlice = qSlice
			poDR2prove.T = filetag.T
			poDR2prove.Sigmas = filetag.Sigmas

			matrix, _, err := split(fileFullPath, blocksize, fstat.Size())
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", fileid, err)
				continue
			}

			poDR2prove.Matrix = matrix
			poDR2prove.S = blocksize
			proveResponseCh := poDR2prove.PoDR2ProofProve(puk.Spk, string(puk.Shared_params), puk.Shared_g, int64(chlng[i].Scan_size))
			select {
			case proveResponse = <-proveResponseCh:
			}
			if proveResponse.StatueMsg.StatusCode != api.Success {
				Err.Sugar().Errorf("[%v] %v", fileid, err)
				continue
			}

			proveInfoTemp := chain.ProveInfo{}
			proveInfoTemp.Cinfo = chlng[i]
			proveInfoTemp.FileId = chlng[i].File_id

			var mus []types.Bytes = make([]types.Bytes, len(proveResponse.MU))
			for i := 0; i < len(proveResponse.MU); i++ {
				mus[i] = make(types.Bytes, 0)
				mus[i] = append(mus[i], proveResponse.MU[i]...)
			}
			proveInfoTemp.Mu = mus
			proveInfoTemp.Sigma = types.Bytes(proveResponse.Sigma)
			proveInfoTemp.MinerId = types.U64(configs.MinerId_I)
			proveInfos = append(proveInfos, proveInfoTemp)
		}
		// proof up chain
		ts := time.Now().Unix()
		code = 0
		for code != int(configs.Code_200) && code != int(configs.Code_600) {
			code, err = chain.PutProofToChain(configs.C.SignaturePrk, configs.MinerId_I, proveInfos)
			if err == nil {
				Out.Sugar().Infof("Proofs submitted successfully")
				break
			}
			if time.Since(time.Unix(ts, 0)).Minutes() > 2.0 {
				Err.Sugar().Errorf("[%v] %v", fileid, err)
				break
			}
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 20)))
		}
		proveInfos = proveInfos[:0]
		time.Sleep(time.Second * time.Duration(tools.RandomInRange(30, 60)))
	}
}

//The task_RemoveInvalidFiles task automatically checks its own failed files and clears them.
//Delete from the local disk first, and then notify the chain to delete.
//It keeps running as a subtask.
func task_RemoveInvalidFiles(ch chan bool) {
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

		if len(invalidFiles) == 0 {
			continue
		}

		fmt.Printf("---> Prepare to remove invalid files [%v]\n", len(invalidFiles))
		for x := 0; x < len(invalidFiles); x++ {
			fmt.Printf("     %v: %s\n", x, string(invalidFiles[x]))
		}

		for i := 0; i < len(invalidFiles); i++ {
			ext := filepath.Ext(string(invalidFiles[i]))
			if ext == "" {
				filedir := filepath.Join(configs.SpaceDir, string(invalidFiles[i]))
				os.RemoveAll(filedir)
				_, err = chain.ClearInvalidFileNoChain(configs.C.SignaturePrk, configs.MinerId_I, invalidFiles[i])
				if err == nil {
					fmt.Printf("     removed successfully [%v]\n", string(invalidFiles[i]))
					Out.Sugar().Infof("%v", err)
				}
				continue
			}
			fileid := strings.TrimSuffix(string(invalidFiles[i]), ext)
			filefullpath := filepath.Join(configs.FilesDir, fileid, string(invalidFiles[i]))
			filetagfullpath := filepath.Join(configs.FilesDir, fileid, string(invalidFiles[i])+".tag")
			os.Remove(filefullpath)
			os.Remove(filetagfullpath)
			_, err = chain.ClearInvalidFileNoChain(configs.C.SignaturePrk, configs.MinerId_I, types.Bytes([]byte(fileid)))
			if err == nil {
				Out.Sugar().Infof("%v", err)
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

func split(filefullpath string, blocksize, filesize int64) ([][]byte, uint64, error) {
	file, err := os.Open(filefullpath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	if filesize/blocksize == 0 {
		return nil, 0, errors.New("filesize invalid")
	}
	n := uint64(math.Ceil(float64(filesize / blocksize)))
	if n == 0 {
		n = 1
	}
	// matrix is indexed as m_ij, so the first dimension has n items and the second has s.
	matrix := make([][]byte, n)
	for i := uint64(0); i < n; i++ {
		piece := make([]byte, blocksize)
		_, err := file.Read(piece)
		if err != nil {
			return nil, 0, err
		}
		matrix[i] = piece
	}
	return matrix, n, nil
}

func genSpaceFile(fpath, content string, rule []byte) error {
	if len(content) != 4096 || len(rule) != 1023 {
		return errors.New("content err")
	}

	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	for i := 0; i < len(rule); i++ {
		f.WriteString(content[rule[i]:])
		f.WriteString(content[:rule[i]])
		f.WriteString("\n")
		f.WriteString(content[rule[i]*rule[i]:])
		f.WriteString(content[:rule[i]*rule[i]])
		f.WriteString("\n")
		if i+1 == len(rule) {
			f.WriteString(content)
			f.WriteString("\n")
			f.WriteString(content[3884:])
		}
	}
	return f.Sync()
}

func genFileTag(fpath string, data []byte) error {
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
