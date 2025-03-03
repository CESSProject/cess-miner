/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AstaFrode/go-substrate-rpc-client/v4/types"
	"github.com/CESSProject/cess-go-sdk/chain"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/pkg/cache"
	"github.com/CESSProject/cess-miner/pkg/com"
	"github.com/CESSProject/cess-miner/pkg/com/pb"
	"github.com/CESSProject/cess-miner/pkg/utils"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const maxNumberOfSingleVerification = 5000

type challengedFile struct {
	Fid       string
	Fragments []string
}

func (n *Node) serviceChallenge(ch chan<- bool, rndIndex []types.U32, rnd []chain.Random, chlgStart, slip, verifySlip uint32) {
	defer func() {
		ch <- true
		n.SetServiceChallenging(false)
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	var err error
	blockhash := ""
	verifySuc := false
	var latestBlock *types.SignedBlock
	serviceProofRecord, err := n.LoadServiceProve()
	if err == nil {
		if serviceProofRecord.Start != chlgStart {
			os.Remove(n.GetServiceProve())
			n.Del("info", n.GetServiceProve())
			n.Delete([]byte(fmt.Sprintf("%s%d", cache.Prefix_service_chall_proof, serviceProofRecord.Start)))
			n.Delete([]byte(fmt.Sprintf("%s%d", cache.Prefix_service_chall_result, serviceProofRecord.Start)))
		} else {
			_, err = n.Cache.Get([]byte(fmt.Sprintf("%s%d", cache.Prefix_service_chall_proof, chlgStart)))
			if err != nil {
				blockhash, err = n.submitServiceProof(serviceProofRecord.Proof, slip)
				if err != nil {
					n.Schal("err", err.Error())
				}
				if blockhash != "" {
					n.Cache.Put([]byte(fmt.Sprintf("%s%d", cache.Prefix_service_chall_proof, chlgStart)), []byte("true"))
					serviceProofRecord.CanSubmitProof = false
					n.SaveServiceProve(serviceProofRecord)
				}
			}
			_, err = n.Cache.Get([]byte(fmt.Sprintf("%s%d", cache.Prefix_service_chall_result, chlgStart)))
			if err != nil {
				if serviceProofRecord.SignatureHex != "" {
					teeSignBytes, err := hex.DecodeString(serviceProofRecord.SignatureHex)
					if err != nil {
						return
					}
					blockhash, err = n.submitServiceResult(types.Bool(serviceProofRecord.Result), teeSignBytes, serviceProofRecord.BloomFilter, serviceProofRecord.TeePublicKey, verifySlip)
					if err != nil {
						n.Schal("err", err.Error())
					}
					if blockhash != "" {
						n.Cache.Put([]byte(fmt.Sprintf("%s%d", cache.Prefix_service_chall_result, chlgStart)), []byte("true"))
						serviceProofRecord.CanSubmitResult = false
						n.SaveServiceProve(serviceProofRecord)
					}
				}
			}
			return
		}
	}

	n.SetServiceChallenging(true)

	n.Schal("info", fmt.Sprintf("service file chain challenge: %v", chlgStart))

	files, totalChallengedLength, err := n.calcChallengeFiles(chlgStart)
	if err != nil {
		n.Schal("err", fmt.Sprintf("calcChallengeFiles err: %v", err))
		return
	}

	n.Schal("info", fmt.Sprintf("total number of files challenged: %d", totalChallengedLength))

	err = n.SaveChallRandom(chlgStart, rndIndex, rnd)
	if err != nil {
		n.Schal("err", fmt.Sprintf("Save service file challenge random err: %v", err))
	}

	serviceProofRecord.Start = chlgStart
	serviceProofRecord.CanSubmitProof = true
	serviceProofRecord.CanSubmitResult = true

	if totalChallengedLength <= 0 {
		teePuk, teeSign, bloomFilter, result, err := n.verifyEmpty(rndIndex, rnd)
		if err != nil {
			n.Schal("err", err.Error())
		} else {
			n.Schal("info", fmt.Sprintf("chall result is %v", result))
			serviceProofRecord.Result = result
			serviceProofRecord.BloomFilter = bloomFilter
			serviceProofRecord.TeePublicKey = teePuk
			serviceProofRecord.SignatureHex = hex.EncodeToString(teeSign)
			verifySuc = true
		}

		serviceProofRecord.Proof = []types.U8{}
		n.SaveServiceProve(serviceProofRecord)

		_, err = n.Get([]byte(fmt.Sprintf("%s%d", cache.Prefix_service_chall_proof, chlgStart)))
		if err != nil {
			n.Schal("info", "will submit chall proof")
			blockhash, err = n.submitServiceProof([]types.U8{}, slip)
			if blockhash != "" {
				n.Schal("info", fmt.Sprintf("submit chall proof hash: %s", blockhash))
				n.Cache.Put([]byte(fmt.Sprintf("%s%d", cache.Prefix_service_chall_proof, chlgStart)), []byte("true"))
				serviceProofRecord.CanSubmitProof = false
				n.SaveServiceProve(serviceProofRecord)
			}
			if err != nil {
				n.Schal("err", fmt.Sprintf("submitServiceProof err: %v", err))
			}
		} else {
			n.Schal("info", "already submited chall proof")
		}

		if !verifySuc {
			for {
				latestBlock, err = n.GetSubstrateAPI().RPC.Chain.GetBlockLatest()
				if err != nil {
					n.Schal("err", err.Error())
					time.Sleep(time.Second * 6)
					continue
				}
				if verifySlip < uint32(latestBlock.Block.Header.Number) {
					return
				}
				teePuk, teeSign, bloomFilter, result, err = n.verifyEmpty(rndIndex, rnd)
				if err != nil {
					n.Schal("err", err.Error())
					verifySuc = false
					time.Sleep(time.Second * 6)
					continue
				}
				serviceProofRecord.Result = result
				serviceProofRecord.BloomFilter = bloomFilter
				serviceProofRecord.TeePublicKey = teePuk
				serviceProofRecord.SignatureHex = hex.EncodeToString(teeSign)
				verifySuc = true
				break
			}
		}
		_, err = n.Get([]byte(fmt.Sprintf("%s%d", cache.Prefix_service_chall_proof, chlgStart)))
		if err == nil {
			blockhash, err = n.submitServiceResult(types.Bool(result), teeSign, bloomFilter, teePuk, verifySlip)
			if blockhash != "" {
				n.Cache.Put([]byte(fmt.Sprintf("service_chall_result:%d", chlgStart)), []byte("true"))
				serviceProofRecord.CanSubmitProof = false
				n.SaveServiceProve(serviceProofRecord)
			}
			if err != nil {
				n.Schal("err", err.Error())
			}
		}
	} else if totalChallengedLength <= maxNumberOfSingleVerification {
		names, us, mus, sigma, usig, err := n.calcSigma(files, rndIndex, rnd)
		if err != nil {
			n.Schal("err", err.Error())
			return
		}
		if sigma == "" {
			n.Schal("err", "proof is empty")
			return
		}
		var serviceProof = make([]types.U8, len(sigma))
		for i := 0; i < len(sigma); i++ {
			serviceProof[i] = types.U8(sigma[i])
		}

		serviceProofRecord.Proof = serviceProof
		n.SaveServiceProve(serviceProofRecord)

		_, err = n.Get([]byte(fmt.Sprintf("%s%d", cache.Prefix_service_chall_proof, chlgStart)))
		if err != nil {
			blockhash, err = n.submitServiceProof(serviceProof, slip)
			if blockhash != "" {
				n.Cache.Put([]byte(fmt.Sprintf("%s%d", cache.Prefix_service_chall_proof, chlgStart)), []byte("true"))
				serviceProofRecord.CanSubmitProof = false
				n.SaveServiceProve(serviceProofRecord)
			}
			if err != nil {
				n.Schal("err", err.Error())
			}
		}

		_, err = n.Get([]byte(fmt.Sprintf("%s%d", cache.Prefix_service_chall_proof, chlgStart)))
		if err == nil {
			for {
				latestBlock, err = n.GetSubstrateAPI().RPC.Chain.GetBlockLatest()
				if err != nil {
					n.Schal("err", err.Error())
					time.Sleep(time.Second * 6)
					continue
				}
				if verifySlip < uint32(latestBlock.Block.Header.Number)+3 {
					return
				}
				teePuk, teeSign, bloomFilter, result, err := n.onceBatchVerify(rndIndex, rnd, names, us, mus, usig, sigma)
				if err != nil {
					n.Schal("err", err.Error())
					time.Sleep(time.Second * 6)
					continue
				}
				blockhash, err = n.submitServiceResult(types.Bool(true), teeSign, bloomFilter, teePuk, verifySlip)
				if blockhash != "" {
					n.Cache.Put([]byte(fmt.Sprintf("%s%d", cache.Prefix_service_chall_result, chlgStart)), []byte("true"))
				}
				if err != nil {
					n.Schal("err", err.Error())
				}
				serviceProofRecord.Result = result
				serviceProofRecord.BloomFilter = bloomFilter
				serviceProofRecord.TeePublicKey = teePuk
				serviceProofRecord.SignatureHex = hex.EncodeToString(teeSign)
				serviceProofRecord.CanSubmitResult = false
				n.SaveServiceProve(serviceProofRecord)
				return
			}
		}
	} else {
		teePuk, teeSign, bloomFilter, proof, err := n.batchGenProofAndVerify(files, rndIndex, rnd, calcBatchQuantity(totalChallengedLength), slip)
		if err != nil {
			n.Schal("err", fmt.Sprintf("batchGenProofAndVerify err: %v", err))
			return
		}
		n.SetServiceChallenging(false)

		serviceProofRecord.Proof = proof
		serviceProofRecord.Result = true
		serviceProofRecord.BloomFilter = bloomFilter
		serviceProofRecord.TeePublicKey = teePuk
		serviceProofRecord.SignatureHex = hex.EncodeToString(teeSign)
		n.SaveServiceProve(serviceProofRecord)

		_, err = n.Get([]byte(fmt.Sprintf("%s%d", cache.Prefix_service_chall_proof, chlgStart)))
		if err != nil {
			blockhash, err = n.submitServiceProof(proof, slip)
			if blockhash != "" {
				n.Cache.Put([]byte(fmt.Sprintf("%s%d", cache.Prefix_service_chall_proof, chlgStart)), []byte("true"))
				serviceProofRecord.CanSubmitProof = false
				n.SaveServiceProve(serviceProofRecord)
			}
			if err != nil {
				n.Schal("err", err.Error())
			}
		}

		_, err = n.Get([]byte(fmt.Sprintf("%s%d", cache.Prefix_service_chall_proof, chlgStart)))
		if err == nil {
			blockhash, err = n.submitServiceResult(types.Bool(true), teeSign, bloomFilter, teePuk, verifySlip)
			if blockhash != "" {
				n.Cache.Put([]byte(fmt.Sprintf("%s%d", cache.Prefix_service_chall_result, chlgStart)), []byte("true"))
				serviceProofRecord.CanSubmitProof = false
				n.SaveServiceProve(serviceProofRecord)
			}
			if err != nil {
				n.Schal("err", err.Error())
			}
		}
	}
}

func calcBatchQuantity(total uint64) uint32 {
	return uint32(total/(total/maxNumberOfSingleVerification+1) + 1)
}

func (n *Node) submitServiceResult(result types.Bool, sign []byte, bloomFilter chain.BloomFilter, teePuk chain.WorkerPublicKey, verifySlip uint32) (string, error) {
	var (
		err       error
		blockHash string
	)
	latestBlock, err := n.GetSubstrateAPI().RPC.Chain.GetBlockLatest()
	if err == nil {
		if verifySlip < uint32(latestBlock.Block.Header.Number) {
			return "", nil
		}
	}
	for i := 0; i < 3; i++ {
		blockHash, err = n.SubmitVerifyServiceResult(result, sign, bloomFilter, teePuk)
		if blockHash != "" {
			return blockHash, err
		}
		if err != nil {
			n.Schal("err", fmt.Sprintf("submit service proof result: %v", err))
		}
		time.Sleep(time.Second)
		continue
	}
	return "", fmt.Errorf("submitServiceProof failed: %v", err)
}

func (n *Node) submitServiceProof(serviceProof []types.U8, slip uint32) (string, error) {
	var (
		err       error
		blockHash string
	)
	latestBlock, err := n.GetSubstrateAPI().RPC.Chain.GetBlockLatest()
	if err == nil {
		if slip < uint32(latestBlock.Block.Header.Number) {
			return "", fmt.Errorf("challenge expired: %d < %d", slip, latestBlock.Block.Header.Number)
		}
	}

	for i := 0; i < 3; i++ {
		n.Schal("info", fmt.Sprintf("[start SubmitServiceProof] %v", time.Now()))
		blockHash, err = n.SubmitServiceProof(serviceProof)
		n.Schal("info", fmt.Sprintf("[end SubmitServiceProof] hash: %s err: %v", blockHash, err))
		if blockHash != "" {
			return blockHash, err
		}
		time.Sleep(time.Second)
	}
	return blockHash, fmt.Errorf("submitServiceProof failed: %v", err)
}

func (n *Node) onceBatchGenProofAndVerify(files []challengedFile, randomIndexList []types.U32, randomList []chain.Random) (chain.WorkerPublicKey, []byte, chain.BloomFilter, []types.U8, bool, error) {
	names, us, mus, sigma, usig, err := n.calcSigma(files, randomIndexList, randomList)
	if err != nil {
		return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, []types.U8{}, false, err
	}

	var serviceProof = make([]types.U8, len(sigma))
	for i := 0; i < len(sigma); i++ {
		serviceProof[i] = types.U8(sigma[i])
	}

	teePuk, teeSign, bloomFilter, result, err := n.onceBatchVerify(randomIndexList, randomList, names, us, mus, usig, sigma)
	if err != nil {
		return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, serviceProof, false, err
	}
	return teePuk, teeSign, bloomFilter, serviceProof, result, nil
}

func (n *Node) calcSigma(files []challengedFile, randomIndexList []types.U32, randomList []chain.Random) ([]string, []string, []string, string, [][]byte, error) {
	var sigma string
	var proveResponse GenProofResponse
	var names = make([]string, 0)
	var us = make([]string, 0)
	var mus = make([]string, 0)
	var usig = make([][]byte, 0)

	qslice := calcQSlice(randomIndexList, randomList)

	timeout := time.NewTicker(time.Duration(time.Minute))
	defer timeout.Stop()

	for i := 0; i < len(files); i++ {
		n.Schal("info", fmt.Sprintf("will calc %s", files[i].Fid))
		for j := 0; j < len(files[i].Fragments); j++ {
			tag, err := n.checkTag(files[i].Fid, files[i].Fragments[j])
			if err != nil {
				n.Schal("err", fmt.Sprintf("checkTag: %v", err))
				n.GenerateRestoralOrder(files[i].Fid, files[i].Fragments[j])
				return names, us, mus, sigma, usig, fmt.Errorf("This challenge has failed due to an invalid tag: %s", files[i].Fragments[j])
			}
			_, err = os.Stat(files[i].Fragments[j])
			if err != nil {
				n.Schal("err", err.Error())
				n.GenerateRestoralOrder(files[i].Fid, files[i].Fragments[j])
				return names, us, mus, sigma, usig, fmt.Errorf("This challenge has failed due to missing fragment: %s", files[i].Fragments[j])
			}
			matrix, _, err := SplitByN(files[i].Fragments[j], int64(len(tag.Tag.T.Phi)))
			if err != nil {
				n.Schal("err", fmt.Sprintf("SplitByN %v err: %v", files[i].Fragments[j], err))
				return names, us, mus, sigma, usig, err
			}

			proveResponseCh := n.GenProof(qslice, nil, tag.Tag.T.Phi, matrix)
			timeout.Reset(time.Minute)
			select {
			case proveResponse = <-proveResponseCh:
			case <-timeout.C:
				proveResponse.StatueMsg.StatusCode = 0
			}

			if proveResponse.StatueMsg.StatusCode != Success {
				n.Schal("err", fmt.Sprintf("GenProof err: %d", proveResponse.StatueMsg.StatusCode))
				return names, us, mus, sigma, usig, err
			}

			sigmaTemp, ok := n.AggrAppendProof(sigma, proveResponse.Sigma)
			if !ok {
				n.Schal("err", "AggrAppendProof: false")
				return names, us, mus, sigma, usig, errors.New("AggrAppendProof failed")
			}
			sigma = sigmaTemp
			names = append(names, tag.Tag.T.Name)
			us = append(us, tag.Tag.T.U)
			mus = append(mus, proveResponse.MU)
			usig = append(usig, tag.USig)
		}
	}

	return names, us, mus, sigma, usig, nil
}

// func (n *Node) checkServiceProofRecord(challStart uint32) error {
// 	serviceProofRecord, err := n.LoadServiceProve()
// 	if err != nil {
// 		return err
// 	}

// 	if serviceProofRecord.Start != challStart {
// 		os.Remove(n.GetServiceProve())
// 		n.Del("info", n.GetServiceProve())
// 		return errors.New("Local service file challenge record is outdated")
// 	}

// 	if serviceProofRecord.CanSubmitProof {
// 		return errors.New("not submit proof")
// 	}

// 	if serviceProofRecord.CanSubmitResult {

// 		n.Schal("info", fmt.Sprintf("local service proof file challenge: %v", serviceProofRecord.Start))

// 		if serviceProofRecord.SignatureHex != "" {
// 			teeSignBytes, err := hex.DecodeString(serviceProofRecord.SignatureHex)
// 			if err != nil {
// 				return err
// 			}
// 			err = n.submitServiceResult(types.Bool(serviceProofRecord.Result), teeSignBytes, serviceProofRecord.BloomFilter, serviceProofRecord.TeePublicKey)
// 			if err != nil {
// 				n.Schal("err", err.Error())
// 			}
// 			serviceProofRecord.CanSubmitResult = false
// 			n.SaveServiceProve(serviceProofRecord)
// 			return nil
// 		}
// 	}

// 	return errors.New("Service proof result not submited")
// }

func (n *Node) onceBatchVerify(randomIndexList []types.U32, randomList []chain.Random, names, us, mus []string, usig [][]byte, sigma string) (chain.WorkerPublicKey, []byte, chain.BloomFilter, bool, error) {
	qslice_pb := calcQSlicePb(randomIndexList, randomList)
	var requestBatchVerify = &pb.RequestBatchVerify{
		AggProof: &pb.RequestBatchVerify_BatchVerifyParam{
			Names: names,
			Us:    us,
			Mus:   mus,
			Sigma: sigma,
		},
		MinerId: n.GetSignatureAccPulickey(),
		Qslices: &qslice_pb,
		USigs:   usig,
	}

	batchVerifyResult, _, err := n.requestBatchVerify(requestBatchVerify, "", 0)
	if err != nil {
		n.Schal("err", fmt.Sprintf("[requestBatchVerify] %v", err))
		return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, false, err
	}

	if len(batchVerifyResult.TeeAccountId) != chain.WorkerPublicKeyLen {
		return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, false, fmt.Errorf("The length of the tee publickey returned by tee is illegal: %d != %d", len(batchVerifyResult.TeeAccountId), chain.WorkerPublicKeyLen)
	}

	if len(batchVerifyResult.Signature) != chain.TeeSigLen {
		return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, false, fmt.Errorf("The length of the signature returned by tee is illegal: %d != %d", len(batchVerifyResult.Signature), chain.TeeSigLen)
	}

	n.Schal("info", fmt.Sprintf("once batch verification result: %v", batchVerifyResult.BatchVerifyResult))

	if len(batchVerifyResult.ServiceBloomFilter) > chain.BloomFilterLen {
		return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, false, fmt.Errorf("The length of the Bloom filter returned by tee is illegal: %d > %d", len(batchVerifyResult.ServiceBloomFilter), chain.BloomFilterLen)
	}
	var bloomFilterChain chain.BloomFilter
	for i := 0; i < len(batchVerifyResult.ServiceBloomFilter); i++ {
		bloomFilterChain[i] = types.U64(batchVerifyResult.ServiceBloomFilter[i])
	}

	var teePuk chain.WorkerPublicKey
	for i := 0; i < chain.WorkerPublicKeyLen; i++ {
		teePuk[i] = types.U8(batchVerifyResult.TeeAccountId[i])
	}

	return teePuk, batchVerifyResult.Signature, bloomFilterChain, batchVerifyResult.BatchVerifyResult, err
}

func (n *Node) verifyEmpty(randomIndexList []types.U32, randomList []chain.Random) (chain.WorkerPublicKey, []byte, chain.BloomFilter, bool, error) {
	qslice_pb := calcQSlicePb(randomIndexList, randomList)
	var requestBatchVerify = &pb.RequestBatchVerify{
		AggProof: &pb.RequestBatchVerify_BatchVerifyParam{
			Names: []string{},
			Us:    []string{},
			Mus:   []string{},
			Sigma: "",
		},
		MinerId: n.GetSignatureAccPulickey(),
		Qslices: &qslice_pb,
		USigs:   [][]byte{},
	}

	batchVerifyResult, _, err := n.requestBatchVerify(requestBatchVerify, "", 0)
	if err != nil {
		return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, false, err
	}

	if len(batchVerifyResult.TeeAccountId) != chain.WorkerPublicKeyLen {
		return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, false, fmt.Errorf("The length of the tee publickey returned by tee is illegal: %d != %d", len(batchVerifyResult.TeeAccountId), chain.WorkerPublicKeyLen)
	}

	if len(batchVerifyResult.Signature) != chain.TeeSigLen {
		return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, false, fmt.Errorf("The length of the signature returned by tee is illegal: %d != %d", len(batchVerifyResult.Signature), chain.TeeSigLen)
	}

	n.Schal("info", fmt.Sprintf("batch verification result of empty file: %v", batchVerifyResult.BatchVerifyResult))

	if len(batchVerifyResult.ServiceBloomFilter) > chain.BloomFilterLen {
		return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, false, fmt.Errorf("The length of the Bloom filter returned by tee is illegal: %d > %d", len(batchVerifyResult.ServiceBloomFilter), chain.BloomFilterLen)
	}
	var bloomFilterChain chain.BloomFilter
	for i := 0; i < len(batchVerifyResult.ServiceBloomFilter); i++ {
		bloomFilterChain[i] = types.U64(batchVerifyResult.ServiceBloomFilter[i])
	}

	var teePuk chain.WorkerPublicKey
	for i := 0; i < chain.WorkerPublicKeyLen; i++ {
		teePuk[i] = types.U8(batchVerifyResult.TeeAccountId[i])
	}

	return teePuk, batchVerifyResult.Signature, bloomFilterChain, batchVerifyResult.BatchVerifyResult, nil
}

func (n *Node) calcChallengeFragments(fid string, start uint32) ([]string, error) {
	var err error
	var fmeta chain.FileMetadata
	for i := 0; i < 3; i++ {
		fmeta, err = n.QueryFile(fid, int32(start))
		if err != nil {
			if errors.Is(err, chain.ERR_RPC_EMPTY_VALUE) {
				return []string{}, nil
			}
			time.Sleep(chain.BlockInterval)
			continue
		}
	}
	if err != nil {
		return []string{}, err
	}

	var challFragments = make([]string, 0)
	for i := 0; i < len(fmeta.SegmentList); i++ {
		for j := 0; j < len(fmeta.SegmentList[i].FragmentList); j++ {
			if sutils.CompareSlice(fmeta.SegmentList[i].FragmentList[j].Miner[:], n.GetSignatureAccPulickey()) {
				if fmeta.SegmentList[i].FragmentList[j].Tag.HasValue() {
					ok, block := fmeta.SegmentList[i].FragmentList[j].Tag.Unwrap()
					if !ok {
						return challFragments, fmt.Errorf("[%s] fragment.Tag.Unwrap failed", string(fmeta.SegmentList[i].FragmentList[j].Hash[:]))
					}
					if uint32(block) <= start {
						challFragments = append(challFragments, filepath.Join(n.GetFileDir(), fid, string(fmeta.SegmentList[i].FragmentList[j].Hash[:])))
					}
				}
			}
		}
	}
	return challFragments, nil
}

func (n *Node) checkTag(fid, fragment string) (TagfileType, error) {
	serviceTagPath := fragment + ".tag"
	//fragmentPath := filepath.Join(n.GetFileDir(), fid, fragment)
	buf, err := os.ReadFile(serviceTagPath)
	if err != nil {
		_, err = os.Stat(fragment)
		if err != nil {
			err = n.DownloadFragment(fid, filepath.Base(fragment), fragment)
			if err != nil {
				n.GenerateRestoralOrder(fid, fragment)
				return TagfileType{}, err
			}
		}
		err = n.calcFragmentTag(fid, fragment)
		if err != nil {
			n.Schal("err", fmt.Sprintf("calc the fragment tag failed: %v", err))
			n.GenerateRestoralOrder(fid, fragment)
			return TagfileType{}, err
		}
	}
	var tag = TagfileType{}
	err = json.Unmarshal(buf, &tag)
	if err != nil {
		n.Schal("err", fmt.Sprintf("invalid tag file: %v", err))
		os.Remove(serviceTagPath)
		n.Del("info", serviceTagPath)
		err = n.calcFragmentTag(fid, fragment)
		if err != nil {
			n.Schal("err", fmt.Sprintf("calc the fragment tag failed: %v", err))
			n.GenerateRestoralOrder(fid, fragment)
			return TagfileType{}, err
		}
	}

	buf, err = os.ReadFile(serviceTagPath)
	if err != nil {
		return TagfileType{}, err
	}

	err = json.Unmarshal(buf, &tag)
	return tag, err
}

func calcQSlice(randomIndexList []types.U32, randomList []chain.Random) []QElement {
	var qslice = make([]QElement, len(randomIndexList))
	for k, v := range randomIndexList {
		qslice[k].I = int64(v)
		var b = make([]byte, len(randomList[k]))
		for i := 0; i < len(randomList[k]); i++ {
			b[i] = byte(randomList[k][i])
		}
		qslice[k].V = new(big.Int).SetBytes(b).String()
	}
	return qslice
}

func calcQSlicePb(randomIndexList []types.U32, randomList []chain.Random) pb.Qslice {
	var qslice = pb.Qslice{}
	qslice.RandomIndexList = make([]uint32, len(randomIndexList))
	qslice.RandomList = make([][]byte, len(randomIndexList))
	for k, v := range randomIndexList {
		qslice.RandomIndexList[k] = uint32(v)
		var b = make([]byte, len(randomList[k]))
		for i := 0; i < len(randomList[k]); i++ {
			b[i] = byte(randomList[k][i])
		}
		qslice.RandomList[k] = b
	}
	return qslice
}

func (n *Node) calcChallengeFiles(challStart uint32) ([]challengedFile, uint64, error) {
	var err error
	var fid string
	var totalFiles uint64
	var files = make([]challengedFile, 0)

	serviceRoothashDir, err := utils.Dirs(n.GetFileDir())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve file dir: %v", err)
	}

	for i := int(0); i < len(serviceRoothashDir); i++ {
		fid = filepath.Base(serviceRoothashDir[i])
		n.Schal("info", fmt.Sprintf("check the file: %s", fid))
		fragments, err := n.calcChallengeFragments(fid, challStart)
		if err != nil {
			return nil, 0, fmt.Errorf("calcChallengeFragments: %v", err)
		}
		totalFiles = totalFiles + uint64(len(fragments))
		files = append(files, challengedFile{Fid: fid, Fragments: append(make([]string, 0), fragments...)})
		n.Schal("info", fmt.Sprintf("number of challenged fragments: %v", len(fragments)))
	}
	return files, totalFiles, nil
}

func (n *Node) batchGenProofAndVerify(files []challengedFile, randomIndexList []types.U32, randomList []chain.Random, number, slip uint32) (chain.WorkerPublicKey, []byte, chain.BloomFilter, []types.U8, error) {
	var ok bool
	var err error
	var sigma string
	var usedTee string
	var sigmaOnChian string
	var proveResponse GenProofResponse
	var batchVerifyResponse *pb.ResponseBatchVerify
	var names = make([]string, 0)
	var us = make([]string, 0)
	var mus = make([]string, 0)
	var usig = make([][]byte, 0)
	var verifyInServiceFileStructureList = make([]*pb.RequestAggregateSignature_VerifyInServiceFileStructure, 0)

	qElement := calcQSlice(randomIndexList, randomList)
	qSlicePb := calcQSlicePb(randomIndexList, randomList)

	var stackedBloomFilters = make([]uint64, 0)

	timeout := time.NewTicker(time.Duration(time.Minute))
	defer timeout.Stop()

	totalFile := 0
	var index uint32 = 1
	for i := int(0); i < len(files); i++ {
		for j := 0; j < len(files[i].Fragments); j++ {
			tag, err := n.checkTag(files[i].Fid, files[i].Fragments[j])
			if err != nil {
				n.Schal("err", fmt.Sprintf("checkTag: %v", err))
				n.GenerateRestoralOrder(files[i].Fid, files[i].Fragments[j])
				return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, nil, fmt.Errorf("This challenge has failed due to an invalid tag: %s", files[i].Fragments[j])
			}

			_, err = os.Stat(files[i].Fragments[j])
			if err != nil {
				n.Schal("err", err.Error())
				n.GenerateRestoralOrder(files[i].Fid, files[i].Fragments[j])
				return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, nil, fmt.Errorf("This challenge has failed due to missing fragment: %s", files[i].Fragments[j])
			}

			matrix, _, err := SplitByN(files[i].Fragments[j], int64(len(tag.Tag.T.Phi)))
			if err != nil {
				n.Schal("err", fmt.Sprintf("SplitByN %v err: %v", files[i].Fragments[j], err))
				return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, nil, err
			}

			proveResponseCh := n.GenProof(qElement, nil, tag.Tag.T.Phi, matrix)
			timeout.Reset(time.Minute * 3)
			select {
			case proveResponse = <-proveResponseCh:
			case <-timeout.C:
				return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, nil, errors.New("GenProof timeout")
			}

			if proveResponse.StatueMsg.StatusCode != Success {
				return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, nil, fmt.Errorf("GenProof failed: %d", proveResponse.StatueMsg.StatusCode)
			}

			sigma, ok = n.AggrAppendProof(sigma, proveResponse.Sigma)
			if !ok {
				return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, nil, errors.New("AggrAppendProof for sigma failed")
			}

			sigmaOnChian, ok = n.AggrAppendProof(sigmaOnChian, proveResponse.Sigma)
			if !ok {
				return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, nil, errors.New("AggrAppendProof for sigmaOnChian failed")
			}
			totalFile += 1
			names = append(names, tag.Tag.T.Name)
			us = append(us, tag.Tag.T.U)
			mus = append(mus, proveResponse.MU)
			usig = append(usig, tag.USig)

			if index%number == 0 {
				var request = &pb.RequestBatchVerify{
					AggProof: &pb.RequestBatchVerify_BatchVerifyParam{
						Names: names,
						Us:    us,
						Mus:   mus,
						Sigma: sigma,
					},
					Qslices:            &qSlicePb,
					USigs:              usig,
					MinerId:            n.GetSignatureAccPulickey(),
					ServiceBloomFilter: stackedBloomFilters,
				}

				n.Schal("info", fmt.Sprintf("names length: %d", len(names)))

				batchVerifyResponse, usedTee, err = n.requestBatchVerify(request, usedTee, slip)
				if err != nil {
					return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, nil, err
				}

				stackedBloomFilters = batchVerifyResponse.GetServiceBloomFilter()
				names = make([]string, 0)
				us = make([]string, 0)
				mus = make([]string, 0)
				usig = make([][]byte, 0)

				verifyInServiceFileStructureList = append(verifyInServiceFileStructureList, &pb.RequestAggregateSignature_VerifyInServiceFileStructure{
					MinerId:            n.GetSignatureAccPulickey(),
					Result:             batchVerifyResponse.GetBatchVerifyResult(),
					Sigma:              sigma,
					ServiceBloomFilter: batchVerifyResponse.GetServiceBloomFilter(),
					Signature:          batchVerifyResponse.GetSignature(),
				})
				sigma = ""
			}
			index += 1
		}
	}

	if len(names) > 0 {
		var request = &pb.RequestBatchVerify{
			AggProof: &pb.RequestBatchVerify_BatchVerifyParam{
				Names: names,
				Us:    us,
				Mus:   mus,
				Sigma: sigma,
			},
			Qslices:            &qSlicePb,
			USigs:              usig,
			MinerId:            n.GetSignatureAccPulickey(),
			ServiceBloomFilter: stackedBloomFilters,
		}

		n.Schal("info", fmt.Sprintf("names length: %d", len(names)))

		batchVerifyResponse, usedTee, err = n.requestBatchVerify(request, usedTee, slip)
		if err != nil {
			return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, nil, err
		}

		stackedBloomFilters = batchVerifyResponse.GetServiceBloomFilter()
		names = []string{}
		us = []string{}
		mus = []string{}
		usig = make([][]byte, 0)

		verifyInServiceFileStructureList = append(verifyInServiceFileStructureList, &pb.RequestAggregateSignature_VerifyInServiceFileStructure{
			MinerId:            n.GetSignatureAccPulickey(),
			Result:             batchVerifyResponse.GetBatchVerifyResult(),
			Sigma:              sigma,
			ServiceBloomFilter: batchVerifyResponse.GetServiceBloomFilter(),
			Signature:          batchVerifyResponse.GetSignature(),
		})
		sigma = ""
	}

	request := &pb.RequestAggregateSignature{
		VerifyInserviceFileHistory: verifyInServiceFileStructureList,
		Qslices:                    &qSlicePb,
	}

	aggregateSignatureResponse, err := n.requestAggregateSignature(request, usedTee, slip)
	if err != nil {
		return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, nil, err
	}

	if len(aggregateSignatureResponse.TeeAccountId) != chain.WorkerPublicKeyLen {
		return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, nil, fmt.Errorf("The length of the tee publickey returned by tee is illegal: %d != %d", len(aggregateSignatureResponse.TeeAccountId), chain.WorkerPublicKeyLen)
	}

	if len(aggregateSignatureResponse.Signature) != chain.TeeSigLen {
		return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, nil, fmt.Errorf("The length of the signature returned by tee is illegal: %d != %d", len(aggregateSignatureResponse.Signature), chain.TeeSigLen)
	}

	n.Schal("info", fmt.Sprintf("batch verification result: %v", true))

	var serviceProof = make([]types.U8, len(sigmaOnChian))
	for i := 0; i < len(sigmaOnChian); i++ {
		serviceProof[i] = types.U8(sigmaOnChian[i])
	}

	if len(verifyInServiceFileStructureList[len(verifyInServiceFileStructureList)-1].ServiceBloomFilter) > chain.BloomFilterLen {
		return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, nil, fmt.Errorf("The length of the Bloom filter returned by tee is illegal: %d > %d", len(verifyInServiceFileStructureList[len(verifyInServiceFileStructureList)-1].ServiceBloomFilter), chain.BloomFilterLen)
	}
	var bloomFilterChain chain.BloomFilter
	for i := 0; i < len(verifyInServiceFileStructureList[len(verifyInServiceFileStructureList)-1].ServiceBloomFilter); i++ {
		bloomFilterChain[i] = types.U64(verifyInServiceFileStructureList[len(verifyInServiceFileStructureList)-1].ServiceBloomFilter[i])
	}

	var teePuk chain.WorkerPublicKey
	for i := 0; i < chain.WorkerPublicKeyLen; i++ {
		teePuk[i] = types.U8(aggregateSignatureResponse.TeeAccountId[i])
	}

	return teePuk, aggregateSignatureResponse.Signature, bloomFilterChain, serviceProof, nil
}

func (n *Node) requestBatchVerify(request *pb.RequestBatchVerify, tee string, slip uint32) (*pb.ResponseBatchVerify, string, error) {
	var err error
	var dialOptions []grpc.DialOption
	var latestBlock *types.SignedBlock
	var tees []string
	var batchVerifyResponse *pb.ResponseBatchVerify
	if tee != "" {
		tees = append(tees, tee)
	} else {
		tees = n.GetAllVerifierTeeEndpoint()
	}

	for {
		latestBlock, err = n.GetSubstrateAPI().RPC.Chain.GetBlockLatest()
		if err != nil {
			n.Schal("err", fmt.Sprintf("GetBlockLatest: %v", err))
			time.Sleep(time.Second * 10)
			continue
		}

		if slip <= uint32(latestBlock.Block.Header.Number) {
			return nil, "", errors.New("challenge expired, RequestBatchVerify failed")
		}
		for i := 0; i < len(tees); i++ {
			if !strings.Contains(tees[i], "443") {
				dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
			} else {
				dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
			}
			batchVerifyResponse, err = com.RequestBatchVerify(tees[i], request, time.Minute*10, dialOptions, nil)
			if err != nil {
				n.Schal("err", fmt.Sprintf("RequestBatchVerify: %v", err))
				time.Sleep(time.Second * 10)
				continue
			}
			return batchVerifyResponse, tees[i], nil
		}
	}
}

func (n *Node) requestAggregateSignature(request *pb.RequestAggregateSignature, usedTee string, slip uint32) (*pb.ResponseAggregateSignature, error) {
	var dialOptions []grpc.DialOption
	if usedTee != "" {
		if !strings.Contains(usedTee, "443") {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		} else {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
		}
		for {
			latestBlock, err := n.GetSubstrateAPI().RPC.Chain.GetBlockLatest()
			if err != nil {
				n.Schal("err", fmt.Sprintf("GetBlockLatest: %v", err))
				time.Sleep(time.Second * 6)
				continue
			}

			if slip <= uint32(latestBlock.Block.Header.Number) {
				return nil, errors.New("challenge expired, requestAggregateSignature failed")
			}

			batchVerifyResponse, err := com.RequestAggregateSignature(usedTee, request, time.Minute*10, dialOptions, nil)
			if err != nil {
				n.Schal("err", fmt.Sprintf("RequestAggregateSignature: %v", err))
				time.Sleep(time.Second * 6)
				continue
			}
			return batchVerifyResponse, nil
		}
	}

	tees := n.GetAllVerifierTeeEndpoint()
	for i := 0; i < len(tees); i++ {
		if !strings.Contains(tees[i], "443") {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		} else {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
		}
		responseAggregateSignature, err := com.RequestAggregateSignature(tees[i], request, time.Minute*10, dialOptions, nil)
		if err != nil {
			n.Schal("err", fmt.Sprintf("RequestAggregateSignature: %v", err))
			continue
		}
		return responseAggregateSignature, nil
	}

	return nil, errors.New("RequestAggregateSignature failed")
}

func (n *Node) DownloadFragment(fid, fragment_hash, savepath string) error {
	fstat, err := os.Stat(savepath)
	if err == nil {
		if fstat.Size() == chain.FragmentSize {
			return nil
		}
	}

	var gwlist = []string{configs.DefaultGW1, configs.DefaultGW2, configs.DefaultGW3}
	ossList, err := n.QueryAllOss(-1)
	if err == nil {
		for i := 0; i < len(ossList); i++ {
			if strings.Contains(string(ossList[i].Domain), "cess.network") {
				continue
			}
			gwlist = append(gwlist, string(ossList[i].Domain))
		}
	}
	url := ""
	message := sutils.GetRandomcode(16)
	sig, err := sutils.SignedSR25519WithMnemonic(n.GetURI(), message)
	if err != nil {
		return fmt.Errorf("[SignedSR25519WithMnemonic] %v", err)
	}
	signstr := hex.EncodeToString(sig)
	for i := 0; i < len(gwlist); i++ {
		if strings.HasSuffix(gwlist[i], "/") {
			url = fmt.Sprintf("%sfragment/download?fid=%s&fragment=%s", url, fid, fragment_hash)
		} else {
			url = fmt.Sprintf("%s/fragment/download?fid=%s&fragment=%s", url, fid, fragment_hash)
		}
		err = DownloadFragmentFromGW(url, savepath, n.GetSignatureAcc(), message, signstr)
		if err != nil {
			continue
		}
		return nil
	}
	return errors.New("Failed to download the fragment from all gateways")
}

func DownloadFragmentFromGW(url, fpath, account, message, signature string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Account", account)
	req.Header.Set("Message", message)
	req.Header.Set("Signature", signature)

	client := &http.Client{
		Timeout:   time.Minute * 3,
		Transport: configs.GlobalTransport,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed code: %d", resp.StatusCode)
	}

	fd, err := os.Create(fpath)
	if err != nil {
		return err
	}
	defer func() {
		if fd != nil {
			fd.Close()
		}
	}()

	length, err := io.Copy(fd, resp.Body)
	if err != nil && err != io.EOF {
		return err
	}
	if length != chain.FragmentSize {
		fd.Close()
		fd = nil
		os.Remove(fpath)
		return fmt.Errorf("invalid fragment size: %d", length)
	}
	err = fd.Sync()
	if err != nil {
		return err
	}
	return nil
}
