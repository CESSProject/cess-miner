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
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type serviceProofInfo struct {
	Names               []string                `json:"names"`
	Us                  []string                `json:"us"`
	Mus                 []string                `json:"mus"`
	Usig                [][]byte                `json:"usig"`
	ServiceBloomFilter  []uint64                `json:"serviceBloomFilter"`
	Signature           []byte                  `json:"signature"`
	AllocatedTeeWorkpuk pattern.WorkerPublicKey `json:"allocatedTeeWorkpuk"`
	Sigma               string                  `json:"sigma"`
	Start               uint32                  `json:"start"`
	ServiceResult       bool                    `json:"serviceResult"`
}

type RandomList struct {
	Index  []uint32 `json:"index"`
	Random [][]byte `json:"random"`
}

func (n *Node) serviceChallenge(
	ch chan<- bool,
	serviceProofSubmited bool,
	latestBlock,
	challVerifyExpiration uint32,
	challStart uint32,
	randomIndexList []types.U32,
	randomList []pattern.Random,
	teePubkey pattern.WorkerPublicKey,
) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	if challVerifyExpiration <= latestBlock {
		n.Schal("err", fmt.Sprintf("%d < %d", challVerifyExpiration, latestBlock))
		return
	}

	var serviceProofRecord serviceProofInfo
	err := n.checkServiceProofRecord(serviceProofSubmited, challStart, randomIndexList, randomList, teePubkey)
	if err == nil {
		return
	}
	if serviceProofSubmited {
		return
	}

	n.Schal("info", fmt.Sprintf("Service file chain challenge: %v", challStart))

	var qslice = make([]proof.QElement, len(randomIndexList))
	for k, v := range randomIndexList {
		qslice[k].I = int64(v)
		var b = make([]byte, pattern.RandomLen)
		for i := 0; i < pattern.RandomLen; i++ {
			b[i] = byte(randomList[k][i])
		}
		qslice[k].V = new(big.Int).SetBytes(b).String()
	}

	err = n.saveRandom(challStart, randomIndexList, randomList)
	if err != nil {
		n.Schal("err", fmt.Sprintf("Save service file challenge random err: %v", err))
	}

	serviceProofRecord = serviceProofInfo{}
	serviceProofRecord.Start = uint32(challStart)
	serviceProofRecord.Names,
		serviceProofRecord.Us,
		serviceProofRecord.Mus,
		serviceProofRecord.Sigma,
		serviceProofRecord.Usig, err = n.calcSigma(challStart, randomIndexList, randomList)
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

	time.Sleep(pattern.BlockInterval * 3)

	_, chall, err := n.QueryChallengeInfo(n.GetSignatureAccPulickey())
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

	n.saveServiceProofRecord(serviceProofRecord)
	var endpoint string
	teeInfo, err := n.GetTee(string(serviceProofRecord.AllocatedTeeWorkpuk[:]))
	if err != nil {
		n.Schal("err", err.Error())
		endpoint, err = n.QueryTeeWorkEndpoint(serviceProofRecord.AllocatedTeeWorkpuk)
		if err != nil {
			n.Schal("err", err.Error())
			return
		}
		endpoint = processEndpoint(endpoint)
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
	if len(teeWorkpuk) != pattern.WorkerPublicKeyLen {
		n.Schal("err", fmt.Sprintf("Invalid tee work public key from tee returned: %v", len(teeWorkpuk)))
		return
	}
	for i := 0; i < pattern.WorkerPublicKeyLen; i++ {
		serviceProofRecord.AllocatedTeeWorkpuk[i] = types.U8(teeWorkpuk[i])
	}
	n.Schal("info", fmt.Sprintf("Batch verification results of service files: %v", serviceProofRecord.ServiceResult))

	var signature pattern.TeeSig
	if len(serviceProofRecord.Signature) != pattern.TeeSigLen {
		n.Schal("err", "invalid batchVerify.Signature")
		return
	}
	for i := 0; i < pattern.TeeSigLen; i++ {
		signature[i] = types.U8(serviceProofRecord.Signature[i])
	}

	var bloomFilter pattern.BloomFilter
	if len(serviceProofRecord.ServiceBloomFilter) != pattern.BloomFilterLen {
		n.Schal("err", "invalid batchVerify.ServiceBloomFilter")
		return
	}
	for i := 0; i < pattern.BloomFilterLen; i++ {
		bloomFilter[i] = types.U64(serviceProofRecord.ServiceBloomFilter[i])
	}

	n.saveServiceProofRecord(serviceProofRecord)
	var teeSignBytes = make(types.Bytes, len(signature))
	for j := 0; j < len(signature); j++ {
		teeSignBytes[j] = byte(signature[j])
	}
	for i := 2; i < 10; i++ {
		txhash, err = n.SubmitServiceProofResult(
			types.Bool(serviceProofRecord.ServiceResult),
			teeSignBytes,
			bloomFilter,
			serviceProofRecord.AllocatedTeeWorkpuk,
		)
		if err != nil {
			n.Schal("err", fmt.Sprintf("[SubmitServiceProofResult] hash: %s, err: %v", txhash, err))
			time.Sleep(time.Minute * time.Duration(i))
			continue
		}
		n.Schal("info", fmt.Sprintf("submit service aggr proof result suc: %s", txhash))
		return
	}
}

// save challenge random number
func (n *Node) saveRandom(
	challStart uint32,
	randomIndexList []types.U32,
	randomList []pattern.Random,
) error {
	randfilePath := filepath.Join(n.DataDir.RandomDir, fmt.Sprintf("random.%d", challStart))
	fstat, err := os.Stat(randfilePath)
	if err == nil && fstat.Size() > 0 {
		return nil
	}
	var rd RandomList
	rd.Index = make([]uint32, len(randomIndexList))
	rd.Random = make([][]byte, len(randomIndexList))
	for i := 0; i < len(randomIndexList); i++ {
		rd.Index[i] = uint32(randomIndexList[i])
		rd.Random[i] = make([]byte, len(randomList[i]))
		for j := 0; j < len(randomList[i]); j++ {
			rd.Random[i][j] = byte(randomList[i][j])
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
func (n *Node) calcSigma(
	challStart uint32,
	randomIndexList []types.U32,
	randomList []pattern.Random,
) ([]string, []string, []string, string, [][]byte, error) {
	var ok bool
	var isChall bool
	var sigma string
	var roothash string
	var fragmentHash string
	var proveResponse proof.GenProofResponse
	var names = make([]string, 0)
	var us = make([]string, 0)
	var mus = make([]string, 0)
	var usig = make([][]byte, 0)
	var qslice = make([]proof.QElement, len(randomIndexList))
	for k, v := range randomIndexList {
		qslice[k].I = int64(v)
		var b = make([]byte, len(randomList[k]))
		for i := 0; i < len(randomList[k]); i++ {
			b[i] = byte(randomList[k][i])
		}
		qslice[k].V = new(big.Int).SetBytes(b).String()
	}

	serviceRoothashDir, err := utils.Dirs(n.GetDirs().FileDir)
	if err != nil {
		n.Schal("err", fmt.Sprintf("[Dirs] %v", err))
		return names, us, mus, sigma, usig, err
	}

	timeout := time.NewTicker(time.Duration(time.Minute))
	defer timeout.Stop()

	for i := int(0); i < len(serviceRoothashDir); i++ {
		roothash = filepath.Base(serviceRoothashDir[i])
		n.Schal("info", fmt.Sprintf("will calc %s", roothash))
		_, err = n.QueryFileMetadata(roothash)
		if err != nil {
			if err.Error() == pattern.ERR_Empty {
				n.Schal("info", fmt.Sprintf("QueryFileMetadata(%s) is empty", roothash))
				continue
			}
		}

		fragments, err := utils.DirFiles(serviceRoothashDir[i], 0)
		if err != nil {
			n.Schal("err", fmt.Sprintf("DirFiles(%s) %v", serviceRoothashDir[i], err))
			return names, us, mus, sigma, usig, err
		}
		for j := 0; j < len(fragments); j++ {
			isChall = false
			fragmentHash = filepath.Base(fragments[j])
			if strings.Contains(fragmentHash, ".tag") {
				continue
			}
			ok, err = n.Has([]byte(Cach_prefix_Tag + roothash + "." + fragmentHash))
			if err != nil {
				n.Schal("err", fmt.Sprintf("Cache.Has(%s.%s): %v", roothash, fragmentHash, err))
			}
			if !ok {
				n.Schal("err", fmt.Sprintf("Cache.NotFound(%s.%s)", roothash, fragmentHash))
				fmeta, err := n.QueryFileMetadata(roothash)
				if err != nil {
					if !strings.Contains(err.Error(), pattern.ERR_Empty) {
						n.Schal("err", fmt.Sprintf("QueryFileMetadata(%s): %v", roothash, err))
						return names, us, mus, sigma, usig, err
					}
					continue
				}
				for _, segment := range fmeta.SegmentList {
					for _, fragment := range segment.FragmentList {
						if sutils.CompareSlice(fragment.Miner[:], n.GetSignatureAccPulickey()) {
							if fragmentHash == string(fragment.Hash[:]) {
								if fragment.Tag.HasValue() {
									isChall = true
									ok, block := fragment.Tag.Unwrap()
									if !ok {
										n.Schal("err", fmt.Sprintf("fragment.Tag.Unwrap(%s.%s): %v", roothash, fragmentHash, err))
										return names, us, mus, sigma, usig, err
									}
									err = n.Put([]byte(Cach_prefix_Tag+roothash+"."+fragmentHash), []byte(fmt.Sprintf("%d", block)))
									if err != nil {
										n.Schal("err", fmt.Sprintf("Cache.Put(%s.%s)(%s): %v", roothash, fragmentHash, fmt.Sprintf("%d", block), err))
									}
									if uint32(block) > challStart {
										isChall = false
										break
									}
								} else {
									isChall = false
									break
								}
							}
						}
					}
					if isChall {
						break
					}
				}
				if !isChall {
					n.Del("info", fragments[j])
					os.Remove(fragments[j])
					continue
				}
				n.Schal("info", fmt.Sprintf("chall go on: %s.%s", roothash, fragmentHash))
			} else {
				n.Schal("info", fmt.Sprintf("calc file: %s.%s", roothash, fragmentHash))
				block, err := n.Get([]byte(Cach_prefix_Tag + roothash + "." + fragmentHash))
				if err != nil {
					n.Schal("err", fmt.Sprintf("Cache.Get(%s.%s): %v", roothash, fragmentHash, err))
					return names, us, mus, sigma, usig, err
				}
				blocknumber, err := strconv.ParseUint(string(block), 10, 32)
				if err != nil {
					n.Schal("err", fmt.Sprintf("ParseUint(%s): %v", string(block), err))
					return names, us, mus, sigma, usig, err
				}
				if blocknumber > uint64(challStart) {
					n.Schal("info", fmt.Sprintf("Not at chall: %d > %d", blocknumber, challStart))
					continue
				}
			}
			serviceTagPath := fmt.Sprintf("%s.tag", fragments[j])
			buf, err := os.ReadFile(serviceTagPath)
			if err != nil {
				err = n.calcFragmentTag(roothash, fragments[j])
				if err != nil {
					n.Schal("err", fmt.Sprintf("calcFragmentTag %v err: %v", fragments[j], err))
					n.GenerateRestoralOrder(roothash, fragmentHash)
					continue
				}
			}
			n.Schal("info", fmt.Sprintf("[%s] Read tag file: %s", roothash, serviceTagPath))
			var tag = &TagfileType{}
			err = json.Unmarshal(buf, tag)
			if err != nil {
				n.Schal("err", fmt.Sprintf("Unmarshal %v err: %v", serviceTagPath, err))
				os.Remove(serviceTagPath)
				n.Del("info", serviceTagPath)
				n.GenerateRestoralOrder(roothash, fragmentHash)
				continue
			}
			_, err = os.Stat(fragments[j])
			if err != nil {
				buf, err = n.GetFragmentFromOss(fragmentHash)
				if err != nil {
					n.Schal("err", fmt.Sprintf("Recovering fragment from cess gateway failed: %v", err))
					return names, us, mus, sigma, usig, err
				}
				if len(buf) < pattern.FragmentSize {
					n.Schal("err", fmt.Sprintf("[%s.%s] Fragment size [%d] received from CESS gateway is wrong", roothash, fragmentHash, len(buf)))
					return names, us, mus, sigma, usig, err
				}
				err = os.WriteFile(fragments[j], buf, os.ModePerm)
				if err != nil {
					n.Schal("err", fmt.Sprintf("[%s] [WriteFile(%s)]: %v", roothash, fragmentHash, err))
					return names, us, mus, sigma, usig, err
				}
			}
			matrix, _, err := proof.SplitByN(fragments[j], int64(len(tag.Tag.T.Phi)))
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

			if proveResponse.StatueMsg.StatusCode != proof.Success {
				n.Schal("err", fmt.Sprintf("GenProof  err: %d", proveResponse.StatueMsg.StatusCode))
				return names, us, mus, sigma, usig, err
			}

			sigmaTemp, ok := n.AggrAppendProof(sigma, proveResponse.Sigma)
			if !ok {
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
	randomList []pattern.Random,
	teePubkey pattern.WorkerPublicKey,
) error {
	var serviceProofRecord serviceProofInfo
	buf, err := os.ReadFile(filepath.Join(n.Workspace(), configs.ServiceProofFile))
	if err != nil {
		return err
	}

	err = json.Unmarshal(buf, &serviceProofRecord)
	if err != nil {
		return err
	}

	if serviceProofRecord.Start != challStart {
		os.Remove(filepath.Join(n.Workspace(), configs.ServiceProofFile))
		n.Del("info", filepath.Join(n.Workspace(), configs.ServiceProofFile))
		return errors.New("Local service file challenge record is outdated")
	}

	n.Schal("info", fmt.Sprintf("local service proof file challenge: %v", serviceProofRecord.Start))

	if !serviceProofSubmited {
		if serviceProofRecord.Names == nil ||
			serviceProofRecord.Us == nil ||
			serviceProofRecord.Mus == nil {
			serviceProofRecord.Names,
				serviceProofRecord.Us,
				serviceProofRecord.Mus,
				serviceProofRecord.Sigma,
				serviceProofRecord.Usig, err = n.calcSigma(challStart, randomIndexList, randomList)
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
		time.Sleep(pattern.BlockInterval * 3)
		_, chall, err := n.QueryChallengeInfo(n.GetSignatureAccPulickey())
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
		if sutils.IsWorkerPublicKeyAllZero(teePubkey) {
			_, chall, err := n.QueryChallengeInfo(n.GetSignatureAccPulickey())
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

	for {
		if serviceProofRecord.ServiceBloomFilter != nil &&
			serviceProofRecord.Signature != nil {
			if len(serviceProofRecord.Signature) != pattern.TeeSigLen {
				n.Schal("err", "invalid batchVerify.Signature")
				break
			}
			var bloomFilter pattern.BloomFilter
			if len(serviceProofRecord.ServiceBloomFilter) != pattern.BloomFilterLen {
				n.Schal("err", "invalid batchVerify.ServiceBloomFilter")
				break
			}
			for i := 0; i < pattern.BloomFilterLen; i++ {
				bloomFilter[i] = types.U64(serviceProofRecord.ServiceBloomFilter[i])
			}
			txhash, err := n.SubmitServiceProofResult(
				types.Bool(serviceProofRecord.ServiceResult),
				serviceProofRecord.Signature[:],
				bloomFilter,
				serviceProofRecord.AllocatedTeeWorkpuk,
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
	var endpoint string
	teeInfo, err := n.GetTee(string(serviceProofRecord.AllocatedTeeWorkpuk[:]))
	if err != nil {
		n.Schal("err", err.Error())
		endpoint, err = n.QueryTeeWorkEndpoint(serviceProofRecord.AllocatedTeeWorkpuk)
		if err != nil {
			n.Schal("err", err.Error())
			return err
		}
		endpoint = processEndpoint(endpoint)
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
	if len(teeWorkpuk) != pattern.WorkerPublicKeyLen {
		n.Schal("err", fmt.Sprintf("Invalid tee work public key from tee returned: %v", len(teeWorkpuk)))
		return nil
	}
	for i := 0; i < pattern.WorkerPublicKeyLen; i++ {
		serviceProofRecord.AllocatedTeeWorkpuk[i] = types.U8(teeWorkpuk[i])
	}
	n.Schal("info", fmt.Sprintf("Batch verification results of service files: %v", serviceProofRecord.ServiceResult))
	if len(serviceProofRecord.Signature) != pattern.TeeSigLen {
		n.Schal("err", "invalid batchVerify.Signature")
		return nil
	}
	var bloomFilter pattern.BloomFilter
	if len(serviceProofRecord.ServiceBloomFilter) != pattern.BloomFilterLen {
		n.Schal("err", "invalid batchVerify.ServiceBloomFilter")
		return nil
	}
	for i := 0; i < pattern.BloomFilterLen; i++ {
		bloomFilter[i] = types.U64(serviceProofRecord.ServiceBloomFilter[i])
	}
	n.saveServiceProofRecord(serviceProofRecord)
	txhash, err := n.SubmitServiceProofResult(
		types.Bool(serviceProofRecord.ServiceResult),
		serviceProofRecord.Signature[:],
		bloomFilter,
		serviceProofRecord.AllocatedTeeWorkpuk,
	)
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
		err = sutils.WriteBufToFile(buf, filepath.Join(n.Workspace(), configs.ServiceProofFile))
		if err != nil {
			n.Schal("err", err.Error())
		}
	}
}

func (n *Node) batchVerify(
	randomIndexList []types.U32,
	randomList []pattern.Random,
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
		batchVerifyResult, err = n.RequestBatchVerify(
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

func encodeToRequestBatchVerify_Qslice(randomIndexList []types.U32, randomList []pattern.Random) *pb.RequestBatchVerify_Qslice {
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
