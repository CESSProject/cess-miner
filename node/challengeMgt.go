/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/proof"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/CESSProject/sdk-go/core/chain"
	"github.com/CESSProject/sdk-go/core/rule"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mr-tron/base58"
)

// challengeMgr
func (n *Node) challengeMgt(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Log.Pnc(utils.RecoverError(err))
		}
	}()

	//var err error
	var txhash string
	var key *proof.RSAKeyPair
	//var challenge client.ChallengeInfo
	var idleProofFileHash []byte
	var serviceProofFileHash []byte
	var idleSiama string
	var serviceSigma string
	var qslice []proof.QElement

	n.Log.Chal("info", "Start challengeMgt task")

	//der := []byte{48, 130, 1, 10, 2, 130, 1, 1, 0, 207, 87, 46, 49, 174, 55, 159, 169, 199, 121, 54, 173, 122, 150, 249, 92, 5, 219, 28, 94, 166, 194, 249, 178, 50, 31, 97, 187, 111, 188, 0, 25, 60, 165, 243, 215, 226, 37, 92, 124, 20, 114, 205, 98, 18, 193, 86, 43, 165, 251, 248, 154, 149, 89, 46, 199, 84, 94, 25, 58, 103, 8, 117, 173, 104, 60, 205, 172, 196, 166, 44, 56, 99, 181, 218, 191, 223, 208, 190, 111, 172, 57, 64, 18, 32, 183, 192, 54, 158, 26, 125, 182, 180, 198, 86, 14, 207, 102, 39, 38, 120, 163, 140, 117, 49, 98, 80, 129, 225, 3, 178, 35, 94, 42, 9, 86, 214, 253, 67, 228, 167, 86, 10, 2, 236, 74, 74, 10, 119, 207, 27, 217, 162, 185, 246, 158, 53, 152, 135, 252, 179, 112, 46, 142, 219, 28, 216, 136, 46, 157, 225, 148, 92, 28, 203, 254, 38, 81, 173, 182, 208, 197, 183, 62, 176, 40, 94, 207, 121, 134, 205, 171, 81, 163, 31, 77, 170, 238, 216, 225, 125, 164, 210, 147, 143, 199, 136, 6, 101, 158, 186, 210, 109, 73, 82, 105, 129, 184, 158, 235, 87, 188, 169, 241, 228, 69, 209, 17, 45, 10, 81, 96, 168, 8, 4, 82, 183, 8, 197, 70, 177, 214, 75, 8, 118, 120, 131, 60, 119, 198, 18, 230, 238, 158, 7, 101, 87, 2, 215, 79, 62, 113, 248, 129, 23, 68, 108, 52, 165, 158, 251, 244, 76, 91, 32, 25, 2, 3, 1, 0, 1}
	pub_e := []byte{1, 0, 1}
	pub_n := []byte{207, 87, 46, 49, 174, 55, 159, 169, 199, 121, 54, 173, 122, 150, 249, 92, 5, 219, 28, 94, 166, 194, 249, 178, 50, 31, 97, 187, 111, 188, 0, 25, 60, 165, 243, 215, 226, 37, 92, 124, 20, 114, 205, 98, 18, 193, 86, 43, 165, 251, 248, 154, 149, 89, 46, 199, 84, 94, 25, 58, 103, 8, 117, 173, 104, 60, 205, 172, 196, 166, 44, 56, 99, 181, 218, 191, 223, 208, 190, 111, 172, 57, 64, 18, 32, 183, 192, 54, 158, 26, 125, 182, 180, 198, 86, 14, 207, 102, 39, 38, 120, 163, 140, 117, 49, 98, 80, 129, 225, 3, 178, 35, 94, 42, 9, 86, 214, 253, 67, 228, 167, 86, 10, 2, 236, 74, 74, 10, 119, 207, 27, 217, 162, 185, 246, 158, 53, 152, 135, 252, 179, 112, 46, 142, 219, 28, 216, 136, 46, 157, 225, 148, 92, 28, 203, 254, 38, 81, 173, 182, 208, 197, 183, 62, 176, 40, 94, 207, 121, 134, 205, 171, 81, 163, 31, 77, 170, 238, 216, 225, 125, 164, 210, 147, 143, 199, 136, 6, 101, 158, 186, 210, 109, 73, 82, 105, 129, 184, 158, 235, 87, 188, 169, 241, 228, 69, 209, 17, 45, 10, 81, 96, 168, 8, 4, 82, 183, 8, 197, 70, 177, 214, 75, 8, 118, 120, 131, 60, 119, 198, 18, 230, 238, 158, 7, 101, 87, 2, 215, 79, 62, 113, 248, 129, 23, 68, 108, 52, 165, 158, 251, 244, 76, 91, 32, 25}

	for {
		pubkey, err := n.Cli.QueryTeePodr2Puk()
		if err != nil || len(pubkey) == 0 {
			time.Sleep(rule.BlockInterval)
			continue
		}
		key = proof.GetKey(nil)
		key.Spk.E = int(new(big.Int).SetBytes(pub_e).Int64())
		key.Spk.N = new(big.Int).SetBytes(pub_n)
		break
	}

	var rd RandomList
	for {
		chal, err := n.Cli.QueryChallengeSt()
		if err != nil {
			fmt.Println("err1:", err)
			continue
		}

		rd.Index = chal.NetSnapshot.Random_index_list
		rd.Random = chal.NetSnapshot.Random
		buff, err := json.Marshal(&rd)
		if err != nil {
			panic(err)
		}

		ff, err := os.OpenFile(filepath.Join(n.Cli.ProofDir, "random"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
		if err != nil {
			panic(err)
		}

		defer ff.Close()
		ff.Write(buff)
		ff.Sync()

		// os.Exit(0)
		n.Log.Chal("info", fmt.Sprintf("Challenge start: %v", chal.NetSnapshot.Start))
		n.Log.Chal("info", fmt.Sprintf("Challenge randomindex: %v random length: %v", len(chal.NetSnapshot.Random_index_list), len(chal.NetSnapshot.Random)))

		// buf, err := n.Cach.Get([]byte(Cach_AggrProof_Report))
		// if err == nil {
		// 	block, err := strconv.Atoi(string(buf))
		// 	if err == nil {
		// 		if uint32(block) == challenge.Start {
		// 			n.Log.Chal("info", fmt.Sprintf("Already challenged: %v", challenge.Start))
		// 			time.Sleep(time.Minute)
		// 			continue
		// 		}
		// 	}
		// }

		idleSiama, _, qslice, err = n.idleAggrProof(key, chal.NetSnapshot.Random_index_list, chal.NetSnapshot.Random, chal.NetSnapshot.Start)
		if err != nil {
			n.Log.Chal("err", fmt.Sprintf("[idleAggrProof] %v", err))
			continue
		}

		serviceSigma, _, err = n.serviceAggrProof(key, qslice, chal.NetSnapshot.Start)
		if err != nil {
			n.Log.Chal("err", fmt.Sprintf("[serviceAggrProof] %v", err))
			continue
		}

		n.Cach.Put([]byte(Cach_prefix_idleSiama), []byte(idleSiama))
		n.Cach.Put([]byte(Cach_prefix_serviceSiama), []byte(serviceSigma))

		// todo: report proof
		// txhash, err = n.Cli.Chain.ReportProof(idleSiama, serviceSigma)
		// if err != nil {
		// 	n.Log.Chal("err", fmt.Sprintf("[ReportProof] %v", err))
		// 	continue
		// }
		// fmt.Println("txhash:", txhash)
		// err = n.Cach.Put([]byte(Cach_AggrProof_Report), []byte(fmt.Sprintf("%v", challenge.Start)))
		// if err != nil {

		// }
		_ = txhash

		idleProofFileHash, _ = utils.CalcPathSHA256Bytes(n.Cli.IproofFile)
		serviceProofFileHash, _ = utils.CalcPathSHA256Bytes(n.Cli.SproofFile)
		err = n.proofAsigmentInfo(idleProofFileHash, serviceProofFileHash, chal.NetSnapshot.Random_index_list, chal.NetSnapshot.Random)
		if err != nil {
			fmt.Println("++err:", err)
		}
		select {}
		time.Sleep(time.Minute)
	}
}

func (n *Node) proofAsigmentInfo(ihash, shash []byte, randomIndexList []uint32, random [][]byte) error {
	var err error
	var proof []chain.ProofAssignmentInfo
	var teeAsigned []byte
	var peerid peer.ID
	teelist, err := n.Cli.Chain.QueryTeeInfoList()
	if err != nil {
		return err
	}

	for _, v := range teelist {
		proof, err = n.Cli.Chain.QueryTeeAssignedProof(v.ControllerAccount[:])
		if err != nil {
			fmt.Println("err:", err)
			continue
		}

		for i := 0; i < len(proof); i++ {
			if chain.CompareSlice(proof[i].SnapShot.Miner[:], n.Cfg.GetPublickey()) {
				teeAsigned = v.ControllerAccount[:]
				peerid, err = peer.Decode(base58.Encode([]byte(string(v.PeerId[:]))))
				if err != nil {
					return err
				}
				break
			}
		}
	}
	_ = peerid
	if teeAsigned == nil {
		fmt.Println("proof not assigned:")
		return fmt.Errorf("proof not assigned")
	}

	var qslice = make([]*pb.Qslice, len(randomIndexList))
	for k, v := range randomIndexList {
		qslice[k] = new(pb.Qslice)
		qslice[k].I = uint64(v)
		qslice[k].V = random[k]
	}
	sign, err := n.Cli.Sign(n.Cli.PeerId)
	if err != nil {
		fmt.Println("err2:", err)
		return err
	}
	pid, _ := peer.Decode(configs.BootPeerId)
	code, err := n.Cli.AggrProofProtocol.AggrProofReq(pid, ihash, shash, qslice, n.Cfg.GetPublickey(), sign)
	if err != nil || code != 0 {
		return errors.New("AggrProofReq failed")
	}

	idleProofFileHashs, _ := utils.CalcPathSHA256(n.Cli.IproofFile)
	serviceProofFileHashs, _ := utils.CalcPathSHA256(n.Cli.SproofFile)

	err = errors.New("123")
	for err != nil {
		code, err = n.Cli.FileProtocol.FileReq(pid, idleProofFileHashs, pb.FileType_IdleMu, n.Cli.IproofFile)
		// if err != nil || code != 0 {
		// 	return errors.New("Idle FileReq failed")
		// }
		time.Sleep(time.Second * 5)
	}

	code, err = n.Cli.FileProtocol.FileReq(pid, serviceProofFileHashs, pb.FileType_CustomMu, n.Cli.SproofFile)
	if err != nil || code != 0 {
		return errors.New("Idle FileReq failed")
	}
	return nil
}

func (n *Node) idleAggrProof(key *proof.RSAKeyPair, randomIndexList []uint32, random [][]byte, start uint32) (string, string, []proof.QElement, error) {
	if len(randomIndexList) != len(random) {
		return "", "", nil, fmt.Errorf("invalid random length")
	}

	idleRoothashs, err := n.Cach.QueryPrefixKeyListByHeigh(Cach_prefix_idle, start)
	if err != nil {
		return "", "", nil, err
	}

	var buf []byte
	var ptags []proof.Tag = make([]proof.Tag, 0)
	var ptag proof.Tag
	var actualCount int
	var pf ProofFileType
	var proveResponse proof.GenProofResponse

	pf.Names = make([]string, len(idleRoothashs))
	pf.Us = make([]string, len(idleRoothashs))
	pf.Mus = make([]string, len(idleRoothashs))

	var qslice = make([]proof.QElement, len(randomIndexList))
	for k, v := range randomIndexList {
		qslice[k].I = int64(v)
		qslice[k].V = new(big.Int).SetBytes(random[k]).String()
	}

	timeout := time.NewTicker(time.Duration(time.Minute))
	defer timeout.Stop()

	for i := int(0); i < len(idleRoothashs); i++ {
		idleTagPath := filepath.Join(n.Cli.IdleTagDir, idleRoothashs[i]+".tag")
		//fmt.Println("idleTagPath:", idleTagPath)
		buf, err = os.ReadFile(idleTagPath)
		if err != nil {
			//fmt.Println("ReadFile", idleTagPath, "err: ", err)
			continue
		}

		var tag pb.Tag
		err = json.Unmarshal(buf, &tag)
		if err != nil {
			fmt.Println("Unmarshal err:", err)
			continue
		}

		matrix, _, err := proof.SplitByN(filepath.Join(n.Cli.IdleDataDir, idleRoothashs[i]), int64(len(tag.T.Phi)))
		if err != nil {
			fmt.Println("SplitByN err:", err)
			continue
		}

		ptag.T.Name = tag.T.Name
		ptag.T.Phi = tag.T.Phi
		ptag.T.U = tag.T.U
		ptag.PhiHash = tag.PhiHash
		ptag.Attest = tag.Attest

		proveResponseCh := key.GenProof(qslice, nil, ptag, matrix)
		timeout.Reset(time.Minute)
		select {
		case proveResponse = <-proveResponseCh:
		case <-timeout.C:
			proveResponse.StatueMsg.StatusCode = 0
		}

		if proveResponse.StatueMsg.StatusCode != proof.Success {
			continue
		}

		ptags = append(ptags, ptag)
		pf.Names[actualCount] = tag.T.Name
		pf.Us[actualCount] = tag.T.U
		pf.Mus[actualCount] = proveResponse.MU
		actualCount++
	}

	sigma := key.AggrGenProof(qslice, ptags)

	pf.Names = pf.Names[:actualCount]
	pf.Us = pf.Us[:actualCount]
	pf.Mus = pf.Mus[:actualCount]
	pf.Sigma = sigma

	//
	buf, err = json.Marshal(&pf)
	if err != nil {
		return "", "", nil, err
	}
	f, err := os.OpenFile(n.Cli.IproofFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return "", "", nil, err
	}
	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	_, err = f.Write(buf)
	if err != nil {
		return "", "", nil, err
	}

	err = f.Sync()
	if err != nil {
		return "", "", nil, err
	}

	f.Close()
	f = nil

	hash, err := utils.CalcPathSHA256(n.Cli.IproofFile)
	if err != nil {
		return "", "", nil, err
	}

	return sigma, hash, qslice, nil
}

func (n *Node) serviceAggrProof(key *proof.RSAKeyPair, qslice []proof.QElement, start uint32) (string, string, error) {
	n.Cach.Put([]byte(Cach_prefix_metadata+"bc1f81f9de240490aae56c322987c83184c53c59c74248675c6016f4a1940d8d"), []byte(fmt.Sprintf("%v", 18589)))

	serviceRoothashs, err := n.Cach.QueryPrefixKeyListByHeigh(Cach_prefix_metadata, start)
	if err != nil {
		return "", "", err
	}

	var buf []byte

	var pf ProofFileType
	var ptags []proof.Tag = make([]proof.Tag, 0)
	var ptag proof.Tag
	var proveResponse proof.GenProofResponse

	pf.Names = make([]string, 0)
	pf.Us = make([]string, 0)
	pf.Mus = make([]string, 0)

	timeout := time.NewTicker(time.Duration(time.Minute))
	defer timeout.Stop()
	for i := int(0); i < len(serviceRoothashs); i++ {
		files, err := utils.DirFiles(filepath.Join(n.Cli.FileDir, serviceRoothashs[i]), 0)
		if err != nil {
			continue
		}
		//fmt.Println("service files:", files)
		time.Sleep(time.Second * 3)
		for j := 0; j < len(files); j++ {
			serviceTagPath := filepath.Join(n.Cli.ServiceTagDir, filepath.Base(files[j])+".tag")
			//fmt.Println("serviceTagPath: ", serviceTagPath)
			buf, err = os.ReadFile(serviceTagPath)
			if err != nil {
				//fmt.Println("ReadFile", serviceTagPath, "err: ", err)
				continue
			}
			var tag pb.Tag
			err = json.Unmarshal(buf, &tag)
			if err != nil {
				fmt.Println("Unmarshal", serviceTagPath, "err: ", err)
				continue
			}
			matrix, _, err := proof.SplitByN(files[j], int64(len(tag.T.Phi)))
			if err != nil {
				fmt.Println("SplitByN", files[j], "err: ", err)
				continue
			}

			ptag.T.Name = tag.T.Name
			ptag.T.Phi = tag.T.Phi
			ptag.T.U = tag.T.U
			ptag.PhiHash = tag.PhiHash
			ptag.Attest = tag.Attest

			proveResponseCh := key.GenProof(qslice, nil, ptag, matrix)
			timeout.Reset(time.Minute)
			select {
			case proveResponse = <-proveResponseCh:
			case <-timeout.C:
				proveResponse.StatueMsg.StatusCode = 0
			}

			if proveResponse.StatueMsg.StatusCode != proof.Success {
				fmt.Println("GenProof  err: ", proveResponse.StatueMsg.StatusCode)
				continue
			}

			ptags = append(ptags, ptag)
			pf.Names = append(pf.Names, tag.T.Name)
			pf.Us = append(pf.Us, tag.T.U)
			pf.Mus = append(pf.Mus, proveResponse.MU)
		}
	}

	sigma := key.AggrGenProof(qslice, ptags)
	pf.Sigma = sigma

	buf, err = json.Marshal(&pf)
	if err != nil {
		return "", "", err
	}
	f, err := os.OpenFile(n.Cli.SproofFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return "", "", err
	}
	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	_, err = f.Write(buf)
	if err != nil {
		return "", "", err
	}
	err = f.Sync()
	if err != nil {
		return "", "", err
	}
	f.Close()
	f = nil

	hash, err := utils.CalcPathSHA256(n.Cli.SproofFile)
	if err != nil {
		return "", "", err
	}

	return sigma, hash, nil
}
