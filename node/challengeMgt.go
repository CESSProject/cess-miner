/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/proof"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/CESSProject/sdk-go/core/pattern"
	sutils "github.com/CESSProject/sdk-go/core/utils"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mr-tron/base58"
)

// challengeMgr
func (n *Node) challengeMgt(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	var txhash string
	var idleSiama string
	var serviceSigma string
	var qslice []proof.QElement

	n.Chal("info", ">>>>> Start challengeMgt task")

	for {
		pubkey, err := n.QueryTeePodr2Puk()
		if err != nil {
			configs.Err(fmt.Sprintf("[QueryTeePodr2Puk] %v", err))
			time.Sleep(pattern.BlockInterval)
			continue
		}
		configs.Ok("Initialize key successfully")
		n.Key.SetKeyN(pubkey)
		break
	}

	for {
		chal, err := n.QueryChallenge(n.GetStakingPublickey())
		if err != nil {
			time.Sleep(time.Minute)
			continue
		}

		n.Chal("info", fmt.Sprintf("Challenge start: %v", chal.Start))
		n.Chal("info", fmt.Sprintf("Challenge randomindex: %v random length: %v", len(chal.RandomIndexList), len(chal.Random)))

		buf, err := n.Get([]byte(Cach_AggrProof_Reported))
		if err == nil {
			block, err := strconv.Atoi(string(buf))
			if err == nil {
				if uint32(block) == chal.Start {
					n.Chal("info", fmt.Sprintf("Already challenged: %v", chal.Start))
					time.Sleep(time.Minute)
					continue
				}
			}
		}

		err = n.saveRandom(chal)
		if err != nil {
			n.Chal("err", fmt.Sprintf("saveRandom [%d] err: %v", chal.Start, err))
		}

		idleSiama, qslice, err = n.idleAggrProof(chal.RandomIndexList, chal.Random, chal.Start)
		if err != nil {
			n.Chal("err", fmt.Sprintf("[idleAggrProof] %v", err))
			continue
		}

		serviceSigma, err = n.serviceAggrProof(qslice, chal.Start)
		if err != nil {
			n.Chal("err", fmt.Sprintf("[serviceAggrProof] %v", err))
			continue
		}

		n.Put([]byte(Cach_prefix_idleSiama), []byte(idleSiama))
		n.Put([]byte(Cach_prefix_serviceSiama), []byte(serviceSigma))

		// report proof
		txhash, err = n.ReportProof(idleSiama, serviceSigma)
		if err != nil {
			n.Chal("err", fmt.Sprintf("[ReportProof] %v", err))
			continue
		}

		n.Chal("info", fmt.Sprintf("ReportProof %v", txhash))
		err = n.Put([]byte(Cach_AggrProof_Reported), []byte(fmt.Sprintf("%v", chal.Start)))
		if err != nil {
			n.Chal("err", fmt.Sprintf("Put Cach_AggrProof_Reported [%d] err: %v", chal.Start, err))
		}

		time.Sleep(pattern.BlockInterval)

		for {
			idleProofFileHash, _ := utils.CalcPathSHA256Bytes(n.GetDirs().IproofFile)
			serviceProofFileHash, _ := utils.CalcPathSHA256Bytes(n.GetDirs().SproofFile)
			err = n.proofAsigmentInfo(idleProofFileHash, serviceProofFileHash, chal.RandomIndexList, chal.Random)
			if err != nil {
				n.Chal("err", fmt.Sprintf("proofAsigmentInfo: %v", err))
				time.Sleep(time.Second * 30)
				continue
			}
			break
		}
		err = n.Put([]byte(Cach_AggrProof_Transfered), []byte(fmt.Sprintf("%v", chal.Start)))
		if err != nil {
			n.Chal("err", fmt.Sprintf("Put Cach_AggrProof_Transfered [%d] err: %v", chal.Start, err))
		}
	}
}

func (n *Node) proofAsigmentInfo(ihash, shash []byte, randomIndexList []uint32, random [][]byte) error {
	var err error
	var proof []pattern.ProofAssignmentInfo
	var teeAsigned []byte
	var peerid peer.ID
	teelist, err := n.QueryTeeInfoList()
	if err != nil {
		return err
	}

	for _, v := range teelist {
		proof, err = n.QueryTeeAssignedProof(v.ControllerAccount[:])
		if err != nil {
			continue
		}

		for i := 0; i < len(proof); i++ {
			if sutils.CompareSlice(proof[i].SnapShot.Miner[:], n.GetStakingPublickey()) {
				teeAsigned = v.ControllerAccount[:]
				peerid, err = peer.Decode(base58.Encode([]byte(string(v.PeerId[:]))))
				if err != nil {
					return err
				}
				break
			}
		}
	}

	if teeAsigned == nil {
		n.Chal("err", "proof not assigned")
		return fmt.Errorf("proof not assigned")
	}

	var qslice = make([]*pb.Qslice, len(randomIndexList))
	for k, v := range randomIndexList {
		qslice[k] = new(pb.Qslice)
		qslice[k].I = uint64(v)
		qslice[k].V = random[k]
	}
	sign, err := n.Sign(n.GetPeerPublickey())
	if err != nil {
		n.Chal("err", fmt.Sprintf("Sign err: %v", err))
		return err
	}

	for {
		code, err := n.AggrProofReq(peerid, ihash, shash, qslice, n.GetStakingPublickey(), sign)
		if err != nil || code != 0 {
			n.Chal("err", fmt.Sprintf("AggrProofReq err: %v, code: %d", err, code))
			time.Sleep(pattern.BlockInterval)
			continue
		}
		break
	}

	idleProofFileHashs, _ := utils.CalcPathSHA256(n.GetDirs().IproofFile)
	serviceProofFileHashs, _ := utils.CalcPathSHA256(n.GetDirs().SproofFile)

	for {
		code, err := n.FileReq(peerid, idleProofFileHashs, pb.FileType_IdleMu, n.GetDirs().IproofFile)
		if err != nil || code != 0 {
			n.Chal("err", fmt.Sprintf("FileType_IdleMu FileReq err: %v, code: %d", err, code))
			time.Sleep(pattern.BlockInterval)
			continue
		}
		break
	}

	for {
		code, err := n.FileReq(peerid, serviceProofFileHashs, pb.FileType_CustomMu, n.GetDirs().SproofFile)
		if err != nil || code != 0 {
			n.Chal("err", fmt.Sprintf("FileType_CustomMu FileReq err: %v, code: %d", err, code))
			time.Sleep(pattern.BlockInterval)
			continue
		}
		break
	}
	return nil
}

func (n *Node) idleAggrProof(randomIndexList []uint32, random [][]byte, start uint32) (string, []proof.QElement, error) {
	if len(randomIndexList) != len(random) {
		return "", nil, fmt.Errorf("invalid random length")
	}

	idleRoothashs, err := n.QueryPrefixKeyListByHeigh(Cach_prefix_idle, start)
	if err != nil {
		return "", nil, err
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
		idleTagPath := filepath.Join(n.GetDirs().IdleTagDir, idleRoothashs[i]+".tag")
		buf, err = os.ReadFile(idleTagPath)
		if err != nil {
			n.Chal("err", fmt.Sprintf("Idletag not found: %v", idleTagPath))
			continue
		}

		var tag pb.Tag
		err = json.Unmarshal(buf, &tag)
		if err != nil {
			n.Chal("err", fmt.Sprintf("Unmarshal err: %v", err))
			continue
		}

		matrix, _, err := proof.SplitByN(filepath.Join(n.GetDirs().IdleDataDir, idleRoothashs[i]), int64(len(tag.T.Phi)))
		if err != nil {
			n.Chal("err", fmt.Sprintf("SplitByN err: %v", err))
			continue
		}

		ptag.T.Name = tag.T.Name
		ptag.T.Phi = tag.T.Phi
		ptag.T.U = tag.T.U
		ptag.PhiHash = tag.PhiHash
		ptag.Attest = tag.Attest

		proveResponseCh := n.Key.GenProof(qslice, nil, ptag, matrix)
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

	sigma := n.Key.AggrGenProof(qslice, ptags)

	pf.Names = pf.Names[:actualCount]
	pf.Us = pf.Us[:actualCount]
	pf.Mus = pf.Mus[:actualCount]
	pf.Sigma = sigma

	//
	buf, err = json.Marshal(&pf)
	if err != nil {
		return "", nil, err
	}
	f, err := os.OpenFile(n.GetDirs().IproofFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return "", nil, err
	}
	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	_, err = f.Write(buf)
	if err != nil {
		return "", nil, err
	}

	err = f.Sync()
	if err != nil {
		return "", nil, err
	}

	f.Close()
	f = nil

	return sigma, qslice, nil
}

func (n *Node) serviceAggrProof(qslice []proof.QElement, start uint32) (string, error) {
	serviceRoothashs, err := n.QueryPrefixKeyListByHeigh(Cach_prefix_metadata, start)
	if err != nil {
		return "", err
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
		files, err := utils.DirFiles(filepath.Join(n.GetDirs().FileDir, serviceRoothashs[i]), 0)
		if err != nil {
			continue
		}

		for j := 0; j < len(files); j++ {
			serviceTagPath := filepath.Join(n.GetDirs().ServiceTagDir, filepath.Base(files[j])+".tag")
			buf, err = os.ReadFile(serviceTagPath)
			if err != nil {
				n.Chal("err", fmt.Sprintf("Servicetag not found: %v", serviceTagPath))
				continue
			}
			var tag pb.Tag
			err = json.Unmarshal(buf, &tag)
			if err != nil {
				n.Chal("err", fmt.Sprintf("Unmarshal %v err: %v", serviceTagPath, err))
				continue
			}
			matrix, _, err := proof.SplitByN(files[j], int64(len(tag.T.Phi)))
			if err != nil {
				n.Chal("err", fmt.Sprintf("SplitByN %v err: %v", serviceTagPath, err))
				continue
			}

			ptag.T.Name = tag.T.Name
			ptag.T.Phi = tag.T.Phi
			ptag.T.U = tag.T.U
			ptag.PhiHash = tag.PhiHash
			ptag.Attest = tag.Attest

			proveResponseCh := n.Key.GenProof(qslice, nil, ptag, matrix)
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

	sigma := n.Key.AggrGenProof(qslice, ptags)
	pf.Sigma = sigma

	buf, err = json.Marshal(&pf)
	if err != nil {
		return "", err
	}
	f, err := os.OpenFile(n.GetDirs().SproofFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return "", err
	}
	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	_, err = f.Write(buf)
	if err != nil {
		return "", err
	}
	err = f.Sync()
	if err != nil {
		return "", err
	}
	f.Close()
	f = nil

	return sigma, nil
}

func (n *Node) saveRandom(chal pattern.ChallengeInfo) error {
	randfilePath := filepath.Join(n.GetDirs().ProofDir, fmt.Sprintf("random.%d", chal.Start))
	fstat, err := os.Stat(randfilePath)
	if err == nil && fstat.Size() > 0 {
		return nil
	}
	var rd RandomList
	rd.Index = chal.RandomIndexList
	rd.Random = chal.Random
	buff, err := json.Marshal(&rd)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(randfilePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(buff)
	if err != nil {
		return err
	}
	return f.Sync()
}
