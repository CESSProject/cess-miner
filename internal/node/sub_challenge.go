package node

import (
	"encoding/json"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/internal/chain"
	. "github.com/CESSProject/cess-bucket/internal/logger"
	"github.com/CESSProject/cess-bucket/internal/pattern"
	"github.com/CESSProject/cess-bucket/internal/proof"
	"github.com/CESSProject/cess-bucket/tools"

	"github.com/pkg/errors"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

// The task_HandlingChallenges task will automatically help you complete file challenges.
// Apart from human influence, it ensures that you submit your certificates in a timely manner.
// It keeps running as a subtask.
func (node *Node) task_HandlingChallenges(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	var (
		err        error
		chlng      []chain.ChallengesInfo
		proveInfos = make([]chain.ProveInfo, 0)
	)

	Chg.Info(">>>>> Start task_HandlingChallenges <<<<<")

	for {
		// if pattern.GetMinerState() != pattern.M_Positive {
		// 	if pattern.GetMinerState() == pattern.M_Pending {
		// 		time.Sleep(time.Second * configs.BlockInterval)
		// 		continue
		// 	}
		// 	time.Sleep(time.Minute * time.Duration(tools.RandomInRange(1, 5)))
		// 	continue
		// }

		chlng, err = chain.GetChallenges()
		if err != nil {
			if err.Error() != chain.ERR_Empty {
				Chg.Sugar().Errorf("%v", err)
			}
			time.Sleep(time.Minute)
			continue
		}

		time.Sleep(time.Second * time.Duration(tools.RandomInRange(30, 60)))
		Chg.Sugar().Infof("--> Number of challenges: %v ", len(chlng))

		for i := 0; i < len(chlng); i++ {
			if len(proveInfos) >= configs.MaxProofData {
				submitProofResult(proveInfos)
				proveInfos = make([]chain.ProveInfo, 0)
			}
			proveInfos = append(proveInfos, calcProof(chlng[i]))
		}

		// proof up chain
		submitProofResult(proveInfos)
		proveInfos = make([]chain.ProveInfo, 0)
	}
}

func submitProofResult(proofs []chain.ProveInfo) {
	var (
		err      error
		tryCount uint8
		txhash   string
	)
	// submit proof results
	if len(proofs) > 0 {
		for {
			txhash, err = chain.SubmitProofs(proofs)
			if err != nil {
				tryCount++
				Chg.Sugar().Errorf("Proof result submitted err: %v", err)
			}
			if txhash != "" {
				Chg.Sugar().Infof("Proof result submitted suc: %v", txhash)
				return
			}
			if tryCount >= 3 {
				return
			}
			time.Sleep(configs.BlockInterval)
		}
	}
	return
}

func calcProof(challenge chain.ChallengesInfo) chain.ProveInfo {
	var (
		err             error
		fileid          string
		fileFullPath    string
		fileTagFullPath string
		filetag         proof.StorageTagType
		proveResponse   proof.GenProofResponse
		proveInfoTemp   chain.ProveInfo
	)

	proveInfoTemp.Cinfo = challenge
	proveInfoTemp.FileId = challenge.File_id
	proveInfoTemp.MinerAcc = types.NewAccountID(pattern.GetMinerAcc())

	fileid = string(challenge.File_id[:])
	if challenge.File_type == 1 {
		//space file
		fileFullPath = filepath.Join(configs.SpaceDir, fileid)
		fileTagFullPath = filepath.Join(configs.SpaceDir, fileid+".tag")
	} else {
		//user file
		fileFullPath = filepath.Join(configs.FilesDir, fileid)
		fileTagFullPath = filepath.Join(configs.FilesDir, fileid+".tag")
	}

	_, err = os.Stat(fileFullPath)
	if err != nil {
		Chg.Sugar().Errorf("[%v] %v", fileid, err)
		return proveInfoTemp
	}

	qSlice, err := proof.PoDR2ChallengeGenerateFromChain(challenge.Block_list, challenge.Random)
	if err != nil {
		Chg.Sugar().Errorf("[%v] %v", fileid, err)
		return proveInfoTemp
	}

	ftag, err := ioutil.ReadFile(fileTagFullPath)
	if err != nil {
		Chg.Sugar().Errorf("[%v] %v", fileid, err)
		return proveInfoTemp
	}

	err = json.Unmarshal(ftag, &filetag)
	if err != nil {
		Chg.Sugar().Errorf("[%v] %v", fileid, err)
		return proveInfoTemp
	}

	proveInfoTemp.U = filetag.T.U

	matrix, _, err := proof.SplitV2(fileFullPath, configs.BlockSize)
	if err != nil {
		Chg.Sugar().Errorf("[%v] %v", fileid, err)
		return proveInfoTemp
	}

	E_bigint, _ := new(big.Int).SetString(filetag.E, 10)
	N_bigint, _ := new(big.Int).SetString(filetag.N, 10)
	proveResponseCh := proof.GetKey(int(E_bigint.Int64()), N_bigint).GenProof(qSlice, filetag.T, filetag.Phi, matrix, filetag.SigRootHash)

	select {
	case proveResponse = <-proveResponseCh:
		if proveResponse.StatueMsg.StatusCode != proof.Success {
			return proveInfoTemp
		}
	}

	proveInfoTemp.Mu = proveResponse.MU
	proveInfoTemp.Sigma = proveResponse.Sigma
	proveInfoTemp.Omega = proveResponse.Omega
	proveInfoTemp.SigRootHash = proveResponse.SigRootHash
	proveInfoTemp.HashMi = make([]types.Bytes, len(proveResponse.HashMi))
	for i := 0; i < len(proveResponse.HashMi); i++ {
		proveInfoTemp.HashMi[i] = make(types.Bytes, 0)
		proveInfoTemp.HashMi[i] = append(proveInfoTemp.HashMi[i], proveResponse.HashMi[i]...)
	}
	return proveInfoTemp
}

func calcFileBlockSizeAndScanSize(fsize int64) (int64, int64) {
	var (
		blockSize     int64
		scanBlockSize int64
	)
	if fsize < configs.SIZE_1KiB {
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
