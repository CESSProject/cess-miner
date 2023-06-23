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

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/proof"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
)

// challengeMgr
func (n *Node) challengeMgt(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	var err error
	var txhash string
	var idleSiama string
	var serviceSigma string
	var peerid peer.ID
	var b []byte
	var chalStart int
	var chal pattern.ChallengeInfo
	var netinfo pattern.ChallengeSnapShot
	var qslice []proof.QElement

	n.Chal("info", ">>>>> start challengeMgt <<<<<")

	for {
		pubkey, err := n.QueryTeePodr2Puk()
		if err != nil {
			configs.Err(fmt.Sprintf("[QueryTeePodr2Puk] %v", err))
			time.Sleep(pattern.BlockInterval)
			continue
		}
		err = n.SetPublickey(pubkey)
		if err != nil {
			configs.Err(fmt.Sprintf("[SetPublickey] %v", err))
			time.Sleep(pattern.BlockInterval)
			continue
		}
		n.Chal("info", "Initialize key successfully")
		break
	}

	for {
		for n.GetChainState() {
			time.Sleep(pattern.BlockInterval * 5)
			chal, err = n.QueryChallenge(n.GetStakingPublickey())
			if err != nil || chal.Start == uint32(0) {
				n.Chal("info", fmt.Sprintf("Did not find your own challenge information: %v", err))
				b, err = n.Get([]byte(Cach_AggrProof_Transfered))
				if err != nil {
					err = n.transferProof()
					if err != nil {
						n.Chal("err", err.Error())
					}
					continue
				}
				netinfo, err = n.QueryChallengeSnapshot()
				if err != nil {
					n.Chal("err", err.Error())
					continue
				}

				temp := strings.Split(string(b), "_")
				if len(temp) <= 1 {
					n.Delete([]byte(Cach_AggrProof_Transfered))
					err = n.transferProof()
					if err != nil {
						n.Chal("err", err.Error())
					}
					continue
				}

				peerid, _, err = n.queryProofAssignedTee()
				if err != nil {
					n.Chal("err", fmt.Sprintf("[queryProofAssignedTee] %v", err))
					continue
				}

				chalStart, err = strconv.Atoi(temp[1])
				if err != nil {
					n.Delete([]byte(Cach_AggrProof_Transfered))
					err = n.transferProof()
					if err != nil {
						n.Chal("err", err.Error())
					}
					continue
				}

				if chalStart != int(netinfo.NetSnapshot.Start) || peerid.Pretty() != temp[0] {
					err = n.transferProof()
					if err != nil {
						n.Chal("err", err.Error())
					}
					continue
				}
				n.Chal("info", "Proof was transmitted")
				time.Sleep(time.Minute)
				continue
			}

			n.Chal("info", fmt.Sprintf("Challenge start: %v", chal.Start))

			err = n.saveRandom(chal)
			if err != nil {
				n.Chal("err", fmt.Sprintf("saveRandom [%d] err: %v", chal.Start, err))
			}

			n.Chal("info", fmt.Sprintf("saveRandom suc: %v", chal.Start))

			idleSiama, qslice, err = n.idleAggrProof(chal.RandomIndexList, chal.Random, chal.Start)
			if err != nil {
				n.Chal("err", fmt.Sprintf("[idleAggrProof] %v", err))
				continue
			}
			n.Chal("info", fmt.Sprintf("idleAggrProof suc: %v", idleSiama))

			serviceSigma, err = n.serviceAggrProof(qslice, chal.Start)
			if err != nil {
				n.Chal("err", fmt.Sprintf("[serviceAggrProof] %v", err))
				continue
			}
			n.Chal("info", fmt.Sprintf("serviceAggrProof suc: %v", serviceSigma))

			n.Put([]byte(Cach_prefix_idleSiama), []byte(idleSiama))
			n.Put([]byte(Cach_prefix_serviceSiama), []byte(serviceSigma))

			// report proof
			txhash, err = n.ReportProof(idleSiama, serviceSigma)
			if err != nil {
				n.Chal("err", fmt.Sprintf("[ReportProof] %v", err))
				continue
			}

			n.Chal("info", fmt.Sprintf("Submit proof suc: %v", txhash))

			err = n.Put([]byte(Cach_AggrProof_Reported), []byte(fmt.Sprintf("%v", chal.Start)))
			if err != nil {
				n.Chal("err", fmt.Sprintf("Put Cach_AggrProof_Reported [%d] err: %v", chal.Start, err))
			}

			time.Sleep(pattern.BlockInterval)

			err = n.transferProof()
			if err != nil {
				n.Chal("err", fmt.Sprintf("Put Cach_AggrProof_Reported [%d] err: %v", chal.Start, err))
				continue
			}
		}
		time.Sleep(pattern.BlockInterval)
	}
}

func (n *Node) pChallenge() error {
	var err error
	var txhash string
	var idleSiama string
	var serviceSigma string
	var peerid peer.ID
	var b []byte
	var chalStart int
	var chal pattern.ChallengeInfo
	var netinfo pattern.ChallengeSnapShot

	var tempInt int
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

	if !haveChallenge {
		b, err = n.Get([]byte(Cach_AggrProof_Transfered))
		if err != nil {
			err = n.transferProof()
			if err != nil {
				n.Chal("err", err.Error())
			}
			continue
		}
		netinfo, err = n.QueryChallengeSnapshot()
		if err != nil {
			n.Chal("err", err.Error())
			continue
		}

		temp := strings.Split(string(b), "_")
		if len(temp) <= 1 {
			n.Delete([]byte(Cach_AggrProof_Transfered))
			err = n.transferProof()
			if err != nil {
				n.Chal("err", err.Error())
			}
			continue
		}

		peerid, _, err = n.queryProofAssignedTee()
		if err != nil {
			n.Chal("err", fmt.Sprintf("[queryProofAssignedTee] %v", err))
			continue
		}

		chalStart, err = strconv.Atoi(temp[1])
		if err != nil {
			n.Delete([]byte(Cach_AggrProof_Transfered))
			err = n.transferProof()
			if err != nil {
				n.Chal("err", err.Error())
			}
			continue
		}

		if chalStart != int(netinfo.NetSnapshot.Start) || peerid.Pretty() != temp[0] {
			err = n.transferProof()
			if err != nil {
				n.Chal("err", err.Error())
			}
			continue
		}
		n.Chal("info", "Proof was transmitted")
		time.Sleep(time.Minute)
		continue
	}

	n.Delete([]byte(Cach_AggrProof_Transfered))

	n.Chal("info", fmt.Sprintf("Start processing challenges: %v", chal.Start))

	var qslice = make([]proof.QElement, len(challenge.NetSnapshot.Random_index_list))
	for k, v := range challenge.NetSnapshot.Random_index_list {
		qslice[k].I = int64(v)
		qslice[k].V = new(big.Int).SetBytes(challenge.NetSnapshot.Random[k]).String()
	}

	err = n.saveRandom(chal)
	if err != nil {
		n.Chal("err", fmt.Sprintf("Save challenge random err: %v", err))
	}

	n.Chal("info", "Save challenge random suc")

	b, err = n.Get([]byte(Cach_IdleChallengeBlock))
	if err != nil {
		idleSiama, qslice, err = n.idleAggrProof(qslice, chal.Start)
		if err != nil {
			return errors.Wrapf(err, "[idleAggrProof]")
		}
		n.Put([]byte(Cach_IdleChallengeBlock), []byte(fmt.Sprintf("%d", chal.Start)))
		n.Chal("info", fmt.Sprintf("Idle data aggregation proof: %s", idleSiama))
	} else {
		tempInt, err = strconv.Atoi(string(b))
		if err != nil {
			n.Delete([]byte(Cach_IdleChallengeBlock))
			idleSiama, qslice, err = n.idleAggrProof(chal.RandomIndexList, chal.Random, chal.Start)
			if err != nil {
				return errors.Wrapf(err, "[idleAggrProof]")
			}
			n.Put([]byte(Cach_IdleChallengeBlock), []byte(fmt.Sprintf("%d", chal.Start)))
			n.Chal("info", fmt.Sprintf("Idle data aggregation proof: %s", idleSiama))
		} else {
			if uint32(tempInt) != chal.Start {
				idleSiama, qslice, err = n.idleAggrProof(chal.RandomIndexList, chal.Random, chal.Start)
				if err != nil {
					return errors.Wrapf(err, "[idleAggrProof]")
				}
				n.Put([]byte(Cach_IdleChallengeBlock), []byte(fmt.Sprintf("%d", chal.Start)))
				n.Chal("info", fmt.Sprintf("Idle data aggregation proof: %s", idleSiama))
			}
		}
	}

	b, err = n.Get([]byte(Cach_ServiceChallengeBlock))
	if err != nil {
		serviceSigma, err = n.serviceAggrProof(qslice, chal.Start)
		if err != nil {
			return errors.Wrapf(err, "[serviceAggrProof]")
		}
		n.Put([]byte(Cach_ServiceChallengeBlock), []byte(fmt.Sprintf("%d", chal.Start)))
		n.Chal("info", fmt.Sprintf("Service data aggregation proof: %s", serviceSigma))
	} else {
		tempInt, err = strconv.Atoi(string(b))
		if err != nil {
			n.Delete([]byte(Cach_ServiceChallengeBlock))
			serviceSigma, err = n.serviceAggrProof(qslice, chal.Start)
			if err != nil {
				return errors.Wrapf(err, "[serviceAggrProof]")
			}
			n.Put([]byte(Cach_ServiceChallengeBlock), []byte(fmt.Sprintf("%d", chal.Start)))
			n.Chal("info", fmt.Sprintf("Service data aggregation proof: %s", serviceSigma))
		} else {
			if uint32(tempInt) != chal.Start {
				serviceSigma, err = n.serviceAggrProof(qslice, chal.Start)
				if err != nil {
					return errors.Wrapf(err, "[serviceAggrProof]")
				}
				n.Put([]byte(Cach_ServiceChallengeBlock), []byte(fmt.Sprintf("%d", chal.Start)))
				n.Chal("info", fmt.Sprintf("Service data aggregation proof: %s", serviceSigma))
			}
		}
	}

	n.Put([]byte(Cach_prefix_idleSiama), []byte(idleSiama))
	n.Put([]byte(Cach_prefix_serviceSiama), []byte(serviceSigma))

	txhash, err = n.ReportProof(idleSiama, serviceSigma)
	if err != nil {
		return errors.Wrapf(err, "[ReportProof]")
	}

	n.Chal("info", fmt.Sprintf("Submit proof suc: %v", txhash))

	err = n.Put([]byte(Cach_AggrProof_Reported), []byte(fmt.Sprintf("%v", chal.Start)))
	if err != nil {
		n.Chal("err", fmt.Sprintf("Put Cach_AggrProof_Reported [%d] err: %v", chal.Start, err))
	}

	time.Sleep(pattern.BlockInterval)

	err = n.transferProof()
	if err != nil {
		n.Chal("err", fmt.Sprintf("Put Cach_AggrProof_Reported [%d] err: %v", chal.Start, err))
		continue
	}
	return nil
}

func (n *Node) transferProof() error {
	chalshort, err := n.QueryChallengeSt()
	if err != nil {
		return errors.Wrapf(err, "[QueryChallengeSt]")
	}

	n.Chal("info", fmt.Sprintf("[QueryChallengeSt] %d", chalshort.NetSnapshot.Start))
	idleProofFileHash, err := sutils.CalcPathSHA256Bytes(n.GetDirs().IproofFile)
	if err != nil {
		return errors.Wrapf(err, "[CalcPathSHA256Bytes]")
	}
	serviceProofFileHash, err := sutils.CalcPathSHA256Bytes(n.GetDirs().SproofFile)
	if err != nil {
		return errors.Wrapf(err, "[CalcPathSHA256Bytes]")
	}
	peerid, code, err := n.proofAssignedInfo(idleProofFileHash, serviceProofFileHash, chalshort.NetSnapshot.Random_index_list, chalshort.NetSnapshot.Random)
	if err != nil || code != 0 {
		return errors.Wrapf(err, "[proofAsigmentInfo]")
	}
	n.Chal("info", fmt.Sprintf("proofAsigmentInfo suc: %d", chalshort.NetSnapshot.Start))
	err = n.Put([]byte(Cach_AggrProof_Transfered), []byte(fmt.Sprintf("%s_%v", peerid, chalshort.NetSnapshot.Start)))
	if err != nil {
		return errors.Wrapf(err, "[PutCache]")
	}
	return nil
}

func (n *Node) proofAssignedInfo(ihash, shash []byte, randomIndexList []uint32, random [][]byte) (string, uint32, error) {
	var err error
	var count uint8
	var code uint32
	var teeAsigned []byte
	var peerid peer.ID
	peerid, teeAsigned, err = n.queryProofAssignedTee()
	if err != nil {
		n.Chal("err", fmt.Sprintf("[queryProofAssignedTee] %v", err))
		return "", code, fmt.Errorf("proof not assigned")
	}

	if teeAsigned == nil {
		n.Chal("err", "proof not assigned")
		return "", code, fmt.Errorf("proof not assigned")
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
		return "", code, err
	}
	count = 0
	n.Chal("info", fmt.Sprintf("Send aggr proof request to: %s", peerid.Pretty()))
	for count < 5 {
		code, err = n.AggrProofReq(peerid, ihash, shash, qslice, n.GetStakingPublickey(), sign)
		if err != nil || code != 0 {
			count++
			n.Chal("err", fmt.Sprintf("AggrProofReq err: %v, code: %d", err, code))
			time.Sleep(pattern.BlockInterval)
			continue
		}
		n.Chal("info", fmt.Sprintf("Aggr proof response suc: %s", peerid.Pretty()))
		break
	}

	if count >= 5 {
		n.Chal("err", fmt.Sprintf("AggrProofReq err: %v", err))
		return "", code, err
	}

	idleProofFileHashs, _ := sutils.CalcPathSHA256(n.GetDirs().IproofFile)
	serviceProofFileHashs, _ := sutils.CalcPathSHA256(n.GetDirs().SproofFile)

	count = 0
	n.Chal("info", fmt.Sprintf("Send aggr proof idle file request to: %s", peerid.Pretty()))
	for count < 5 {
		code, err = n.FileReq(peerid, idleProofFileHashs, pb.FileType_IdleMu, n.GetDirs().IproofFile)
		if err != nil || code != 0 {
			count++
			n.Chal("err", fmt.Sprintf("FileType_IdleMu FileReq err: %v, code: %d", err, code))
			time.Sleep(pattern.BlockInterval)
			continue
		}
		n.Chal("info", fmt.Sprintf("Aggr proof idle file response suc: %s", peerid.Pretty()))
		break
	}
	if count >= 5 {
		n.Chal("err", fmt.Sprintf("FileReq FileType_IdleMu err: %v", err))
		return "", code, err
	}

	count = 0
	n.Chal("info", fmt.Sprintf("Send aggr proof service file request to: %s", peerid.Pretty()))
	for count < 5 {
		code, err = n.FileReq(peerid, serviceProofFileHashs, pb.FileType_CustomMu, n.GetDirs().SproofFile)
		if err != nil || code != 0 {
			count++
			n.Chal("err", fmt.Sprintf("FileType_CustomMu FileReq err: %v, code: %d", err, code))
			time.Sleep(pattern.BlockInterval)
			continue
		}
		n.Chal("info", fmt.Sprintf("Aggr proof service file response suc: %s", peerid.Pretty()))
		break
	}
	return peerid.Pretty(), code, err
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

func (n *Node) queryProofAssignedTee() (peer.ID, []byte, error) {
	var err error
	var teelist []pattern.TeeWorkerInfo
	var proof []pattern.ProofAssignmentInfo
	var teeAsigned []byte
	var peerid peer.ID
	teelist, err = n.QueryTeeInfoList()
	if err != nil {
		return "", nil, errors.Wrapf(err, "[QueryTeeInfoList]")
	}

	for _, v := range teelist {
		proof, err = n.QueryTeeAssignedProof(v.ControllerAccount[:])
		if err != nil {
			continue
		}

		for i := 0; i < len(proof); i++ {
			if sutils.CompareSlice(proof[i].SnapShot.Miner[:], n.GetStakingPublickey()) {
				peerid, err = peer.Decode(base58.Encode([]byte(string(v.PeerId[:]))))
				if err != nil {
					return "", nil, errors.Wrapf(err, "[peer.Decode]")
				}
				teeAsigned = v.ControllerAccount[:]
				n.Chal("info", fmt.Sprintf("proof assigned tee: %s", peerid.Pretty()))
				return peerid, teeAsigned, nil
			}
		}
	}
	return peerid, teeAsigned, err
}
