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
	"github.com/CESSProject/cess-miner/pkg/cache"
	"github.com/CESSProject/cess-miner/pkg/confile"
	"github.com/CESSProject/cess-miner/pkg/logger"
	"github.com/CESSProject/cess-miner/pkg/utils"
	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func serviceChallenge(
	cli *chain.ChainClient,
	r *RunningState,
	l logger.Logger,
	teeRecord *TeeRecord,
	peernode *core.PeerNode,
	ws *Workspace,
	cace cache.Cache,
	rsaKey *RSAKeyPair,
	cfg *confile.Confile,
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
		r.SetServiceChallengeFlag(false)
		if err := recover(); err != nil {
			l.Pnc(utils.RecoverError(err))
		}
	}()

	if challVerifyExpiration <= latestBlock {
		l.Schal("err", fmt.Sprintf("%d < %d", challVerifyExpiration, latestBlock))
		return
	}

	err := checkServiceProofRecord(cli, l, peernode, ws, teeRecord, cace, rsaKey, cfg, serviceProofSubmited, challStart, randomIndexList, randomList, teePubkey)
	if err == nil {
		return
	}
	if serviceProofSubmited {
		return
	}

	l.Schal("info", fmt.Sprintf("Service file chain challenge: %v", challStart))

	var qslice = make([]QElement, len(randomIndexList))
	for k, v := range randomIndexList {
		qslice[k].I = int64(v)
		var b = make([]byte, chain.RandomLen)
		for i := 0; i < chain.RandomLen; i++ {
			b[i] = byte(randomList[k][i])
		}
		qslice[k].V = new(big.Int).SetBytes(b).String()
	}

	err = ws.SaveChallRandom(challStart, randomIndexList, randomList)
	if err != nil {
		l.Schal("err", fmt.Sprintf("Save service file challenge random err: %v", err))
	}

	var serviceProofRecord serviceProofInfo
	serviceProofRecord.Start = uint32(challStart)
	serviceProofRecord.SubmitProof = true
	serviceProofRecord.SubmitResult = true
	serviceProofRecord.Names,
		serviceProofRecord.Us,
		serviceProofRecord.Mus,
		serviceProofRecord.Sigma,
		serviceProofRecord.Usig, err = calcSigma(cli, ws, l, teeRecord, rsaKey, cfg, challStart, randomIndexList, randomList)
	if err != nil {
		l.Schal("err", fmt.Sprintf("[calcSigma] %v", err))
		return
	}

	ws.SaveServiceProve(serviceProofRecord)

	var serviceProof = make([]types.U8, len(serviceProofRecord.Sigma))
	for i := 0; i < len(serviceProofRecord.Sigma); i++ {
		serviceProof[i] = types.U8(serviceProofRecord.Sigma[i])
	}

	for i := 0; i < 5; i++ {
		txhash, err := cli.SubmitServiceProof(serviceProof)
		if err != nil {
			l.Schal("err", fmt.Sprintf("[SubmitServiceProof] %v", err))
			time.Sleep(time.Minute)
			continue
		}
		l.Schal("info", fmt.Sprintf("submit service aggr proof suc: %s", txhash))
		break
	}
	serviceProofRecord.SubmitProof = false
	ws.SaveServiceProve(serviceProofRecord)
	time.Sleep(chain.BlockInterval * 3)

	_, chall, err := cli.QueryChallengeSnapShot(cli.GetSignatureAccPulickey(), -1)
	if err != nil {
		l.Schal("err", err.Error())
		return
	}
	if chall.ProveInfo.ServiceProve.HasValue() {
		_, sProve := chall.ProveInfo.ServiceProve.Unwrap()
		serviceProofRecord.AllocatedTeeWorkpuk = sProve.TeePubkey
	} else {
		return
	}

	ws.SaveServiceProve(serviceProofRecord)
	var endpoint string
	teeInfo, err := teeRecord.GetTee(string(serviceProofRecord.AllocatedTeeWorkpuk[:]))
	if err != nil {
		l.Schal("err", err.Error())
		endpoint, err = cli.QueryEndpoints(serviceProofRecord.AllocatedTeeWorkpuk, -1)
		if err != nil {
			l.Schal("err", err.Error())
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
		serviceProofRecord.ServiceResult, err = batchVerify(cli, l, peernode, randomIndexList, randomList, endpoint, serviceProofRecord)
	if err != nil {
		l.Schal("err", fmt.Sprintf("[batchVerify] %v", err))
		return
	}
	if len(teeWorkpuk) != chain.WorkerPublicKeyLen {
		l.Schal("err", fmt.Sprintf("Invalid tee work public key from tee returned: %v", len(teeWorkpuk)))
		return
	}
	for i := 0; i < chain.WorkerPublicKeyLen; i++ {
		serviceProofRecord.AllocatedTeeWorkpuk[i] = types.U8(teeWorkpuk[i])
	}
	l.Schal("info", fmt.Sprintf("Batch verification results of service files: %v", serviceProofRecord.ServiceResult))

	var signature chain.TeeSig
	if len(serviceProofRecord.Signature) != chain.TeeSigLen {
		l.Schal("err", "invalid batchVerify.Signature")
		return
	}
	for i := 0; i < chain.TeeSigLen; i++ {
		signature[i] = types.U8(serviceProofRecord.Signature[i])
	}

	var bloomFilter chain.BloomFilter
	if len(serviceProofRecord.ServiceBloomFilter) != chain.BloomFilterLen {
		l.Schal("err", "invalid batchVerify.ServiceBloomFilter")
		return
	}
	for i := 0; i < chain.BloomFilterLen; i++ {
		bloomFilter[i] = types.U64(serviceProofRecord.ServiceBloomFilter[i])
	}

	ws.SaveServiceProve(serviceProofRecord)
	var teeSignBytes = make(types.Bytes, len(signature))
	for j := 0; j < len(signature); j++ {
		teeSignBytes[j] = byte(signature[j])
	}
	for i := 0; i < 5; i++ {
		txhash, err := cli.SubmitVerifyServiceResult(
			types.Bool(serviceProofRecord.ServiceResult),
			teeSignBytes,
			bloomFilter,
			serviceProofRecord.AllocatedTeeWorkpuk,
		)
		if err != nil {
			l.Schal("err", fmt.Sprintf("[SubmitServiceProofResult] hash: %s, err: %v", txhash, err))
			time.Sleep(time.Minute)
			continue
		}
		l.Schal("info", fmt.Sprintf("submit service aggr proof result suc: %s", txhash))
		break
	}
	serviceProofRecord.SubmitResult = false
	ws.SaveServiceProve(serviceProofRecord)
}

// calc sigma
func calcSigma(
	cli *chain.ChainClient,
	ws *Workspace,
	l logger.Logger,
	teeRecord *TeeRecord,
	rsaKey *RSAKeyPair,
	cfg *confile.Confile,
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

	serviceRoothashDir, err := utils.Dirs(ws.GetFileDir())
	if err != nil {
		l.Schal("err", fmt.Sprintf("[Dirs] %v", err))
		return names, us, mus, sigma, usig, err
	}

	timeout := time.NewTicker(time.Duration(time.Minute))
	defer timeout.Stop()

	for i := int(0); i < len(serviceRoothashDir); i++ {
		roothash = filepath.Base(serviceRoothashDir[i])
		l.Schal("info", fmt.Sprintf("will calc %s", roothash))

		fragments, err := calcChallengeFragments(cli, roothash, challStart)
		if err != nil {
			l.Schal("err", fmt.Sprintf("calcChallengeFragments(%s): %v", roothash, err))
			return names, us, mus, sigma, usig, err
		}
		l.Schal("info", fmt.Sprintf("fragments: %v", fragments))
		for j := 0; j < len(fragments); j++ {
			fragmentPath = filepath.Join(ws.GetFileDir(), roothash, fragments[j])
			serviceTagPath = filepath.Join(ws.GetFileDir(), roothash, fragments[j]+".tag")
			tag, err := checkTag(cli, ws, teeRecord, cfg, l, roothash, fragments[j])
			if err != nil {
				l.Schal("err", fmt.Sprintf("checkTag: %v", err))
				continue
			}

			_, err = os.Stat(filepath.Join(ws.GetFileDir(), roothash, fragments[j]))
			if err != nil {
				l.Schal("err", err.Error())
				return names, us, mus, sigma, usig, err
			}
			matrix, _, err := SplitByN(fragmentPath, int64(len(tag.Tag.T.Phi)))
			if err != nil {
				l.Schal("err", fmt.Sprintf("SplitByN %v err: %v", serviceTagPath, err))
				return names, us, mus, sigma, usig, err
			}

			if rsaKey == nil || rsaKey.Spk == nil {
				l.Schal("err", "rsa public key is nil")
				return names, us, mus, sigma, usig, errors.New("rsa public key is nil")
			}

			proveResponseCh := rsaKey.GenProof(qslice, nil, tag.Tag.T.Phi, matrix)
			timeout.Reset(time.Minute)
			select {
			case proveResponse = <-proveResponseCh:
			case <-timeout.C:
				proveResponse.StatueMsg.StatusCode = 0
			}

			if proveResponse.StatueMsg.StatusCode != Success {
				l.Schal("err", fmt.Sprintf("GenProof  err: %d", proveResponse.StatueMsg.StatusCode))
				return names, us, mus, sigma, usig, err
			}

			sigmaTemp, ok := rsaKey.AggrAppendProof(sigma, proveResponse.Sigma)
			if !ok {
				l.Schal("err", "AggrAppendProof: false")
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

func checkServiceProofRecord(
	cli *chain.ChainClient,
	l logger.Logger,
	peernode *core.PeerNode,
	ws *Workspace,
	teeRecord *TeeRecord,
	cace cache.Cache,
	rasKey *RSAKeyPair,
	cfg *confile.Confile,
	serviceProofSubmited bool,
	challStart uint32,
	randomIndexList []types.U32,
	randomList []chain.Random,
	teePubkey chain.WorkerPublicKey,
) error {
	serviceProofRecord, err := ws.LoadServiceProve()
	if err != nil {
		return err
	}

	if serviceProofRecord.Start != challStart {
		os.Remove(ws.GetServiceProve())
		l.Del("info", ws.GetServiceProve())
		return errors.New("Local service file challenge record is outdated")
	}

	if !serviceProofRecord.SubmitProof && !serviceProofRecord.SubmitResult {
		return nil
	}

	l.Schal("info", fmt.Sprintf("local service proof file challenge: %v", serviceProofRecord.Start))

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
				l.Schal("err", fmt.Sprintf("[calcSigma] %v", err))
				return nil
			}
		}
		ws.SaveServiceProve(serviceProofRecord)

		var serviceProve = make([]types.U8, len(serviceProofRecord.Sigma))
		for i := 0; i < len(serviceProofRecord.Sigma); i++ {
			serviceProve[i] = types.U8(serviceProofRecord.Sigma[i])
		}
		_, err = cli.SubmitServiceProof(serviceProve)
		if err != nil {
			l.Schal("err", fmt.Sprintf("[SubmitServiceProof] %v", err))
			return nil
		}
		time.Sleep(chain.BlockInterval * 3)
		_, chall, err := cli.QueryChallengeSnapShot(cli.GetSignatureAccPulickey(), -1)
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
			_, chall, err := cli.QueryChallengeSnapShot(cli.GetSignatureAccPulickey(), -1)
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
				l.Schal("err", "invalid batchVerify.Signature")
				break
			}
			var bloomFilter chain.BloomFilter
			if len(serviceProofRecord.ServiceBloomFilter) != chain.BloomFilterLen {
				l.Schal("err", "invalid batchVerify.ServiceBloomFilter")
				break
			}
			for i := 0; i < chain.BloomFilterLen; i++ {
				bloomFilter[i] = types.U64(serviceProofRecord.ServiceBloomFilter[i])
			}
			for i := 0; i < 5; i++ {
				txhash, err := cli.SubmitVerifyServiceResult(
					types.Bool(serviceProofRecord.ServiceResult),
					serviceProofRecord.Signature[:],
					bloomFilter,
					serviceProofRecord.AllocatedTeeWorkpuk,
				)
				if err != nil {
					l.Schal("err", fmt.Sprintf("[SubmitServiceProofResult] hash: %s, err: %v", txhash, err))
					time.Sleep(time.Minute)
					continue
				}
				l.Schal("info", fmt.Sprintf("submit service aggr proof result suc: %s", txhash))
				break
			}
			serviceProofRecord.SubmitResult = false
			ws.SaveServiceProve(serviceProofRecord)
			return nil
		}
		break
	}
	var endpoint string
	teeInfo, err := teeRecord.GetTee(string(serviceProofRecord.AllocatedTeeWorkpuk[:]))
	if err != nil {
		l.Schal("err", err.Error())
		endpoint, err = cli.QueryEndpoints(serviceProofRecord.AllocatedTeeWorkpuk, -1)
		if err != nil {
			l.Schal("err", err.Error())
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
		serviceProofRecord.ServiceResult, err = batchVerify(cli, l, peernode, randomIndexList, randomList, endpoint, serviceProofRecord)
	if err != nil {
		return nil
	}
	if len(teeWorkpuk) != chain.WorkerPublicKeyLen {
		l.Schal("err", fmt.Sprintf("Invalid tee work public key from tee returned: %v", len(teeWorkpuk)))
		return nil
	}
	for i := 0; i < chain.WorkerPublicKeyLen; i++ {
		serviceProofRecord.AllocatedTeeWorkpuk[i] = types.U8(teeWorkpuk[i])
	}
	l.Schal("info", fmt.Sprintf("Batch verification results of service files: %v", serviceProofRecord.ServiceResult))
	if len(serviceProofRecord.Signature) != chain.TeeSigLen {
		l.Schal("err", "invalid batchVerify.Signature")
		return nil
	}
	var bloomFilter chain.BloomFilter
	if len(serviceProofRecord.ServiceBloomFilter) != chain.BloomFilterLen {
		l.Schal("err", "invalid batchVerify.ServiceBloomFilter")
		return nil
	}
	for i := 0; i < chain.BloomFilterLen; i++ {
		bloomFilter[i] = types.U64(serviceProofRecord.ServiceBloomFilter[i])
	}
	ws.SaveServiceProve(serviceProofRecord)

	for i := 0; i < 5; i++ {
		txhash, err := cli.SubmitVerifyServiceResult(
			types.Bool(serviceProofRecord.ServiceResult),
			serviceProofRecord.Signature[:],
			bloomFilter,
			serviceProofRecord.AllocatedTeeWorkpuk,
		)
		if err != nil {
			l.Schal("err", fmt.Sprintf("[SubmitServiceProofResult] hash: %s, err: %v", txhash, err))
			time.Sleep(time.Minute)
			continue
		}
		l.Schal("info", fmt.Sprintf("submit service aggr proof result suc: %s", txhash))
		break
	}
	serviceProofRecord.SubmitResult = false
	ws.SaveServiceProve(serviceProofRecord)
	return nil
}

func batchVerify(
	cli *chain.ChainClient,
	l logger.Logger,
	peernode *core.PeerNode,
	randomIndexList []types.U32,
	randomList []chain.Random,
	teeEndPoint string,
	serviceProofRecord serviceProofInfo,
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
		MinerId:  cli.GetSignatureAccPulickey(),
		Qslices:  qslice_pb,
		USigs:    serviceProofRecord.Usig,
	}
	var dialOptions []grpc.DialOption
	if !strings.Contains(teeEndPoint, "443") {
		dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	} else {
		dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
	}
	l.Schal("info", fmt.Sprintf("req tee batch verify: %s", teeEndPoint))
	l.Schal("info", fmt.Sprintf("serviceProofRecord.Names: %v", serviceProofRecord.Names))
	l.Schal("info", fmt.Sprintf("len(serviceProofRecord.Us): %v", len(serviceProofRecord.Us)))
	l.Schal("info", fmt.Sprintf("len(serviceProofRecord.Mus): %v", len(serviceProofRecord.Mus)))
	l.Schal("info", fmt.Sprintf("Sigma: %v", serviceProofRecord.Sigma))
	for i := 0; i < 5; {
		timeout = time.Minute * timeoutStep
		batchVerifyResult, err = peernode.RequestBatchVerify(
			teeEndPoint,
			requestBatchVerify,
			timeout,
			dialOptions,
			nil,
		)
		if err != nil {
			if strings.Contains(err.Error(), configs.Err_ctx_exceeded) {
				i++
				l.Schal("err", fmt.Sprintf("[RequestBatchVerify] %v", err))
				timeoutStep += 10
				time.Sleep(time.Minute * 3)
				continue
			}
			if strings.Contains(err.Error(), configs.Err_tee_Busy) {
				l.Schal("err", fmt.Sprintf("[RequestBatchVerify] %v", err))
				time.Sleep(time.Minute * 3)
				continue
			}
			l.Schal("err", fmt.Sprintf("[RequestBatchVerify] %v", err))
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

func calcChallengeFragments(cli *chain.ChainClient, fid string, start uint32) ([]string, error) {
	var err error
	var fmeta chain.FileMetadata
	for i := 0; i < 3; i++ {
		fmeta, err = cli.QueryFile(fid, int32(start))
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
			if sutils.CompareSlice(fmeta.SegmentList[i].FragmentList[j].Miner[:], cli.GetSignatureAccPulickey()) {
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

func checkTag(cli *chain.ChainClient, ws *Workspace, teeRecord *TeeRecord, cfg *confile.Confile, l logger.Logger, fid, fragment string) (TagfileType, error) {
	serviceTagPath := filepath.Join(ws.GetFileDir(), fid, fragment+".tag")
	fragmentPath := filepath.Join(ws.GetFileDir(), fid, fragment)
	buf, err := os.ReadFile(serviceTagPath)
	if err != nil {
		err = calcFragmentTag(cli, l, teeRecord, ws, cfg, fid, fragmentPath)
		if err != nil {
			l.Schal("err", fmt.Sprintf("calc the fragment tag failed: %v", err))
			cli.GenerateRestoralOrder(fid, fragment)
			return TagfileType{}, err
		}
	}
	var tag = TagfileType{}
	err = json.Unmarshal(buf, &tag)
	if err != nil {
		l.Schal("err", fmt.Sprintf("invalid tag file: %v", err))
		os.Remove(serviceTagPath)
		l.Del("info", serviceTagPath)
		err = calcFragmentTag(cli, l, teeRecord, ws, cfg, fid, fragmentPath)
		if err != nil {
			l.Schal("err", fmt.Sprintf("calc the fragment tag failed: %v", err))
			cli.GenerateRestoralOrder(fid, fragment)
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
