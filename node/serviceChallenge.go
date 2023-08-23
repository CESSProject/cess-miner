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
	"time"

	"github.com/CESSProject/cess-bucket/pkg/proof"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
)

type serviceProofInfo struct {
	Names                 []string `json:"names"`
	Us                    []string `json:"us"`
	Mus                   []string `json:"mus"`
	ServiceBloomFilter    []uint64 `json:"serviceBloomFilter"`
	TeePeerId             []byte   `json:"teePeerId"`
	Signature             []byte   `json:"signature"`
	AllocatedTeeAccountId []byte   `json:"allocatedTeeAccountId"`
	AllocatedTeeAccount   string   `json:"allocatedTeeAccount"`
	Sigma                 string   `json:"sigma"`
	Start                 uint32   `json:"start"`
	ServiceResult         bool     `json:"serviceResult"`
}

func (n *Node) serviceChallenge(
	ch chan<- bool,
	latestBlock,
	challExpiration uint32,
	challenge pattern.ChallengeInfo_V2,
	minerChalInfo pattern.MinerSnapShot_V2,
) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	var haveChallenge = true

	if challExpiration <= latestBlock {
		haveChallenge = false
	}

	if !haveChallenge {
		if !minerChalInfo.ServiceSubmitted {
			n.Schal("err", "Proof of service files not submitted")
			return
		}
	}

	var found bool
	var serviceProofRecord serviceProofInfo
	err := n.checkServiceProofRecord(challenge, minerChalInfo)
	if err == nil {
		return
	}

	n.Schal("info", fmt.Sprintf("Service file chain challenge: %v", challenge.NetSnapShot.Start))

	var qslice = make([]proof.QElement, len(challenge.NetSnapShot.RandomIndexList))
	for k, v := range challenge.NetSnapShot.RandomIndexList {
		qslice[k].I = int64(v)
		var b = make([]byte, len(challenge.NetSnapShot.RandomList[k]))
		for i := 0; i < len(challenge.NetSnapShot.RandomList[k]); i++ {
			b[i] = byte(challenge.NetSnapShot.RandomList[k][i])
		}
		qslice[k].V = new(big.Int).SetBytes(b).String()
	}

	err = n.saveRandom(challenge)
	if err != nil {
		n.Schal("err", fmt.Sprintf("Save service file challenge random err: %v", err))
	}

	serviceProofRecord = serviceProofInfo{}
	serviceProofRecord.Start = uint32(challenge.NetSnapShot.Start)
	serviceProofRecord.Names, serviceProofRecord.Us, serviceProofRecord.Mus, serviceProofRecord.Sigma, err = n.calcSigma(challenge)
	if err != nil {
		n.Schal("err", fmt.Sprintf("[calcSigma] %v", err))
		return
	}

	n.saveServiceProofRecord(serviceProofRecord)

	var serviceProof = make([]types.U8, len(serviceProofRecord.Sigma))
	for i := 0; i < len(serviceProofRecord.Sigma); i++ {
		serviceProof[i] = types.U8(serviceProofRecord.Sigma[i])
	}

	txhash, err := n.SubmitServiceProof(serviceProof)
	if err != nil {
		n.Schal("err", fmt.Sprintf("[SubmitServiceProof] %v", err))
		return
	}
	n.Schal("info", fmt.Sprintf("submit service aggr proof suc: %s", txhash))

	time.Sleep(pattern.BlockInterval * 2)

	found = false
	teeAccounts := n.GetAllTeeWorkAccount()
	for _, v := range teeAccounts {
		if found {
			break
		}

		publickey, _ := sutils.ParsingPublickey(v)
		serviceProofInfos, err := n.QueryUnverifiedServiceProof(publickey)
		if err != nil {
			continue
		}

		for i := 0; i < len(serviceProofInfos); i++ {
			if sutils.CompareSlice(serviceProofInfos[i].MinerSnapShot.Miner[:], n.GetSignatureAccPulickey()) {
				serviceProofRecord.AllocatedTeeAccount = v
				serviceProofRecord.AllocatedTeeAccountId = publickey
				found = true
				break
			}
		}
	}

	if !found {
		n.Schal("err", "No tee found to verify service files prove")
		return
	}

	teePeerIdPubkey, _ := n.GetTeeWork(serviceProofRecord.AllocatedTeeAccount)

	teeAddrInfo, ok := n.GetPeer(base58.Encode(teePeerIdPubkey))
	if !ok {
		n.Schal("err", fmt.Sprintf("Not discovered tee peer: %s", base58.Encode(teePeerIdPubkey)))
		return
	}

	err = n.Connect(n.GetCtxQueryFromCtxCancel(), teeAddrInfo)
	if err != nil {
		n.Schal("err", fmt.Sprintf("Connect tee peer err: %v", err))
	}

	serviceProofRecord.ServiceBloomFilter, serviceProofRecord.TeePeerId, serviceProofRecord.Signature, serviceProofRecord.ServiceResult, err = n.batchVerify(challenge, teeAddrInfo, serviceProofRecord)
	if err != nil {
		n.Schal("err", fmt.Sprintf("[batchVerify] %v", err))
		return
	}

	n.Schal("info", fmt.Sprintf("Batch verification results of service files: %v", serviceProofRecord.ServiceResult))

	var signature pattern.TeeSignature
	if len(pattern.TeeSignature{}) != len(serviceProofRecord.Signature) {
		n.Schal("err", "invalid batchVerify.Signature")
		return
	}
	for i := 0; i < len(serviceProofRecord.Signature); i++ {
		signature[i] = types.U8(serviceProofRecord.Signature[i])
	}

	var bloomFilter pattern.BloomFilter
	if len(pattern.BloomFilter{}) != len(serviceProofRecord.ServiceBloomFilter) {
		n.Schal("err", "invalid batchVerify.ServiceBloomFilter")
		return
	}
	for i := 0; i < len(serviceProofRecord.ServiceBloomFilter); i++ {
		bloomFilter[i] = types.U64(serviceProofRecord.ServiceBloomFilter[i])
	}

	n.saveServiceProofRecord(serviceProofRecord)

	txhash, err = n.SubmitServiceProofResult(types.Bool(serviceProofRecord.ServiceResult), signature, bloomFilter, serviceProofRecord.AllocatedTeeAccountId)
	if err != nil {
		n.Schal("err", fmt.Sprintf("[SubmitServiceProofResult] hash: %s, err: %v", txhash, err))
		return
	}
	n.Schal("info", fmt.Sprintf("submit service aggr proof result suc: %s", txhash))
	return
}

func (n *Node) poisServiceChallengeResult(ch chan<- bool, latestBlock, challVerifyExpiration uint32, serviceChallTeeAcc string, challenge pattern.ChallengeInfo_V2, minerChalInfo pattern.MinerSnapShot_V2) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	if challVerifyExpiration <= latestBlock {
		return
	}

	n.Schal("info", fmt.Sprintf("Service file chain challenge: %v", challenge.NetSnapShot.Start))

	var serviceProofRecord serviceProofInfo
	buf, err := os.ReadFile(filepath.Join(n.Workspace(), "serviceproof"))
	if err != nil {
		n.Schal("err", fmt.Sprintf("[ReadFile(serviceproof)] %v", err))
		return
	}

	err = json.Unmarshal(buf, &serviceProofRecord)
	if err != nil {
		os.Remove(filepath.Join(n.Workspace(), "serviceproof"))
		n.Schal("err", fmt.Sprintf("[Unmarshal(serviceproof)] %v", err))
		return
	}

	n.Schal("info", fmt.Sprintf("local service proof file challenge: %v", serviceProofRecord.Start))
	if serviceProofRecord.Start != uint32(challenge.NetSnapShot.Start) {
		os.Remove(filepath.Join(n.Workspace(), "serviceproof"))
		return
	}

	allocatedTeeAccountId, err := sutils.ParsingPublickey(serviceChallTeeAcc)
	if err != nil {
		n.Schal("err", fmt.Sprintf("[ParsingPublickey(%s)] %v", serviceChallTeeAcc, err))
		return
	}

	if serviceProofRecord.ServiceBloomFilter != nil &&
		serviceProofRecord.TeePeerId != nil &&
		serviceProofRecord.Signature != nil {
		var signature pattern.TeeSignature
		if len(pattern.TeeSignature{}) != len(serviceProofRecord.Signature) {
			n.Schal("err", "invalid batchVerify.Signature")
			return
		}
		for i := 0; i < len(serviceProofRecord.Signature); i++ {
			signature[i] = types.U8(serviceProofRecord.Signature[i])
		}

		var bloomFilter pattern.BloomFilter
		if len(pattern.BloomFilter{}) != len(serviceProofRecord.ServiceBloomFilter) {
			n.Schal("err", "invalid batchVerify.ServiceBloomFilter")
			return
		}
		for i := 0; i < len(serviceProofRecord.ServiceBloomFilter); i++ {
			bloomFilter[i] = types.U64(serviceProofRecord.ServiceBloomFilter[i])
		}

		txhash, err := n.SubmitServiceProofResult(types.Bool(serviceProofRecord.ServiceResult), signature, bloomFilter, allocatedTeeAccountId)
		if err != nil {
			n.Schal("err", fmt.Sprintf("[SubmitServiceProofResult] hash: %s, err: %v", txhash, err))
			return
		}
		n.Schal("info", fmt.Sprintf("submit service aggr proof result suc: %s", txhash))
		return
	}

	teePeerIdPubkey, _ := n.GetTeeWork(serviceChallTeeAcc)

	teeAddrInfo, ok := n.GetPeer(base58.Encode(teePeerIdPubkey))
	if !ok {
		n.Schal("err", fmt.Sprintf("Not found tee peer: %s", base58.Encode(teePeerIdPubkey)))
		return
	}
	err = n.Connect(n.GetCtxQueryFromCtxCancel(), teeAddrInfo)
	if err != nil {
		n.Schal("err", fmt.Sprintf("Connect tee peer err: %v", err))
	}
	peeridSign, err := n.Sign(n.GetPeerPublickey())
	if err != nil {
		n.Schal("err", fmt.Sprintf("[Sign] %v", err))
		return
	}

	var randomIndexList = make([]uint32, len(challenge.NetSnapShot.RandomIndexList))
	for i := 0; i < len(challenge.NetSnapShot.RandomIndexList); i++ {
		randomIndexList[i] = uint32(challenge.NetSnapShot.RandomIndexList[i])
	}
	var randomList = make([][]byte, len(challenge.NetSnapShot.RandomList))
	for i := 0; i < len(challenge.NetSnapShot.RandomList); i++ {
		randomList[i] = make([]byte, len(challenge.NetSnapShot.RandomList[i]))
		for j := 0; j < len(challenge.NetSnapShot.RandomList[i]); j++ {
			randomList[i][j] = byte(challenge.NetSnapShot.RandomList[i][j])
		}
	}

	var qslice_pb = &pb.RequestBatchVerify_Qslice{
		RandomIndexList: randomIndexList,
		RandomList:      randomList,
	}
	n.Schal("info", fmt.Sprintf("req tee batch verify: %s", base58.Encode(teePeerIdPubkey)))
	batchVerify, err := n.PoisServiceRequestBatchVerifyP2P(
		teeAddrInfo.ID,
		serviceProofRecord.Names,
		serviceProofRecord.Us,
		serviceProofRecord.Mus,
		serviceProofRecord.Sigma,
		n.GetPeerPublickey(),
		n.GetSignatureAccPulickey(),
		peeridSign,
		qslice_pb,
		time.Duration(time.Minute*10),
	)
	if err != nil {
		n.Schal("err", fmt.Sprintf("[PoisServiceRequestBatchVerifyP2P] %v", err))
		return
	}
	serviceProofRecord.ServiceResult = batchVerify.BatchVerifyResult
	serviceProofRecord.ServiceBloomFilter = batchVerify.ServiceBloomFilter
	serviceProofRecord.TeePeerId = batchVerify.TeePeerId
	serviceProofRecord.Signature = batchVerify.Signature
	buf, err = json.Marshal(&serviceProofRecord)
	if err != nil {
		n.Schal("err", err.Error())
	}
	err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), "serviceproof"))
	if err != nil {
		n.Schal("err", err.Error())
	}

	var signature pattern.TeeSignature
	if len(pattern.TeeSignature{}) != len(batchVerify.Signature) {
		n.Schal("err", "invalid batchVerify.Signature")
		return
	}
	for i := 0; i < len(batchVerify.Signature); i++ {
		signature[i] = types.U8(batchVerify.Signature[i])
	}

	var bloomFilter pattern.BloomFilter
	if len(pattern.BloomFilter{}) != len(batchVerify.ServiceBloomFilter) {
		n.Schal("err", "invalid batchVerify.ServiceBloomFilter")
		return
	}
	for i := 0; i < len(batchVerify.ServiceBloomFilter); i++ {
		bloomFilter[i] = types.U64(batchVerify.ServiceBloomFilter[i])
	}

	txhash, err := n.SubmitServiceProofResult(types.Bool(batchVerify.BatchVerifyResult), signature, bloomFilter, allocatedTeeAccountId)
	if err != nil {
		n.Schal("err", fmt.Sprintf("[SubmitServiceProofResult] hash: %s, err: %v", txhash, err))
		return
	}
	n.Schal("info", fmt.Sprintf("submit service aggr proof result suc: %s", txhash))
	return
}

// save challenge random number
func (n *Node) saveRandom(challenge pattern.ChallengeInfo_V2) error {
	randfilePath := filepath.Join(n.GetDirs().ProofDir, fmt.Sprintf("random.%d", challenge.NetSnapShot.Start))
	fstat, err := os.Stat(randfilePath)
	if err == nil && fstat.Size() > 0 {
		return nil
	}
	var rd RandomList
	rd.Index = make([]uint32, len(challenge.NetSnapShot.RandomIndexList))
	rd.Random = make([][]byte, len(challenge.NetSnapShot.RandomIndexList))
	for i := 0; i < len(challenge.NetSnapShot.RandomIndexList); i++ {
		rd.Index[i] = uint32(challenge.NetSnapShot.RandomIndexList[i])
		rd.Random[i] = make([]byte, len(challenge.NetSnapShot.RandomList[i]))
		for j := 0; j < len(challenge.NetSnapShot.RandomList[i]); j++ {
			rd.Random[i][j] = byte(challenge.NetSnapShot.RandomList[i][j])
		}
	}
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

// calc sigma
func (n *Node) calcSigma(challenge pattern.ChallengeInfo_V2) ([]string, []string, []string, string, error) {
	var sigma string
	var proveResponse proof.GenProofResponse
	var names = make([]string, 0)
	var us = make([]string, 0)
	var mus = make([]string, 0)
	var qslice = make([]proof.QElement, len(challenge.NetSnapShot.RandomIndexList))
	for k, v := range challenge.NetSnapShot.RandomIndexList {
		qslice[k].I = int64(v)
		var b = make([]byte, len(challenge.NetSnapShot.RandomList[k]))
		for i := 0; i < len(challenge.NetSnapShot.RandomList[k]); i++ {
			b[i] = byte(challenge.NetSnapShot.RandomList[k][i])
		}
		qslice[k].V = new(big.Int).SetBytes(b).String()
	}

	serviceRoothashDir, err := utils.Dirs(n.GetDirs().FileDir)
	if err != nil {
		n.Schal("err", fmt.Sprintf("[Dirs] %v", err))
		return names, us, mus, sigma, err
	}

	timeout := time.NewTicker(time.Duration(time.Minute))
	defer timeout.Stop()

	for i := int(0); i < len(serviceRoothashDir); i++ {
		roothash := filepath.Base(serviceRoothashDir[i])
		fmeta, err := n.QueryFileMetadataByBlock(roothash, uint64(challenge.NetSnapShot.Start))
		if err != nil {
			if err.Error() != pattern.ERR_Empty {
				n.Schal("err", fmt.Sprintf("[QueryFileMetadata(%s)] %v", roothash, err.Error()))
				return names, us, mus, sigma, err
			}
			os.RemoveAll(serviceRoothashDir[i])
			continue
		}

		for _, segment := range fmeta.SegmentList {
			for _, fragment := range segment.FragmentList {
				if !sutils.CompareSlice(fragment.Miner[:], n.GetSignatureAccPulickey()) {
					os.Remove(filepath.Join(serviceRoothashDir[i], string(fragment.Hash[:])))
				}
			}
		}

		files, err := utils.DirFiles(serviceRoothashDir[i], 0)
		if err != nil {
			n.Schal("err", err.Error())
			continue
		}

		for j := 0; j < len(files); j++ {
			serviceTagPath := filepath.Join(n.GetDirs().ServiceTagDir, filepath.Base(files[j])+".tag")
			buf, err := os.ReadFile(serviceTagPath)
			if err != nil {
				n.Schal("err", fmt.Sprintf("Servicetag not found: %v", serviceTagPath))
				continue
			}
			var tag pb.Tag
			err = json.Unmarshal(buf, &tag)
			if err != nil {
				n.Schal("err", fmt.Sprintf("Unmarshal %v err: %v", serviceTagPath, err))
				continue
			}
			matrix, _, err := proof.SplitByN(files[j], int64(len(tag.T.Phi)))
			if err != nil {
				n.Schal("err", fmt.Sprintf("SplitByN %v err: %v", serviceTagPath, err))
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
				n.Schal("err", fmt.Sprintf("GenProof  err: %d", proveResponse.StatueMsg.StatusCode))
				continue
			}

			sigmaTemp, ok := n.key.AggrAppendProof(sigma, qslice, tag.T.Phi)
			if !ok {
				continue
			}
			sigma = sigmaTemp
			names = append(names, tag.T.Name)
			us = append(us, tag.T.U)
			mus = append(mus, proveResponse.MU)
		}
	}
	return names, us, mus, sigma, nil
}

func (n *Node) checkServiceProofRecord(challenge pattern.ChallengeInfo_V2, minerChalInfo pattern.MinerSnapShot_V2) error {
	var found bool
	var serviceProofRecord serviceProofInfo
	buf, err := os.ReadFile(filepath.Join(n.Workspace(), "serviceproof"))
	if err != nil {
		return err
	}

	err = json.Unmarshal(buf, &serviceProofRecord)
	if err != nil {
		return err
	}

	if serviceProofRecord.Start != uint32(challenge.NetSnapShot.Start) {
		os.Remove(filepath.Join(n.Workspace(), "serviceproof"))
		return errors.New("Local service file challenge record is outdated")
	}

	n.Schal("info", fmt.Sprintf("local service proof file challenge: %v", serviceProofRecord.Start))

	if !minerChalInfo.ServiceSubmitted {
		if serviceProofRecord.Names == nil ||
			serviceProofRecord.Us == nil ||
			serviceProofRecord.Mus == nil {
			serviceProofRecord.Names, serviceProofRecord.Us, serviceProofRecord.Mus, serviceProofRecord.Sigma, err = n.calcSigma(challenge)
			if err != nil {
				n.Schal("err", fmt.Sprintf("[calcSigma] %v", err))
				return nil
			}
		}
		n.saveServiceProofRecord(serviceProofRecord)

		var serviceProve = make([]types.U8, len(serviceProofRecord.Sigma))
		for i := 0; i < len(serviceProofRecord.Sigma); i++ {
			serviceProve[i] = types.U8(serviceProofRecord.Sigma[i])
		}
		_, err = n.SubmitServiceProof(serviceProve)
		if err != nil {
			n.Schal("err", fmt.Sprintf("[SubmitServiceProof] %v", err))
			return nil
		}
		time.Sleep(pattern.BlockInterval * 2)
	}

	found = false
	teeAccounts := n.GetAllTeeWorkAccount()
	for _, v := range teeAccounts {
		if found {
			break
		}
		publickey, _ := sutils.ParsingPublickey(v)
		serviceProofInfos, err := n.QueryUnverifiedServiceProof(publickey)
		if err != nil {
			continue
		}

		for i := 0; i < len(serviceProofInfos); i++ {
			if sutils.CompareSlice(serviceProofInfos[i].MinerSnapShot.Miner[:], n.GetSignatureAccPulickey()) {
				serviceProofRecord.AllocatedTeeAccount = v
				serviceProofRecord.AllocatedTeeAccountId = publickey
				found = true
				break
			}
		}
	}

	if !found {
		n.Schal("err", "No tee found to verify service files prove")
		return nil
	}

	for {
		if serviceProofRecord.ServiceBloomFilter != nil &&
			serviceProofRecord.TeePeerId != nil &&
			serviceProofRecord.Signature != nil {
			var signature pattern.TeeSignature
			if len(pattern.TeeSignature{}) != len(serviceProofRecord.Signature) {
				n.Schal("err", "invalid batchVerify.Signature")
				break
			}
			for i := 0; i < len(serviceProofRecord.Signature); i++ {
				signature[i] = types.U8(serviceProofRecord.Signature[i])
			}

			var bloomFilter pattern.BloomFilter
			if len(pattern.BloomFilter{}) != len(serviceProofRecord.ServiceBloomFilter) {
				n.Schal("err", "invalid batchVerify.ServiceBloomFilter")
				break
			}
			for i := 0; i < len(serviceProofRecord.ServiceBloomFilter); i++ {
				bloomFilter[i] = types.U64(serviceProofRecord.ServiceBloomFilter[i])
			}

			txhash, err := n.SubmitServiceProofResult(
				types.Bool(serviceProofRecord.ServiceResult),
				signature,
				bloomFilter,
				serviceProofRecord.AllocatedTeeAccountId,
			)
			if err != nil {
				n.Schal("err", fmt.Sprintf("[SubmitServiceProofResult] hash: %s, err: %v", txhash, err))
				break
			}
			n.Schal("info", fmt.Sprintf("submit service aggr proof result suc: %s", txhash))
			return nil
		}
		break
	}

	teePeerIdPubkey, _ := n.GetTeeWork(serviceProofRecord.AllocatedTeeAccount)

	teeAddrInfo, ok := n.GetPeer(base58.Encode(teePeerIdPubkey))
	if !ok {
		n.Schal("err", fmt.Sprintf("Not discovered tee peer: %s", base58.Encode(teePeerIdPubkey)))
		return nil
	}
	err = n.Connect(n.GetCtxQueryFromCtxCancel(), teeAddrInfo)
	if err != nil {
		n.Schal("err", fmt.Sprintf("Connect tee peer err: %v", err))
	}
	serviceProofRecord.ServiceBloomFilter, serviceProofRecord.TeePeerId, serviceProofRecord.Signature, serviceProofRecord.ServiceResult, err = n.batchVerify(challenge, teeAddrInfo, serviceProofRecord)
	if err != nil {
		return nil
	}
	n.Schal("info", fmt.Sprintf("Batch verification results of service files: %v", serviceProofRecord.ServiceResult))

	var signature pattern.TeeSignature
	if len(pattern.TeeSignature{}) != len(serviceProofRecord.Signature) {
		n.Schal("err", "invalid batchVerify.Signature")
		return nil
	}
	for i := 0; i < len(serviceProofRecord.Signature); i++ {
		signature[i] = types.U8(serviceProofRecord.Signature[i])
	}

	var bloomFilter pattern.BloomFilter
	if len(pattern.BloomFilter{}) != len(serviceProofRecord.ServiceBloomFilter) {
		n.Schal("err", "invalid batchVerify.ServiceBloomFilter")
		return nil
	}
	for i := 0; i < len(serviceProofRecord.ServiceBloomFilter); i++ {
		bloomFilter[i] = types.U64(serviceProofRecord.ServiceBloomFilter[i])
	}

	n.saveServiceProofRecord(serviceProofRecord)

	txhash, err := n.SubmitServiceProofResult(types.Bool(serviceProofRecord.ServiceResult), signature, bloomFilter, serviceProofRecord.AllocatedTeeAccountId)
	if err != nil {
		n.Schal("err", fmt.Sprintf("[SubmitServiceProofResult] hash: %s, err: %v", txhash, err))
		return nil
	}
	n.Schal("info", fmt.Sprintf("submit service aggr proof result suc: %s", txhash))
	return nil
}

func (n *Node) saveServiceProofRecord(serviceProofRecord serviceProofInfo) {
	buf, err := json.Marshal(&serviceProofRecord)
	if err == nil {
		err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), "serviceproof"))
		if err != nil {
			n.Schal("err", err.Error())
		}
	}
}

func (n *Node) batchVerify(challenge pattern.ChallengeInfo_V2, teeAddrInfo peer.AddrInfo, serviceProofRecord serviceProofInfo) ([]uint64, []byte, []byte, bool, error) {
	var randomIndexList = make([]uint32, len(challenge.NetSnapShot.RandomIndexList))
	for i := 0; i < len(challenge.NetSnapShot.RandomIndexList); i++ {
		randomIndexList[i] = uint32(challenge.NetSnapShot.RandomIndexList[i])
	}
	var randomList = make([][]byte, len(challenge.NetSnapShot.RandomList))
	for i := 0; i < len(challenge.NetSnapShot.RandomList); i++ {
		randomList[i] = make([]byte, len(challenge.NetSnapShot.RandomList[i]))
		for j := 0; j < len(challenge.NetSnapShot.RandomList[i]); j++ {
			randomList[i][j] = byte(challenge.NetSnapShot.RandomList[i][j])
		}
	}

	var qslice_pb = &pb.RequestBatchVerify_Qslice{
		RandomIndexList: randomIndexList,
		RandomList:      randomList,
	}

	peeridSign, err := n.Sign(n.GetPeerPublickey())
	if err != nil {
		n.Schal("err", fmt.Sprintf("[Sign] %v", err))
		return nil, nil, nil, false, err
	}

	n.Schal("info", fmt.Sprintf("req tee batch verify: %s", teeAddrInfo.ID.Pretty()))
	batchVerify, err := n.PoisServiceRequestBatchVerifyP2P(
		teeAddrInfo.ID,
		serviceProofRecord.Names,
		serviceProofRecord.Us,
		serviceProofRecord.Mus,
		serviceProofRecord.Sigma,
		n.GetPeerPublickey(),
		n.GetSignatureAccPulickey(),
		peeridSign,
		qslice_pb,
		time.Duration(time.Minute*10),
	)
	if err != nil {
		n.Schal("err", fmt.Sprintf("[PoisServiceRequestBatchVerifyP2P] %v", err))
		return nil, nil, nil, false, err
	}
	return batchVerify.ServiceBloomFilter, batchVerify.TeePeerId, batchVerify.Signature, batchVerify.BatchVerifyResult, nil
}
