package task

import (
	"cess-bucket/configs"
	"cess-bucket/internal/chain"
	. "cess-bucket/internal/logger"
	api "cess-bucket/internal/proof/apiv1"
	"cess-bucket/tools"
	"encoding/json"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

//The task_HandlingChallenges task will automatically help you complete file challenges.
//Apart from human influence, it ensures that you submit your certificates in a timely manner.
//It keeps running as a subtask.
func task_HandlingChallenges(ch chan bool) {
	var (
		fileid          string
		fileFullPath    string
		fileTagFullPath string
		blocksize       int64
		filetag         api.TagInfo
		poDR2prove      api.PoDR2Prove
		proveResponse   api.PoDR2ProveResponse
	)
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
		ch <- true
	}()
	Chg.Info(">>>>> Start task_HandlingChallenges <<<<<")

	for {
		chlng, err := chain.GetChallenges()
		if err != nil {
			if err.Error() != chain.ERR_Empty {
				Chg.Sugar().Errorf("%v", err)
			}
			time.Sleep(time.Minute * time.Duration(tools.RandomInRange(1, 3)))
			continue
		}

		if len(chlng) == 0 {
			time.Sleep(time.Minute * time.Duration(tools.RandomInRange(2, 5)))
			continue
		}

		time.Sleep(time.Second * time.Duration(tools.RandomInRange(30, 60)))
		Chg.Sugar().Infof("--> Number of challenges: %v ", len(chlng))
		for x := 0; x < len(chlng); x++ {
			Chg.Sugar().Infof("  %v: %s", x, string(chlng[x].File_id))
		}
		var proveInfos = make([]chain.ProveInfo, 0)
		for i := 0; i < len(chlng); i++ {
			if len(proveInfos) >= 80 {
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
				Chg.Sugar().Errorf("[%v] %v", fileid, err)
				continue
			}
			if chlng[i].File_type == 1 {
				blocksize = configs.BlockSize
			} else {
				blocksize, _ = calcFileBlockSizeAndScanSize(fstat.Size())
			}

			qSlice, err := api.PoDR2ChallengeGenerateFromChain(chlng[i].Block_list, chlng[i].Random)
			if err != nil {
				Chg.Sugar().Errorf("[%v] %v", fileid, err)
				continue
			}

			ftag, err := ioutil.ReadFile(fileTagFullPath)
			if err != nil {
				Chg.Sugar().Errorf("[%v] %v", fileid, err)
				continue
			}
			err = json.Unmarshal(ftag, &filetag)
			if err != nil {
				Chg.Sugar().Errorf("[%v] %v", fileid, err)
				continue
			}

			poDR2prove.QSlice = qSlice
			poDR2prove.T = filetag.T
			poDR2prove.Sigmas = filetag.Sigmas

			matrix, _, err := split(fileFullPath, blocksize, fstat.Size())
			if err != nil {
				Chg.Sugar().Errorf("[%v] %v", fileid, err)
				continue
			}

			poDR2prove.Matrix = matrix
			poDR2prove.S = blocksize
			proveResponseCh := poDR2prove.PoDR2ProofProve(configs.Spk, string(configs.Shared_params), configs.Shared_g, int64(configs.ScanBlockSize))
			select {
			case proveResponse = <-proveResponseCh:
			}
			if proveResponse.StatueMsg.StatusCode != api.Success {
				Chg.Sugar().Errorf("[%v] PoDR2ProofProve failed", fileid)
				continue
			}

			var proveInfoTemp chain.ProveInfo
			proveInfoTemp.Cinfo = chlng[i]
			proveInfoTemp.FileId = chlng[i].File_id

			var mus []types.Bytes = make([]types.Bytes, len(proveResponse.MU))
			for i := 0; i < len(proveResponse.MU); i++ {
				mus[i] = make(types.Bytes, 0)
				mus[i] = append(mus[i], proveResponse.MU[i]...)
			}
			proveInfoTemp.Mu = mus
			proveInfoTemp.Sigma = types.Bytes(proveResponse.Sigma)
			proveInfoTemp.MinerAcc = types.NewAccountID(configs.PublicKey)
			proveInfos = append(proveInfos, proveInfoTemp)
		}

		if len(proveInfos) == 0 {
			continue
		}
		// proof up chain
		ts := time.Now().Unix()
		var txhash string
		for {
			txhash, err = chain.SubmitProofs(proveInfos)
			if err != nil {
				Chg.Sugar().Errorf("SubmitProofs fail: %v", err)
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 20)))
			}

			if txhash != "" {
				Chg.Sugar().Infof("SubmitProofs suc: %v", txhash)
				break
			}
			if time.Since(time.Unix(ts, 0)).Minutes() > 2.0 {
				Chg.Sugar().Errorf("SubmitProofs fail and exit")
				break
			}
		}
	}
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

func split(filefullpath string, blocksize, filesize int64) ([][]byte, uint64, error) {
	file, err := os.Open(filefullpath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	if filesize/blocksize == 0 {
		return nil, 0, errors.New("filesize invalid")
	}
	n := filesize / blocksize
	if n == 0 {
		n = 1
	}
	// matrix is indexed as m_ij, so the first dimension has n items and the second has s.
	matrix := make([][]byte, n)
	for i := int64(0); i < n; i++ {
		piece := make([]byte, blocksize)
		_, err := file.Read(piece)
		if err != nil {
			return nil, 0, err
		}
		matrix[i] = piece
	}
	return matrix, uint64(n), nil
}
