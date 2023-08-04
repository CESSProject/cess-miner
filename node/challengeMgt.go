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
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/pkg/cache"
	"github.com/CESSProject/cess-bucket/pkg/proof"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
)

func (n *Node) poisChallenge(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

}

// challengeMgr
// func (n *Node) challengeMgt(ch chan<- bool) {
// 	defer func() {
// 		ch <- true
// 		if err := recover(); err != nil {
// 			n.Pnc(utils.RecoverError(err))
// 		}
// 	}()

// 	var err error
// 	var recordErr string

// 	n.Chal("info", ">>>>> start challengeMgt <<<<<")

// 	tick := time.NewTicker(time.Minute)
// 	defer tick.Stop()

// 	for {
// 		select {
// 		case <-tick.C:
// 			if n.GetChainState() {
// 				err = n.pChallenge()
// 				if err != nil {
// 					if recordErr != err.Error() {
// 						n.Chal("err", err.Error())
// 						recordErr = err.Error()
// 					}
// 				}
// 			} else {
// 				if recordErr != pattern.ERR_RPC_CONNECTION.Error() {
// 					n.Chal("err", pattern.ERR_RPC_CONNECTION.Error())
// 					recordErr = pattern.ERR_RPC_CONNECTION.Error()
// 				}
// 			}
// 		}
// 	}
// }

func (n *Node) pChallenge() error {
	var err error
	var haveChallenge bool
	var challenge pattern.ChallengeSnapshot

	challenge, err = n.QueryChallengeSt()
	if err != nil {
		return errors.Wrapf(err, "[QueryChallengeSnapshot]")
	}

	for _, v := range challenge.MinerSnapshot {
		if n.GetSignatureAcc() == v.Miner {
			haveChallenge = true
			break
		}
	}

	latestBlock, err := n.QueryBlockHeight("")
	if err != nil {
		return errors.Wrapf(err, "[QueryBlockHeight]")
	}

	challExpiration, err := n.QueryChallengeExpiration()
	if err != nil {
		return errors.Wrapf(err, "[QueryChallengeExpiration]")
	}

	if challExpiration <= latestBlock {
		haveChallenge = false
	}

	var b []byte
	var tempInt int
	var peerid peer.ID

	if !haveChallenge {
		b, err = n.Get([]byte(Cach_AggrProof_Transfered))
		if err != nil {
			if err == cache.NotFound {
				err = n.transferProof(challenge)
				if err != nil {
					return errors.Wrapf(err, "[transferProof]")
				}
				return nil
			}
		}

		temp := strings.Split(string(b), "_")
		if len(temp) <= 1 {
			n.Delete([]byte(Cach_AggrProof_Transfered))
			err = n.transferProof(challenge)
			if err != nil {
				return errors.Wrapf(err, "[transferProof]")
			}
			return nil
		}

		peerid, _, err = n.queryProofAssignedTee()
		if err != nil {
			return errors.Wrapf(err, "[queryProofAssignedTee]")
		}

		tempInt, err = strconv.Atoi(temp[1])
		if err != nil {
			n.Delete([]byte(Cach_AggrProof_Transfered))
			err = n.transferProof(challenge)
			if err != nil {
				return errors.Wrapf(err, "[transferProof]")
			}
			return nil
		}

		if uint32(tempInt) != challenge.NetSnapshot.Start || peerid.Pretty() != temp[0] {
			err = n.transferProof(challenge)
			if err != nil {
				return errors.Wrapf(err, "[transferProof]")
			}
		}
		return nil
	}

	n.Delete([]byte(Cach_AggrProof_Transfered))

	n.Chal("info", fmt.Sprintf("Start processing challenges: %v", challenge.NetSnapshot.Start))

	var qslice = make([]proof.QElement, len(challenge.NetSnapshot.Random_index_list))
	for k, v := range challenge.NetSnapshot.Random_index_list {
		qslice[k].I = int64(v)
		qslice[k].V = new(big.Int).SetBytes(challenge.NetSnapshot.Random[k]).String()
	}

	err = n.saveRandom(challenge)
	if err != nil {
		n.Chal("err", fmt.Sprintf("Save challenge random err: %v", err))
	}

	n.Chal("info", "Save challenge random suc")

	var idleSiama string
	var serviceSigma string

	b, err = n.Get([]byte(Cach_IdleChallengeBlock))
	if err != nil {
		idleSiama, err = n.idleAggrProof(qslice, challenge.NetSnapshot.Start)
		if err != nil {
			return errors.Wrapf(err, "[idleAggrProof]")
		}
		n.Put([]byte(Cach_prefix_idleSiama), []byte(idleSiama))
		n.Put([]byte(Cach_IdleChallengeBlock), []byte(fmt.Sprintf("%d", challenge.NetSnapshot.Start)))
		n.Chal("info", fmt.Sprintf("Idle data aggregation proof: %s", idleSiama))
	} else {
		tempInt, err = strconv.Atoi(string(b))
		if err != nil {
			n.Delete([]byte(Cach_IdleChallengeBlock))
			idleSiama, err = n.idleAggrProof(qslice, challenge.NetSnapshot.Start)
			if err != nil {
				return errors.Wrapf(err, "[idleAggrProof]")
			}
			n.Put([]byte(Cach_prefix_idleSiama), []byte(idleSiama))
			n.Put([]byte(Cach_IdleChallengeBlock), []byte(fmt.Sprintf("%d", challenge.NetSnapshot.Start)))
			n.Chal("info", fmt.Sprintf("Idle data aggregation proof: %s", idleSiama))
		} else {
			if uint32(tempInt) != challenge.NetSnapshot.Start {
				idleSiama, err = n.idleAggrProof(qslice, challenge.NetSnapshot.Start)
				if err != nil {
					return errors.Wrapf(err, "[idleAggrProof]")
				}
				n.Put([]byte(Cach_prefix_idleSiama), []byte(idleSiama))
				n.Put([]byte(Cach_IdleChallengeBlock), []byte(fmt.Sprintf("%d", challenge.NetSnapshot.Start)))
				n.Chal("info", fmt.Sprintf("Idle data aggregation proof: %s", idleSiama))
			} else {
				b, err = n.Get([]byte(Cach_prefix_idleSiama))
				if err != nil {
					idleSiama, err = n.idleAggrProof(qslice, challenge.NetSnapshot.Start)
					if err != nil {
						return errors.Wrapf(err, "[idleAggrProof]")
					}
					n.Put([]byte(Cach_prefix_idleSiama), []byte(idleSiama))
					n.Put([]byte(Cach_IdleChallengeBlock), []byte(fmt.Sprintf("%d", challenge.NetSnapshot.Start)))
					n.Chal("info", fmt.Sprintf("Idle data aggregation proof: %s", idleSiama))
				} else {
					idleSiama = string(b)
				}
			}
		}
	}

	b, err = n.Get([]byte(Cach_ServiceChallengeBlock))
	if err != nil {
		serviceSigma, err = n.serviceAggrProof(qslice, challenge.NetSnapshot.Start)
		if err != nil {
			return errors.Wrapf(err, "[serviceAggrProof]")
		}
		n.Put([]byte(Cach_prefix_serviceSiama), []byte(serviceSigma))
		n.Put([]byte(Cach_ServiceChallengeBlock), []byte(fmt.Sprintf("%d", challenge.NetSnapshot.Start)))
		n.Chal("info", fmt.Sprintf("Service data aggregation proof: %s", serviceSigma))
	} else {
		tempInt, err = strconv.Atoi(string(b))
		if err != nil {
			n.Delete([]byte(Cach_ServiceChallengeBlock))
			serviceSigma, err = n.serviceAggrProof(qslice, challenge.NetSnapshot.Start)
			if err != nil {
				return errors.Wrapf(err, "[serviceAggrProof]")
			}
			n.Put([]byte(Cach_prefix_serviceSiama), []byte(serviceSigma))
			n.Put([]byte(Cach_ServiceChallengeBlock), []byte(fmt.Sprintf("%d", challenge.NetSnapshot.Start)))
			n.Chal("info", fmt.Sprintf("Service data aggregation proof: %s", serviceSigma))
		} else {
			if uint32(tempInt) != challenge.NetSnapshot.Start {
				serviceSigma, err = n.serviceAggrProof(qslice, challenge.NetSnapshot.Start)
				if err != nil {
					return errors.Wrapf(err, "[serviceAggrProof]")
				}
				n.Put([]byte(Cach_prefix_serviceSiama), []byte(serviceSigma))
				n.Put([]byte(Cach_ServiceChallengeBlock), []byte(fmt.Sprintf("%d", challenge.NetSnapshot.Start)))
				n.Chal("info", fmt.Sprintf("Service data aggregation proof: %s", serviceSigma))
			} else {
				b, err = n.Get([]byte(Cach_prefix_serviceSiama))
				if err != nil {
					serviceSigma, err = n.idleAggrProof(qslice, challenge.NetSnapshot.Start)
					if err != nil {
						return errors.Wrapf(err, "[serviceAggrProof]")
					}
					n.Put([]byte(Cach_prefix_serviceSiama), []byte(serviceSigma))
					n.Put([]byte(Cach_ServiceChallengeBlock), []byte(fmt.Sprintf("%d", challenge.NetSnapshot.Start)))
					n.Chal("info", fmt.Sprintf("Service data aggregation proof: %s", serviceSigma))
				} else {
					serviceSigma = string(b)
				}
			}
		}
	}

	if idleSiama == "" && serviceSigma == "" {
		return errors.New("Both proofs are empty")
	}

	txhash, err := n.ReportProof(idleSiama, serviceSigma)
	if err != nil {
		return errors.Wrapf(err, "[ReportProof]")
	}

	n.Chal("info", fmt.Sprintf("Reported challenge results: %v", txhash))

	time.Sleep(pattern.BlockInterval * 3)

	err = n.transferProof(challenge)
	if err != nil {
		return errors.Wrapf(err, "[transferProof]")
	}
	return nil
}

func (n *Node) transferProof(challenge pattern.ChallengeSnapshot) error {
	idleProofFileHash, err := sutils.CalcPathSHA256Bytes(n.GetDirs().IproofFile)
	if err != nil {
		return errors.Wrapf(err, "[CalcPathSHA256Bytes]")
	}
	serviceProofFileHash, err := sutils.CalcPathSHA256Bytes(n.GetDirs().SproofFile)
	if err != nil {
		return errors.Wrapf(err, "[CalcPathSHA256Bytes]")
	}
	peerid, code, err := n.proofAssignedInfo(idleProofFileHash, serviceProofFileHash, challenge.NetSnapshot.Random_index_list, challenge.NetSnapshot.Random)
	if err != nil || code != 0 {
		return errors.Wrapf(err, "[proofAsigmentInfo]")
	}
	err = n.Put([]byte(Cach_AggrProof_Transfered), []byte(fmt.Sprintf("%s_%v", peerid, challenge.NetSnapshot.Start)))
	if err != nil {
		return errors.Wrapf(err, "[PutCache]")
	}
	return nil
}

func (n *Node) proofAssignedInfo(ihash, shash []byte, randomIndexList []uint32, random [][]byte) (string, uint32, error) {
	var err error
	var code uint32
	var teeAsigned []byte
	var peerid peer.ID
	peerid, teeAsigned, err = n.queryProofAssignedTee()
	if err != nil {
		return "", code, errors.Wrapf(err, "[queryProofAssignedTee]")
	}

	if teeAsigned == nil {
		return "", code, errors.New("proof not assigned")
	}

	var qslice = make([]*pb.Qslice, len(randomIndexList))
	for k, v := range randomIndexList {
		qslice[k] = new(pb.Qslice)
		qslice[k].I = uint64(v)
		qslice[k].V = random[k]
	}

	sign, err := n.Sign(n.GetPeerPublickey())
	if err != nil {
		return "", code, errors.Wrapf(err, "[Sign]")
	}

	addr, ok := n.GetPeer(peerid.Pretty())
	if !ok {
		addr, err = n.DHTFindPeer(peerid.Pretty())
		if err != nil {
			return "", code, fmt.Errorf("No verification proof tee found: %s", peerid.Pretty())
		}
	}

	err = n.Connect(n.GetCtxQueryFromCtxCancel(), addr)
	if err != nil {
		return "", code, fmt.Errorf("Failed to connect to verification proof tee: %s", peerid.Pretty())
	}

	code, err = n.AggrProofReq(peerid, ihash, shash, qslice, n.GetStakingPublickey(), sign)
	if err != nil || code != 0 {
		return "", code, errors.New(fmt.Sprintf("AggrProofReq to %s err: %v, code: %d", peerid.Pretty(), err, code))

	}
	n.Chal("info", fmt.Sprintf("Aggr proof response suc: %s", peerid.Pretty()))

	idleProofFileHashs, _ := sutils.CalcPathSHA256(n.GetDirs().IproofFile)
	serviceProofFileHashs, _ := sutils.CalcPathSHA256(n.GetDirs().SproofFile)

	code, err = n.FileReq(peerid, idleProofFileHashs, pb.FileType_IdleMu, n.GetDirs().IproofFile)
	if err != nil || code != 0 {
		return "", code, errors.New(fmt.Sprintf("FileReq FileType_IdleMu err: %v,code: %d", err, code))
	}
	n.Chal("info", fmt.Sprintf("Aggr proof idle file response suc: %s", peerid.Pretty()))

	code, err = n.FileReq(peerid, serviceProofFileHashs, pb.FileType_CustomMu, n.GetDirs().SproofFile)
	if err != nil || code != 0 {
		return peerid.Pretty(), code, errors.New(fmt.Sprintf("FileReq FileType_IdleMu err: %v,code: %d", err, code))
	}

	n.Chal("info", fmt.Sprintf("Aggr proof service file response suc: %s", peerid.Pretty()))
	return peerid.Pretty(), 0, nil
}

func (n *Node) idleAggrProof(qslice []proof.QElement, start uint32) (string, error) {
	idleRoothashs, err := n.QueryPrefixKeyListByHeigh(Cach_prefix_idle, start)
	if err != nil {
		return "", err
	}

	var buf []byte
	var actualCount int
	var pf ProofFileType
	var proveResponse proof.GenProofResponse
	var sigma string
	var tag pb.Tag

	pf.Names = make([]string, len(idleRoothashs))
	pf.Us = make([]string, len(idleRoothashs))
	pf.Mus = make([]string, len(idleRoothashs))

	timeout := time.NewTicker(time.Duration(time.Minute))
	defer timeout.Stop()

	for i := int(0); i < len(idleRoothashs); i++ {
		idleTagPath := filepath.Join(n.GetDirs().IdleTagDir, idleRoothashs[i]+".tag")
		buf, err = os.ReadFile(idleTagPath)
		if err != nil {
			n.Chal("err", fmt.Sprintf("Idletag not found: %v", idleTagPath))
			continue
		}

		err = json.Unmarshal(buf, &tag)
		if err != nil {
			n.Chal("err", fmt.Sprintf("Unmarshal err: %v", err))
			continue
		}

		matrix, _, err := proof.SplitByN(filepath.Join(n.GetDirs().IdleDataDir, idleRoothashs[i]), int64(len(tag.T.Phi)))
		if err != nil {
			n.Delete([]byte(Cach_prefix_idle + idleRoothashs[i]))
			os.Remove(idleTagPath)
			n.Chal("err", fmt.Sprintf("SplitByN err: %v", err))
			continue
		}

		proveResponseCh := n.key.GenProof(qslice, nil, tag.T.Phi, matrix)
		timeout.Reset(time.Minute)
		select {
		case proveResponse = <-proveResponseCh:
		case <-timeout.C:
			proveResponse.StatueMsg.StatusCode = 0
		}

		if proveResponse.StatueMsg.StatusCode != proof.Success {
			continue
		}

		sigmaTemp, ok := n.key.AggrAppendProof(sigma, qslice, tag.T.Phi)
		if !ok {
			continue
		}
		sigma = sigmaTemp
		pf.Names[actualCount] = tag.T.Name
		pf.Us[actualCount] = tag.T.U
		pf.Mus[actualCount] = proveResponse.MU
		actualCount++
	}

	pf.Names = pf.Names[:actualCount]
	pf.Us = pf.Us[:actualCount]
	pf.Mus = pf.Mus[:actualCount]
	pf.Sigma = sigma

	//
	buf, err = json.Marshal(&pf)
	if err != nil {
		return "", err
	}
	f, err := os.OpenFile(n.GetDirs().IproofFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
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

func (n *Node) serviceAggrProof(qslice []proof.QElement, start uint32) (string, error) {
	serviceRoothashs, err := n.QueryPrefixKeyListByHeigh(Cach_prefix_metadata, start)
	if err != nil {
		return "", err
	}

	var buf []byte
	var sigma string
	var pf ProofFileType
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

			proveResponseCh := n.key.GenProof(qslice, nil, tag.T.Phi, matrix)
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

			sigmaTemp, ok := n.key.AggrAppendProof(sigma, qslice, tag.T.Phi)
			if !ok {
				continue
			}
			sigma = sigmaTemp
			pf.Names = append(pf.Names, tag.T.Name)
			pf.Us = append(pf.Us, tag.T.U)
			pf.Mus = append(pf.Mus, proveResponse.MU)
		}
	}
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

func (n *Node) saveRandom(challenge pattern.ChallengeSnapshot) error {
	randfilePath := filepath.Join(n.GetDirs().ProofDir, fmt.Sprintf("random.%d", challenge.NetSnapshot.Start))
	fstat, err := os.Stat(randfilePath)
	if err == nil && fstat.Size() > 0 {
		return nil
	}
	var rd RandomList
	rd.Index = challenge.NetSnapshot.Random_index_list
	rd.Random = challenge.NetSnapshot.Random
	buff, err := json.Marshal(&rd)
	if err != nil {
		return errors.Wrapf(err, "[json.Marshal]")
	}

	f, err := os.OpenFile(randfilePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "[OpenFile]")
	}
	defer f.Close()
	_, err = f.Write(buff)
	if err != nil {
		return errors.Wrapf(err, "[Write]")
	}
	return f.Sync()
}

func (n *Node) queryProofAssignedTee() (peer.ID, []byte, error) {
	var err error

	tees := n.GetAllTeeWorkAccount()

	for _, v := range tees {
		puk, err := sutils.ParsingPublickey(v)
		if err != nil {
			continue
		}
		proof, err := n.QueryTeeAssignedProof(puk)
		if err != nil {
			continue
		}

		for i := 0; i < len(proof); i++ {
			if sutils.CompareSlice(proof[i].SnapShot.Miner[:], n.GetStakingPublickey()) {
				teepeerid, ok := n.GetTeeWork(v)
				if !ok {
					continue
				}
				peerid, err := peer.Decode(base58.Encode(teepeerid))
				if err != nil {
					return "", nil, errors.Wrapf(err, "[peer.Decode]")
				}
				return peerid, puk, nil
			}
		}
	}
	return peer.ID(""), nil, err
}
