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
	"strings"
	"time"

	"github.com/CESSProject/cess-go-sdk/chain"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/node/common"
	"github.com/CESSProject/cess-miner/pkg/com"
	"github.com/CESSProject/cess-miner/pkg/com/pb"
	"github.com/CESSProject/cess-miner/pkg/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (n *Node) serviceChallenge(
	ch chan<- bool,
	serviceProofSubmited bool,
	latestBlock,
	challVerifyExpiration uint32,
	challStart uint32,
	randomIndexList []types.U32,
	randomList []chain.Random,
	teePubkey chain.WorkerPublicKey,
) {
	defer func() {
		ch <- true
		//n.SetServiceChallengeFlag(false)
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	if challVerifyExpiration <= latestBlock {
		n.Schal("err", fmt.Sprintf("%d < %d", challVerifyExpiration, latestBlock))
		return
	}

	err := n.checkServiceProofRecord(serviceProofSubmited, challStart, randomIndexList, randomList, teePubkey)
	if err == nil {
		return
	}
	if serviceProofSubmited {
		return
	}

	n.Schal("info", fmt.Sprintf("Service file chain challenge: %v", challStart))

	var qslice = make([]QElement, len(randomIndexList))
	for k, v := range randomIndexList {
		qslice[k].I = int64(v)
		var b = make([]byte, chain.RandomLen)
		for i := 0; i < chain.RandomLen; i++ {
			b[i] = byte(randomList[k][i])
		}
		qslice[k].V = new(big.Int).SetBytes(b).String()
	}

	err = n.SaveChallRandom(challStart, randomIndexList, randomList)
	if err != nil {
		n.Schal("err", fmt.Sprintf("Save service file challenge random err: %v", err))
	}

	var serviceProofRecord common.ServiceProofInfo
	serviceProofRecord.Start = uint32(challStart)
	serviceProofRecord.SubmitProof = true
	serviceProofRecord.SubmitResult = true
	serviceProofRecord.Names,
		serviceProofRecord.Us,
		serviceProofRecord.Mus,
		serviceProofRecord.Sigma,
		serviceProofRecord.Usig, err = n.calcSigma(challStart, randomIndexList, randomList)
	if err != nil {
		n.Schal("err", fmt.Sprintf("[calcSigma] %v", err))
		return
	}

	n.SaveServiceProve(serviceProofRecord)

	var serviceProof = make([]types.U8, len(serviceProofRecord.Sigma))
	for i := 0; i < len(serviceProofRecord.Sigma); i++ {
		serviceProof[i] = types.U8(serviceProofRecord.Sigma[i])
	}

	for i := 0; i < 5; i++ {
		txhash, err := n.SubmitServiceProof(serviceProof)
		if err != nil {
			n.Schal("err", fmt.Sprintf("[SubmitServiceProof] %v", err))
			time.Sleep(time.Minute)
			continue
		}
		n.Schal("info", fmt.Sprintf("submit service aggr proof suc: %s", txhash))
		break
	}
	serviceProofRecord.SubmitProof = false
	n.SaveServiceProve(serviceProofRecord)
	time.Sleep(chain.BlockInterval * 3)

	_, chall, err := n.QueryChallengeSnapShot(n.GetSignatureAccPulickey(), -1)
	if err != nil {
		n.Schal("err", err.Error())
		return
	}
	if chall.ProveInfo.ServiceProve.HasValue() {
		_, sProve := chall.ProveInfo.ServiceProve.Unwrap()
		serviceProofRecord.AllocatedTeeWorkpuk = sProve.TeePubkey
	} else {
		return
	}

	n.SaveServiceProve(serviceProofRecord)
	var endpoint string
	teeInfo, err := n.GetTee(string(serviceProofRecord.AllocatedTeeWorkpuk[:]))
	if err != nil {
		n.Schal("err", err.Error())
		endpoint, err = n.QueryEndpoints(serviceProofRecord.AllocatedTeeWorkpuk, -1)
		if err != nil {
			n.Schal("err", err.Error())
			return
		}
		endpoint = ProcessTeeEndpoint(endpoint)
	} else {
		endpoint = teeInfo.EndPoint
	}
	var teeWorkpuk []byte
	serviceProofRecord.ServiceBloomFilter,
		teeWorkpuk,
		serviceProofRecord.Signature,
		serviceProofRecord.ServiceResult, err = n.batchVerify(randomIndexList, randomList, endpoint, serviceProofRecord)
	if err != nil {
		n.Schal("err", fmt.Sprintf("[batchVerify] %v", err))
		return
	}
	if len(teeWorkpuk) != chain.WorkerPublicKeyLen {
		n.Schal("err", fmt.Sprintf("Invalid tee work public key from tee returned: %v", len(teeWorkpuk)))
		return
	}
	for i := 0; i < chain.WorkerPublicKeyLen; i++ {
		serviceProofRecord.AllocatedTeeWorkpuk[i] = types.U8(teeWorkpuk[i])
	}
	n.Schal("info", fmt.Sprintf("Batch verification results of service files: %v", serviceProofRecord.ServiceResult))

	var signature chain.TeeSig
	if len(serviceProofRecord.Signature) != chain.TeeSigLen {
		n.Schal("err", "invalid batchVerify.Signature")
		return
	}
	for i := 0; i < chain.TeeSigLen; i++ {
		signature[i] = types.U8(serviceProofRecord.Signature[i])
	}

	var bloomFilter chain.BloomFilter
	if len(serviceProofRecord.ServiceBloomFilter) != chain.BloomFilterLen {
		n.Schal("err", "invalid batchVerify.ServiceBloomFilter")
		return
	}
	for i := 0; i < chain.BloomFilterLen; i++ {
		bloomFilter[i] = types.U64(serviceProofRecord.ServiceBloomFilter[i])
	}

	n.SaveServiceProve(serviceProofRecord)
	var teeSignBytes = make(types.Bytes, len(signature))
	for j := 0; j < len(signature); j++ {
		teeSignBytes[j] = byte(signature[j])
	}
	for i := 0; i < 5; i++ {
		txhash, err := n.SubmitVerifyServiceResult(
			types.Bool(serviceProofRecord.ServiceResult),
			teeSignBytes,
			bloomFilter,
			serviceProofRecord.AllocatedTeeWorkpuk,
		)
		if err != nil {
			n.Schal("err", fmt.Sprintf("[SubmitServiceProofResult] hash: %s, err: %v", txhash, err))
			time.Sleep(time.Minute)
			continue
		}
		n.Schal("info", fmt.Sprintf("submit service aggr proof result suc: %s", txhash))
		break
	}
	serviceProofRecord.SubmitResult = false
	n.SaveServiceProve(serviceProofRecord)
}

// calc sigma
func (n *Node) calcSigma(
	challStart uint32,
	randomIndexList []types.U32,
	randomList []chain.Random,
) ([]string, []string, []string, string, [][]byte, error) {
	var sigma string
	var roothash string
	var fragmentPath string
	var serviceTagPath string
	var proveResponse GenProofResponse
	var names = make([]string, 0)
	var us = make([]string, 0)
	var mus = make([]string, 0)
	var usig = make([][]byte, 0)
	var qslice = make([]QElement, len(randomIndexList))
	for k, v := range randomIndexList {
		qslice[k].I = int64(v)
		var b = make([]byte, len(randomList[k]))
		for i := 0; i < len(randomList[k]); i++ {
			b[i] = byte(randomList[k][i])
		}
		qslice[k].V = new(big.Int).SetBytes(b).String()
	}

	serviceRoothashDir, err := utils.Dirs(n.GetFileDir())
	if err != nil {
		n.Schal("err", fmt.Sprintf("[Dirs] %v", err))
		return names, us, mus, sigma, usig, err
	}

	timeout := time.NewTicker(time.Duration(time.Minute))
	defer timeout.Stop()

	for i := int(0); i < len(serviceRoothashDir); i++ {
		roothash = filepath.Base(serviceRoothashDir[i])
		n.Schal("info", fmt.Sprintf("will calc %s", roothash))

		fragments, err := n.calcChallengeFragments(roothash, challStart)
		if err != nil {
			n.Schal("err", fmt.Sprintf("calcChallengeFragments(%s): %v", roothash, err))
			return names, us, mus, sigma, usig, err
		}
		n.Schal("info", fmt.Sprintf("fragments: %v", fragments))
		for j := 0; j < len(fragments); j++ {
			fragmentPath = filepath.Join(n.GetFileDir(), roothash, fragments[j])
			serviceTagPath = filepath.Join(n.GetFileDir(), roothash, fragments[j]+".tag")
			tag, err := n.checkTag(roothash, fragments[j])
			if err != nil {
				n.Schal("err", fmt.Sprintf("checkTag: %v", err))
				continue
			}

			_, err = os.Stat(filepath.Join(n.GetFileDir(), roothash, fragments[j]))
			if err != nil {
				n.Schal("err", err.Error())
				return names, us, mus, sigma, usig, err
			}
			matrix, _, err := SplitByN(fragmentPath, int64(len(tag.Tag.T.Phi)))
			if err != nil {
				n.Schal("err", fmt.Sprintf("SplitByN %v err: %v", serviceTagPath, err))
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
				n.Schal("err", fmt.Sprintf("GenProof  err: %d", proveResponse.StatueMsg.StatusCode))
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

func (n *Node) checkServiceProofRecord(
	serviceProofSubmited bool,
	challStart uint32,
	randomIndexList []types.U32,
	randomList []chain.Random,
	teePubkey chain.WorkerPublicKey,
) error {
	serviceProofRecord, err := n.LoadServiceProve()
	if err != nil {
		return err
	}

	if serviceProofRecord.Start != challStart {
		os.Remove(n.GetServiceProve())
		n.Del("info", n.GetServiceProve())
		return errors.New("Local service file challenge record is outdated")
	}

	if !serviceProofRecord.SubmitProof && !serviceProofRecord.SubmitResult {
		return nil
	}

	n.Schal("info", fmt.Sprintf("local service proof file challenge: %v", serviceProofRecord.Start))

	if !serviceProofSubmited && serviceProofRecord.SubmitProof {
		if serviceProofRecord.Names == nil ||
			serviceProofRecord.Us == nil ||
			serviceProofRecord.Mus == nil {
			serviceProofRecord.Names,
				serviceProofRecord.Us,
				serviceProofRecord.Mus,
				serviceProofRecord.Sigma,
				serviceProofRecord.Usig, err = calcSigma(cli, ws, l, teeRecord, rasKey, cfg, challStart, randomIndexList, randomList)
			if err != nil {
				n.Schal("err", fmt.Sprintf("[calcSigma] %v", err))
				return nil
			}
		}
		n.SaveServiceProve(serviceProofRecord)

		var serviceProve = make([]types.U8, len(serviceProofRecord.Sigma))
		for i := 0; i < len(serviceProofRecord.Sigma); i++ {
			serviceProve[i] = types.U8(serviceProofRecord.Sigma[i])
		}
		_, err = n.SubmitServiceProof(serviceProve)
		if err != nil {
			n.Schal("err", fmt.Sprintf("[SubmitServiceProof] %v", err))
			return nil
		}
		time.Sleep(chain.BlockInterval * 3)
		_, chall, err := n.QueryChallengeSnapShot(n.GetSignatureAccPulickey(), -1)
		if err != nil {
			return err
		}
		if chall.ProveInfo.ServiceProve.HasValue() {
			_, sProve := chall.ProveInfo.ServiceProve.Unwrap()
			serviceProofRecord.AllocatedTeeWorkpuk = sProve.TeePubkey
		} else {
			return errors.New("chall.ProveInfo.ServiceProve is empty")
		}
	} else {
		if chain.IsWorkerPublicKeyAllZero(teePubkey) {
			_, chall, err := n.QueryChallengeSnapShot(n.GetSignatureAccPulickey(), -1)
			if err != nil {
				return err
			}
			if chall.ProveInfo.ServiceProve.HasValue() {
				_, sProve := chall.ProveInfo.ServiceProve.Unwrap()
				serviceProofRecord.AllocatedTeeWorkpuk = sProve.TeePubkey
			} else {
				return errors.New("chall.ProveInfo.ServiceProve is empty")
			}
		} else {
			serviceProofRecord.AllocatedTeeWorkpuk = teePubkey
		}
	}

	if !serviceProofRecord.SubmitResult {
		return nil
	}

	for {
		if serviceProofRecord.ServiceBloomFilter != nil &&
			serviceProofRecord.Signature != nil {
			if len(serviceProofRecord.Signature) != chain.TeeSigLen {
				n.Schal("err", "invalid batchVerify.Signature")
				break
			}
			var bloomFilter chain.BloomFilter
			if len(serviceProofRecord.ServiceBloomFilter) != chain.BloomFilterLen {
				n.Schal("err", "invalid batchVerify.ServiceBloomFilter")
				break
			}
			for i := 0; i < chain.BloomFilterLen; i++ {
				bloomFilter[i] = types.U64(serviceProofRecord.ServiceBloomFilter[i])
			}
			for i := 0; i < 5; i++ {
				txhash, err := n.SubmitVerifyServiceResult(
					types.Bool(serviceProofRecord.ServiceResult),
					serviceProofRecord.Signature[:],
					bloomFilter,
					serviceProofRecord.AllocatedTeeWorkpuk,
				)
				if err != nil {
					n.Schal("err", fmt.Sprintf("[SubmitServiceProofResult] hash: %s, err: %v", txhash, err))
					time.Sleep(time.Minute)
					continue
				}
				n.Schal("info", fmt.Sprintf("submit service aggr proof result suc: %s", txhash))
				break
			}
			serviceProofRecord.SubmitResult = false
			n.SaveServiceProve(serviceProofRecord)
			return nil
		}
		break
	}
	var endpoint string
	teeInfo, err := n.GetTee(string(serviceProofRecord.AllocatedTeeWorkpuk[:]))
	if err != nil {
		n.Schal("err", err.Error())
		endpoint, err = n.QueryEndpoints(serviceProofRecord.AllocatedTeeWorkpuk, -1)
		if err != nil {
			n.Schal("err", err.Error())
			return err
		}
		endpoint = ProcessTeeEndpoint(endpoint)
	} else {
		endpoint = teeInfo.EndPoint
	}

	var teeWorkpuk []byte
	serviceProofRecord.ServiceBloomFilter,
		teeWorkpuk,
		serviceProofRecord.Signature,
		serviceProofRecord.ServiceResult, err = n.batchVerify(randomIndexList, randomList, endpoint, serviceProofRecord)
	if err != nil {
		return nil
	}
	if len(teeWorkpuk) != chain.WorkerPublicKeyLen {
		n.Schal("err", fmt.Sprintf("Invalid tee work public key from tee returned: %v", len(teeWorkpuk)))
		return nil
	}
	for i := 0; i < chain.WorkerPublicKeyLen; i++ {
		serviceProofRecord.AllocatedTeeWorkpuk[i] = types.U8(teeWorkpuk[i])
	}
	n.Schal("info", fmt.Sprintf("Batch verification results of service files: %v", serviceProofRecord.ServiceResult))
	if len(serviceProofRecord.Signature) != chain.TeeSigLen {
		n.Schal("err", "invalid batchVerify.Signature")
		return nil
	}
	var bloomFilter chain.BloomFilter
	if len(serviceProofRecord.ServiceBloomFilter) != chain.BloomFilterLen {
		n.Schal("err", "invalid batchVerify.ServiceBloomFilter")
		return nil
	}
	for i := 0; i < chain.BloomFilterLen; i++ {
		bloomFilter[i] = types.U64(serviceProofRecord.ServiceBloomFilter[i])
	}
	n.SaveServiceProve(serviceProofRecord)

	for i := 0; i < 5; i++ {
		txhash, err := n.SubmitVerifyServiceResult(
			types.Bool(serviceProofRecord.ServiceResult),
			serviceProofRecord.Signature[:],
			bloomFilter,
			serviceProofRecord.AllocatedTeeWorkpuk,
		)
		if err != nil {
			n.Schal("err", fmt.Sprintf("[SubmitServiceProofResult] hash: %s, err: %v", txhash, err))
			time.Sleep(time.Minute)
			continue
		}
		n.Schal("info", fmt.Sprintf("submit service aggr proof result suc: %s", txhash))
		break
	}
	serviceProofRecord.SubmitResult = false
	n.SaveServiceProve(serviceProofRecord)
	return nil
}

func (n *Node) batchVerify(
	randomIndexList []types.U32,
	randomList []chain.Random,
	teeEndPoint string,
	serviceProofRecord common.ServiceProofInfo,
) ([]uint64, []byte, []byte, bool, error) {
	var err error
	qslice_pb := encodeToRequestBatchVerify_Qslice(randomIndexList, randomList)
	var batchVerifyParam = &pb.RequestBatchVerify_BatchVerifyParam{
		Names: serviceProofRecord.Names,
		Us:    serviceProofRecord.Us,
		Mus:   serviceProofRecord.Mus,
		Sigma: serviceProofRecord.Sigma,
	}
	var batchVerifyResult *pb.ResponseBatchVerify
	var timeoutStep time.Duration = 10
	var timeout time.Duration
	var requestBatchVerify = &pb.RequestBatchVerify{
		AggProof: batchVerifyParam,
		MinerId:  n.GetSignatureAccPulickey(),
		Qslices:  qslice_pb,
		USigs:    serviceProofRecord.Usig,
	}
	var dialOptions []grpc.DialOption
	if !strings.Contains(teeEndPoint, "443") {
		dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	} else {
		dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
	}
	n.Schal("info", fmt.Sprintf("req tee batch verify: %s", teeEndPoint))
	n.Schal("info", fmt.Sprintf("serviceProofRecord.Names: %v", serviceProofRecord.Names))
	n.Schal("info", fmt.Sprintf("len(serviceProofRecord.Us): %v", len(serviceProofRecord.Us)))
	n.Schal("info", fmt.Sprintf("len(serviceProofRecord.Mus): %v", len(serviceProofRecord.Mus)))
	n.Schal("info", fmt.Sprintf("Sigma: %v", serviceProofRecord.Sigma))
	for i := 0; i < 5; {
		timeout = time.Minute * timeoutStep
		batchVerifyResult, err = com.RequestBatchVerify(
			teeEndPoint,
			requestBatchVerify,
			timeout,
			dialOptions,
			nil,
		)
		if err != nil {
			if strings.Contains(err.Error(), configs.Err_ctx_exceeded) {
				i++
				n.Schal("err", fmt.Sprintf("[RequestBatchVerify] %v", err))
				timeoutStep += 10
				time.Sleep(time.Minute * 3)
				continue
			}
			if strings.Contains(err.Error(), configs.Err_tee_Busy) {
				n.Schal("err", fmt.Sprintf("[RequestBatchVerify] %v", err))
				time.Sleep(time.Minute * 3)
				continue
			}
			n.Schal("err", fmt.Sprintf("[RequestBatchVerify] %v", err))
			return nil, nil, nil, false, err
		}
		return batchVerifyResult.ServiceBloomFilter, batchVerifyResult.TeeAccountId, batchVerifyResult.Signature, batchVerifyResult.BatchVerifyResult, err
	}
	return nil, nil, nil, false, err
}

func encodeToRequestBatchVerify_Qslice(randomIndexList []types.U32, randomList []chain.Random) *pb.RequestBatchVerify_Qslice {
	var randomIndexList_pb = make([]uint32, len(randomIndexList))
	for i := 0; i < len(randomIndexList); i++ {
		randomIndexList_pb[i] = uint32(randomIndexList[i])
	}
	var randomList_pb = make([][]byte, len(randomList))
	for i := 0; i < len(randomList); i++ {
		randomList_pb[i] = make([]byte, len(randomList[i]))
		for j := 0; j < len(randomList[i]); j++ {
			randomList_pb[i][j] = byte(randomList[i][j])
		}
	}
	return &pb.RequestBatchVerify_Qslice{
		RandomIndexList: randomIndexList_pb,
		RandomList:      randomList_pb,
	}
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
						return challFragments, fmt.Errorf("[%s] fragment.Tag.Unwrap %v", string(fmeta.SegmentList[i].FragmentList[j].Hash[:]), err)
					}
					if uint32(block) <= start {
						challFragments = append(challFragments, string(fmeta.SegmentList[i].FragmentList[j].Hash[:]))
					}
				}
			}
		}
	}
	return challFragments, nil
}

func (n *Node) checkTag(fid, fragment string) (TagfileType, error) {
	serviceTagPath := filepath.Join(n.GetFileDir(), fid, fragment+".tag")
	fragmentPath := filepath.Join(n.GetFileDir(), fid, fragment)
	buf, err := os.ReadFile(serviceTagPath)
	if err != nil {
		err = n.calcFragmentTag(fid, fragmentPath)
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
		err = n.calcFragmentTag(fid, fragmentPath)
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
