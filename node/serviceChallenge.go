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
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type serviceProofInfo struct {
	Names                 []string `json:"names"`
	Us                    []string `json:"us"`
	Mus                   []string `json:"mus"`
	Usig                  [][]byte `json:"usig"`
	ServiceBloomFilter    []uint64 `json:"serviceBloomFilter"`
	TeeAccountId          []byte   `json:"teeAccountId"`
	Signature             []byte   `json:"signature"`
	AllocatedTeeAccountId []byte   `json:"allocatedTeeAccountId"`
	AllocatedTeeAccount   string   `json:"allocatedTeeAccount"`
	Sigma                 string   `json:"sigma"`
	Start                 uint32   `json:"start"`
	ServiceResult         bool     `json:"serviceResult"`
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
	teeAcc types.AccountID,
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
	err := n.checkServiceProofRecord(serviceProofSubmited, challStart, randomIndexList, randomList, teeAcc)
	if err == nil {
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

	time.Sleep(pattern.BlockInterval * 2)

	_, chall, err := n.QueryChallengeInfo(n.GetSignatureAccPulickey())
	if err != nil {
		n.Schal("err", err.Error())
		return
	}
	ok := chall.ProveInfo.ServiceProve.HasValue()
	if ok {
		_, sProve := chall.ProveInfo.ServiceProve.Unwrap()
		serviceProofRecord.AllocatedTeeAccount, _ = sutils.EncodePublicKeyAsCessAccount(sProve.TeeAcc[:])
		serviceProofRecord.AllocatedTeeAccountId = sProve.TeeAcc[:]
	} else {
		return
	}

	n.saveServiceProofRecord(serviceProofRecord)

	teeInfo, err := n.GetTee(serviceProofRecord.AllocatedTeeAccount)
	if err != nil {
		n.Schal("info", fmt.Sprintf("[%s] Not found tee", serviceProofRecord.AllocatedTeeAccount))
		return
	}

	serviceProofRecord.ServiceBloomFilter,
		serviceProofRecord.TeeAccountId,
		serviceProofRecord.Signature,
		serviceProofRecord.ServiceResult, err = n.batchVerify(randomIndexList, randomList, teeInfo.EndPoint, serviceProofRecord)
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

	txhash, err = n.SubmitServiceProofResult(
		types.Bool(serviceProofRecord.ServiceResult),
		signature,
		bloomFilter,
		serviceProofRecord.AllocatedTeeAccountId,
	)
	if err != nil {
		n.Schal("err", fmt.Sprintf("[SubmitServiceProofResult] hash: %s, err: %v", txhash, err))
		return
	}
	n.Schal("info", fmt.Sprintf("submit service aggr proof result suc: %s", txhash))
	return
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
	var recover bool
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
		fragments, err := utils.DirFiles(serviceRoothashDir[i], 0)
		if err != nil {
			n.Schal("err", fmt.Sprintf("DirFiles(%s) %v", serviceRoothashDir[i], err))
			return names, us, mus, sigma, usig, err
		}
		for j := 0; j < len(fragments); j++ {
			recover = false
			isChall = true
			fragmentHash = filepath.Base(fragments[j])
			ok, err = n.Has([]byte(Cach_prefix_Tag + fragmentHash))
			if err != nil {
				n.Schal("err", fmt.Sprintf("Cache.Has(%s.%s): %v", roothash, fragmentHash, err))
				return names, us, mus, sigma, usig, err
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
									ok, block := fragment.Tag.Unwrap()
									if !ok {
										n.Schal("err", fmt.Sprintf("fragment.Tag.Unwrap(%s.%s): %v", roothash, fragmentHash, err))
										return names, us, mus, sigma, usig, err
									}
									err = n.Put([]byte(Cach_prefix_Tag+fragmentHash), []byte(fmt.Sprintf("%d", block)))
									if err != nil {
										n.Schal("err", fmt.Sprintf("Cache.Put(%s.%s)(%s): %v", roothash, fragmentHash, fmt.Sprintf("%d", block), err))
									}
									if uint32(block) > challStart {
										isChall = false
										break
									}
								}
							}
						}
					}
					if !isChall {
						break
					}
				}
				if !isChall {
					continue
				}
			} else {
				n.Schal("info", fmt.Sprintf("calc file: %s.%s", roothash, fragmentHash))
				block, err := n.Get([]byte(Cach_prefix_Tag + fragmentHash))
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
					continue
				}
			}
			serviceTagPath := filepath.Join(n.DataDir.TagDir, fmt.Sprintf("%s.tag", fragmentHash))
			buf, err := os.ReadFile(serviceTagPath)
			if err != nil {
				if strings.Contains(err.Error(), configs.Err_file_not_fount) {
					recover = true
				} else {
					n.Schal("err", fmt.Sprintf("[%s] ReadFile(%s): %v", roothash, fmt.Sprintf("%s.tag", fragmentHash), err))
					return names, us, mus, sigma, usig, err
				}
			}
			if len(buf) < pattern.FragmentSize {
				recover = true
				n.Schal("err", fmt.Sprintf("[%s.%s] File fragment size [%d] is not equal to %d", roothash, fragmentHash, len(buf), pattern.FragmentSize))
			}
			if recover {
				buf, err = n.GetFragmentFromOss(fragmentHash)
				if err != nil {
					n.Schal("err", fmt.Sprintf("Recovering fragment from cess gateway failed: %v", err))
					return names, us, mus, sigma, usig, err
				}
				if len(buf) < pattern.FragmentSize {
					n.Schal("err", fmt.Sprintf("[%s.%s] Fragment size [%d] received from CESS gateway is wrong", roothash, fragmentHash, len(buf)))
					return names, us, mus, sigma, usig, err
				}
				err = os.WriteFile(serviceTagPath, buf, os.ModePerm)
				if err != nil {
					n.Schal("err", fmt.Sprintf("[%s] [WriteFile(%s)]: %v", roothash, fragmentHash, err))
					return names, us, mus, sigma, usig, err
				}
			}
			var tag pb.ResponseGenTag
			err = json.Unmarshal(buf, &tag)
			if err != nil {
				n.Schal("err", fmt.Sprintf("Unmarshal %v err: %v", serviceTagPath, err))
				return names, us, mus, sigma, usig, err
			}
			matrix, _, err := proof.SplitByN(filepath.Join(roothash, fragmentHash), int64(len(tag.Tag.T.Phi)))
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
			break
		}
	}
	return names, us, mus, sigma, usig, nil
}

func (n *Node) checkServiceProofRecord(
	serviceProofSubmited bool,
	challStart uint32,
	randomIndexList []types.U32,
	randomList []pattern.Random,
	teeAcc types.AccountID,
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
		time.Sleep(pattern.BlockInterval * 2)
		_, chall, err := n.QueryChallengeInfo(n.GetSignatureAccPulickey())
		if err != nil {
			return err
		}
		ok := chall.ProveInfo.ServiceProve.HasValue()
		if ok {
			_, sProve := chall.ProveInfo.ServiceProve.Unwrap()
			serviceProofRecord.AllocatedTeeAccount, _ = sutils.EncodePublicKeyAsCessAccount(sProve.TeeAcc[:])
			serviceProofRecord.AllocatedTeeAccountId = sProve.TeeAcc[:]
		} else {
			return errors.New("chall.ProveInfo.ServiceProve is empty")
		}
	} else {
		serviceProofRecord.AllocatedTeeAccount, err = sutils.EncodePublicKeyAsCessAccount(teeAcc[:])
		if err != nil {
			_, chall, err := n.QueryChallengeInfo(n.GetSignatureAccPulickey())
			if err != nil {
				return err
			}
			ok := chall.ProveInfo.ServiceProve.HasValue()
			if ok {
				_, sProve := chall.ProveInfo.ServiceProve.Unwrap()
				serviceProofRecord.AllocatedTeeAccount, _ = sutils.EncodePublicKeyAsCessAccount(sProve.TeeAcc[:])
				serviceProofRecord.AllocatedTeeAccountId = sProve.TeeAcc[:]
			} else {
				return errors.New("chall.ProveInfo.ServiceProve is empty")
			}
		} else {
			serviceProofRecord.AllocatedTeeAccountId = teeAcc[:]
		}
	}

	for {
		if serviceProofRecord.ServiceBloomFilter != nil &&
			serviceProofRecord.TeeAccountId != nil &&
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

	teeInfo, err := n.GetTee(serviceProofRecord.AllocatedTeeAccount)
	if err != nil {
		n.Schal("err", err.Error())
		return err
	}

	serviceProofRecord.ServiceBloomFilter,
		serviceProofRecord.TeeAccountId,
		serviceProofRecord.Signature,
		serviceProofRecord.ServiceResult, err = n.batchVerify(randomIndexList, randomList, teeInfo.EndPoint, serviceProofRecord)
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

	var qslice_pb = &pb.RequestBatchVerify_Qslice{
		RandomIndexList: randomIndexList_pb,
		RandomList:      randomList_pb,
	}

	peeridSign, err := n.Sign(n.GetPeerPublickey())
	if err != nil {
		n.Schal("err", fmt.Sprintf("[Sign] %v", err))
		return nil, nil, nil, false, err
	}

	var batchVerifyParam = &pb.RequestBatchVerify_BatchVerifyParam{
		Names: serviceProofRecord.Names,
		Us:    serviceProofRecord.Us,
		Mus:   serviceProofRecord.Mus,
		Sigma: serviceProofRecord.Sigma,
	}
	var batchVerify *pb.ResponseBatchVerify
	var timeoutStep time.Duration = 10
	var timeout time.Duration
	var requestBatchVerify = &pb.RequestBatchVerify{
		AggProof:        batchVerifyParam,
		PeerId:          n.GetPeerPublickey(),
		MinerPbk:        n.GetSignatureAccPulickey(),
		MinerPeerIdSign: peeridSign,
		Qslices:         qslice_pb,
		//USig: ,
	}
	var dialOptions []grpc.DialOption
	if !strings.Contains(teeEndPoint, "https://") {
		dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	} else {
		dialOptions = nil
	}
	n.Schal("info", fmt.Sprintf("req tee batch verify: %s", teeEndPoint))
	for i := 0; i < 3; i++ {
		timeout = time.Minute * timeoutStep
		batchVerify, err = n.RequestBatchVerify(
			teeEndPoint,
			requestBatchVerify,
			timeout,
			dialOptions,
			nil,
		)
		if err != nil {
			if strings.Contains(err.Error(), configs.Err_ctx_exceeded) {
				n.Schal("err", fmt.Sprintf("[RequestBatchVerify] %v", err))
				timeoutStep += 10
				time.Sleep(time.Minute)
				continue
			}
			n.Schal("err", fmt.Sprintf("[RequestBatchVerify] %v", err))
			return nil, nil, nil, false, err
		}
		return batchVerify.ServiceBloomFilter, batchVerify.TeeAccountId, batchVerify.Signature, batchVerify.BatchVerifyResult, err
	}
	return nil, nil, nil, false, err
}
