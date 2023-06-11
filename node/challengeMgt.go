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
	var b []byte
	var chalStart int
	var chal pattern.ChallengeInfo
	var netinfo pattern.ChallengeSnapShot
	var qslice []proof.QElement

	n.Chal("info", ">>>>> Start challengeMgt task")

	for {
		pubkey, err := n.QueryTeePodr2Puk()
		if err != nil {
			configs.Err(fmt.Sprintf("[QueryTeePodr2Puk] %v", err))
			time.Sleep(pattern.BlockInterval)
			continue
		}
		err = n.key.SetPublickey(pubkey)
		if err != nil {
			configs.Err(fmt.Sprintf("[SetPublickey] %v", err))
			time.Sleep(pattern.BlockInterval)
			continue
		}
		configs.Ok("Initialize key successfully")
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
				chalStart, err = strconv.Atoi(string(b))
				if err != nil {
					n.Delete([]byte(Cach_AggrProof_Transfered))
					err = n.transferProof()
					if err != nil {
						n.Chal("err", err.Error())
					}
					continue
				}

				if chalStart != int(netinfo.NetSnapshot.Start) {
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
			err = n.Put([]byte(Cach_AggrProof_Transfered), []byte(fmt.Sprintf("%v", chal.Start)))
			if err != nil {
				n.Chal("err", fmt.Sprintf("Put Cach_AggrProof_Transfered [%d] err: %v", chal.Start, err))
			}
		}
		time.Sleep(pattern.BlockInterval)
	}
}

func (n *Node) transferProof() error {
	chalshort, err := n.QueryChallengeSt()
	if err != nil {
		return errors.Wrapf(err, "[QueryChallengeSt]")
	}

	n.Chal("info", fmt.Sprintf("[QueryChallengeSt] %d", chalshort.NetSnapshot.Start))
	idleProofFileHash, err := utils.CalcPathSHA256Bytes(n.GetDirs().IproofFile)
	if err != nil {
		return errors.Wrapf(err, "[CalcPathSHA256Bytes]")
	}
	serviceProofFileHash, err := utils.CalcPathSHA256Bytes(n.GetDirs().SproofFile)
	if err != nil {
		return errors.Wrapf(err, "[CalcPathSHA256Bytes]")
	}
	err = n.proofAsigmentInfo(idleProofFileHash, serviceProofFileHash, chalshort.NetSnapshot.Random_index_list, chalshort.NetSnapshot.Random)
	if err != nil {
		return errors.Wrapf(err, "[proofAsigmentInfo]")
	}
	n.Chal("info", fmt.Sprintf("proofAsigmentInfo suc: %d", chalshort.NetSnapshot.Start))
	err = n.Put([]byte(Cach_AggrProof_Transfered), []byte(fmt.Sprintf("%v", chalshort.NetSnapshot.Start)))
	if err != nil {
		return errors.Wrapf(err, "[PutCache]")
	}
	return nil
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
				n.Chal("info", fmt.Sprintf("proof assigned tee: %s", peerid.Pretty()))
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

	n.Chal("info", fmt.Sprintf("Send aggr proof request to: %s", peerid.Pretty()))
	for {
		code, err := n.AggrProofReq(peerid, ihash, shash, qslice, n.GetStakingPublickey(), sign)
		if err != nil || code != 0 {
			n.Chal("err", fmt.Sprintf("AggrProofReq err: %v, code: %d", err, code))
			time.Sleep(pattern.BlockInterval * 5)
			continue
		}
		n.Chal("info", fmt.Sprintf("Aggr proof response suc: %s", peerid.Pretty()))
		break
	}

	idleProofFileHashs, _ := utils.CalcPathSHA256(n.GetDirs().IproofFile)
	serviceProofFileHashs, _ := utils.CalcPathSHA256(n.GetDirs().SproofFile)

	n.Chal("info", fmt.Sprintf("Send aggr proof idle file request to: %s", peerid.Pretty()))
	for {
		code, err := n.FileReq(peerid, idleProofFileHashs, pb.FileType_IdleMu, n.GetDirs().IproofFile)
		if err != nil || code != 0 {
			n.Chal("err", fmt.Sprintf("FileType_IdleMu FileReq err: %v, code: %d", err, code))
			time.Sleep(pattern.BlockInterval * 5)
			continue
		}
		n.Chal("info", fmt.Sprintf("Aggr proof idle file response suc: %s", peerid.Pretty()))
		break
	}

	n.Chal("info", fmt.Sprintf("Send aggr proof service file request to: %s", peerid.Pretty()))
	for {
		code, err := n.FileReq(peerid, serviceProofFileHashs, pb.FileType_CustomMu, n.GetDirs().SproofFile)
		if err != nil || code != 0 {
			n.Chal("err", fmt.Sprintf("FileType_CustomMu FileReq err: %v, code: %d", err, code))
			time.Sleep(pattern.BlockInterval * 5)
			continue
		}
		n.Chal("info", fmt.Sprintf("Aggr proof service file response suc: %s", peerid.Pretty()))
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
	fmt.Println("> > > idleRoothash:", idleRoothashs)
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
		fmt.Println("> > > idleTagPath:", idleTagPath)
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

		proveResponseCh := n.key.GenProof(qslice, nil, ptag, matrix)
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

	sigma := n.key.AggrGenProof(qslice, ptags)

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

			proveResponseCh := n.key.GenProof(qslice, nil, ptag, matrix)
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

	sigma := n.key.AggrGenProof(qslice, ptags)
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
