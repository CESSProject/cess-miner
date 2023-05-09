/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/proof"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/sdk-go/core/chain"
	"github.com/CESSProject/sdk-go/core/client"
	"github.com/CESSProject/sdk-go/core/rule"
)

// challengeMgr
func (n *Node) challengeMgr(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Log.Pnc(utils.RecoverError(err))
		}
	}()

	var err error
	var key *proof.RSAKeyPair
	var challenge client.ChallengeInfo

	for {
		pubkey, err := n.Cli.QueryTeePodr2Puk()
		if err != nil || len(pubkey) == 0 {
			time.Sleep(rule.BlockInterval)
			continue
		}
		n.Log.Chal("info", fmt.Sprintf("TEEKey: %v", pubkey))
		key = proof.GetKey(pubkey)
		break
	}

	for {
		challenge, err = n.Cli.QueryChallenge(n.Cfg.GetPublickey())
		if err != nil {
			if err.Error() != chain.ERR_Empty {
				n.Log.Chal("err", err.Error())
				continue
			}
		}
		if challenge.Start == 0 {
			continue
		}

		n.Log.Chal("info", fmt.Sprintf("Challenge start: %v", challenge.Start))
		n.Log.Chal("info", fmt.Sprintf("Challenge random: %v", challenge.Random))

		//Query all files before start
		utils.DirFiles(filepath.Join(n.Cli.Workspace(), configs.SpaceDir), 0)

		//Calc all files proof
		key = key

		//submit proof
	}
	// var (
	// 	err        error
	// 	tStart     time.Time
	// 	chlng      []chain.ChallengesInfo
	// 	proveInfos = make([]chain.ProveInfo, 0)
	// )

	// //Chg.Info(">>>>> Start task_HandlingChallenges <<<<<")

	// for {
	// 	// if pattern.GetMinerState() != pattern.M_Positive {
	// 	// 	if pattern.GetMinerState() == pattern.M_Pending {
	// 	// 		time.Sleep(time.Second * configs.BlockInterval)
	// 	// 		continue
	// 	// 	}
	// 	// 	time.Sleep(time.Minute * time.Duration(tools.RandomInRange(1, 5)))
	// 	// 	continue
	// 	// }

	// 	// chlng, err = chain.GetChallenges()
	// 	// if err != nil {
	// 	// 	if err.Error() != chain.ERR_Empty {
	// 	// 		//Chg.Sugar().Errorf("%v", err)
	// 	// 	}
	// 	// 	time.Sleep(time.Minute)
	// 	// 	continue
	// 	// }

	// 	// time.Sleep(time.Second * time.Duration(tools.RandomInRange(30, 60)))
	// 	// //Chg.Sugar().Infof("--> Number of challenges: %v ", len(chlng))

	// 	// for i := 0; i < len(chlng); i++ {
	// 	// 	if len(proveInfos) >= configs.MaxProofData {
	// 	// 		submitProofResult(proveInfos)
	// 	// 		proveInfos = make([]chain.ProveInfo, 0)
	// 	// 	}
	// 	// 	tStart = time.Now()
	// 	// 	prf := calcProof(chlng[i])
	// 	// 	//Chg.Sugar().Infof("calc challenge time: %v ", time.Since(tStart).Microseconds())
	// 	// 	proveInfos = append(proveInfos, prf)
	// 	// }

	// 	// // proof up chain
	// 	// submitProofResult(proveInfos)
	// 	// proveInfos = make([]chain.ProveInfo, 0)
	// 	// time.Sleep(configs.BlockInterval)
	// }
}

// func submitProofResult(proofs []chain.ProveInfo) {
// 	var (
// 		err      error
// 		tryCount uint8
// 		txhash   string
// 	)
// 	// submit proof results
// 	if len(proofs) > 0 {
// 		// fmt.Println("---------------")
// 		// fmt.Println("FileId:", string(proofs[0].FileId[:]))
// 		// fmt.Println("chal:", proofs[0].Cinfo)
// 		// fmt.Println("u:", proofs[0].U)
// 		// fmt.Println("mu:", proofs[0].Mu)
// 		// fmt.Println("sigma:", proofs[0].Sigma)
// 		// fmt.Println("Omega:", proofs[0].Omega)
// 		// fmt.Println("SigRoothash:", proofs[0].SigRootHash)
// 		// fmt.Println("HashMi:", proofs[0].HashMi)
// 		// fmt.Println("---------------")

// 		for {
// 			txhash, err = chain.SubmitProofs(proofs)
// 			if err != nil {
// 				tryCount++
// 				//Chg.Sugar().Errorf("Proof result submitted err: %v", err)
// 			}
// 			if txhash != "" {
// 				//Chg.Sugar().Infof("Proof result submitted suc: %v", txhash)
// 				return
// 			}
// 			if tryCount >= 3 {
// 				return
// 			}
// 			time.Sleep(configs.BlockInterval)
// 		}
// 	}
// 	return
// }

// func calcProof(challenge chain.ChallengesInfo) chain.ProveInfo {
// 	var (
// 		err             error
// 		fileid          string
// 		shardId         string
// 		fileFullPath    string
// 		fileTagFullPath string
// 		filetag         proof.StorageTagType
// 		proveResponse   proof.GenProofResponse
// 		proveInfoTemp   chain.ProveInfo
// 	)

// 	proveInfoTemp.Cinfo = challenge
// 	proveInfoTemp.FileId = challenge.File_id
// 	acc, _ := types.NewAccountID(pattern.GetMinerAcc())
// 	proveInfoTemp.MinerAcc = *acc

// 	fileid = string(challenge.File_id[:])
// 	if challenge.File_type == 1 {
// 		//space file
// 		fileFullPath = filepath.Join(configs.SpaceDir, fileid)
// 		fileTagFullPath = filepath.Join(configs.SpaceDir, fileid+".tag")
// 	} else {
// 		//user file
// 		shardId = string(challenge.Shard_id[:])
// 		fileid = strings.Split(shardId, ".")[0]
// 		fileFullPath = filepath.Join(configs.FileDir, shardId)
// 		fileTagFullPath = filepath.Join(configs.FileDir, shardId+".tag")
// 	}

// 	_, err = os.Stat(fileFullPath)
// 	if err != nil {
// 		//Chg.Sugar().Errorf("[%v] %v", fileid, err)
// 		return proveInfoTemp
// 	}

// 	qSlice, err := proof.PoDR2ChallengeGenerateFromChain(challenge.Block_list, challenge.Random)
// 	if err != nil {
// 		//Chg.Sugar().Errorf("[%v] %v", fileid, err)
// 		return proveInfoTemp
// 	}

// 	ftag, err := ioutil.ReadFile(fileTagFullPath)
// 	if err != nil {
// 		//Chg.Sugar().Errorf("[%v] %v", fileid, err)
// 		return proveInfoTemp
// 	}

// 	err = json.Unmarshal(ftag, &filetag)
// 	if err != nil {
// 		//Chg.Sugar().Errorf("[%v] %v", fileid, err)
// 		return proveInfoTemp
// 	}

// 	proveInfoTemp.U = filetag.T.U

// 	matrix, _, err := proof.SplitV2(fileFullPath, configs.BlockSize)
// 	if err != nil {
// 		//Chg.Sugar().Errorf("[%v] %v", fileid, err)
// 		return proveInfoTemp
// 	}

// 	E_bigint, _ := new(big.Int).SetString(filetag.E, 10)
// 	N_bigint, _ := new(big.Int).SetString(filetag.N, 10)

// 	fmt.Println("Will gen proof: ", string(challenge.File_id[:]))
// 	proveResponseCh := proof.GetKey(int(E_bigint.Int64()), N_bigint).GenProof(qSlice, filetag.T, filetag.Phi, matrix, filetag.SigRootHash)
// 	select {
// 	case proveResponse = <-proveResponseCh:
// 		if proveResponse.StatueMsg.StatusCode != proof.Success {
// 			return proveInfoTemp
// 		}
// 	}
// 	fmt.Println("Gen proof suc: ", string(challenge.File_id[:]))
// 	fmt.Println()

// 	// Chg.Sugar().Infof("fileid: %v", fileid)
// 	// Chg.Sugar().Infof("len(MU)", len(proveResponse.MU))
// 	// Chg.Sugar().Infof("len(Sigma)", len(proveResponse.Sigma))
// 	// Chg.Sugar().Infof("len(Omega)", len(proveResponse.Omega))
// 	// Chg.Sugar().Infof("len(SigRootHash)", len(proveResponse.SigRootHash))
// 	// Chg.Sugar().Infof("len(HashMi)", len(proveResponse.HashMi)*32)

// 	proveInfoTemp.Mu = proveResponse.MU
// 	proveInfoTemp.Sigma = proveResponse.Sigma
// 	proveInfoTemp.Omega = proveResponse.Omega
// 	proveInfoTemp.SigRootHash = proveResponse.SigRootHash
// 	proveInfoTemp.HashMi = make([]types.Bytes, len(proveResponse.HashMi))
// 	for i := 0; i < len(proveResponse.HashMi); i++ {
// 		proveInfoTemp.HashMi[i] = make(types.Bytes, 0)
// 		proveInfoTemp.HashMi[i] = append(proveInfoTemp.HashMi[i], proveResponse.HashMi[i]...)
// 	}
// 	return proveInfoTemp
// }

// func calcFileBlockSizeAndScanSize(fsize int64) (int64, int64) {
// 	var (
// 		blockSize     int64
// 		scanBlockSize int64
// 	)
// 	if fsize < configs.SIZE_1KiB {
// 		return fsize, fsize
// 	}
// 	if fsize > math.MaxUint32 {
// 		blockSize = math.MaxUint32
// 		scanBlockSize = blockSize / 8
// 		return blockSize, scanBlockSize
// 	}
// 	blockSize = fsize / 16
// 	scanBlockSize = blockSize / 8
// 	return blockSize, scanBlockSize
// }

// func split(filefullpath string, blocksize, filesize int64) ([][]byte, uint64, error) {
// 	file, err := os.Open(filefullpath)
// 	if err != nil {
// 		return nil, 0, err
// 	}
// 	defer file.Close()

// 	if filesize/blocksize == 0 {
// 		return nil, 0, errors.New("filesize invalid")
// 	}
// 	n := filesize / blocksize
// 	if n == 0 {
// 		n = 1
// 	}
// 	// matrix is indexed as m_ij, so the first dimension has n items and the second has s.
// 	matrix := make([][]byte, n)
// 	for i := int64(0); i < n; i++ {
// 		piece := make([]byte, blocksize)
// 		_, err := file.Read(piece)
// 		if err != nil {
// 			return nil, 0, err
// 		}
// 		matrix[i] = piece
// 	}
// 	return matrix, uint64(n), nil
// }
