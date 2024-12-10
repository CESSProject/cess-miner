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
	challStart uint32,
	randomIndexList []types.U32,
	randomList []chain.Random,
) {
	defer func() {
		ch <- true
		n.SetServiceChallenging(false)
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()
	var err error

	if serviceProofSubmited {
		err = n.checkServiceProofRecord(challStart)
		if err == nil {
			return
		}
	}

	var serviceProofRecord common.ServiceProofInfo
	serviceProofRecord.Start = challStart
	serviceProofRecord.CanSubmitProof = true
	serviceProofRecord.CanSubmitResult = true

	n.SetServiceChallenging(true)

	n.Schal("info", fmt.Sprintf("Service file chain challenge: %v", challStart))

	err = n.SaveChallRandom(challStart, randomIndexList, randomList)
	if err != nil {
		n.Schal("err", fmt.Sprintf("Save service file challenge random err: %v", err))
	}

	teePuk, teeSign, bloomFilter, proof, err := n.batchGenProofAndVerify(challStart, randomIndexList, randomList)
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

	err = n.submitServiceProof(proof)
	if err != nil {
		n.Schal("err", err.Error())
		return
	}

	serviceProofRecord.Proof = proof
	serviceProofRecord.CanSubmitProof = false
	n.SaveServiceProve(serviceProofRecord)

	time.Sleep(chain.BlockInterval * 2)

	err = n.submitServiceResult(types.Bool(true), teeSign, bloomFilter, teePuk)
	if err != nil {
		n.Schal("err", err.Error())
	}

	serviceProofRecord.CanSubmitResult = false
	n.SaveServiceProve(serviceProofRecord)
}

func (n *Node) submitServiceResult(result types.Bool, sign []byte, bloomFilter chain.BloomFilter, teePuk chain.WorkerPublicKey) error {
	var (
		err       error
		blockHash string
	)
	for i := 0; i < 5; i++ {
		blockHash, err = n.SubmitVerifyServiceResult(result, sign, bloomFilter, teePuk)
		if err == nil && blockHash != "" {
			n.Schal("info", fmt.Sprintf("submit service aggr proof result suc: %s", blockHash))
			return nil
		}
		n.Schal("err", fmt.Sprintf("[SubmitServiceProofResult] hash: %s, err: %v", blockHash, err))
		time.Sleep(time.Minute)
		continue
	}
	return fmt.Errorf("submitServiceProof failed: %v", err)
}

func (n *Node) submitServiceProof(serviceProof []types.U8) error {
	var (
		err       error
		blockHash string
		challInfo chain.ChallengeInfo
	)
	for i := 0; i < 5; i++ {
		n.Schal("info", fmt.Sprintf("[start sub] %v", time.Now()))
		blockHash, err = n.SubmitServiceProof(serviceProof)
		n.Schal("info", fmt.Sprintf("[end sub]: %s", blockHash))
		if err == nil && blockHash != "" {
			return nil
		}

		time.Sleep(chain.BlockInterval * 3)

		_, challInfo, err = n.QueryChallengeSnapShot(n.GetSignatureAccPulickey(), -1)
		if err != nil {
			if err.Error() != chain.ERR_Empty {
				n.Schal("err", fmt.Sprintf("[QueryChallengeInfo] %v", err))
			}
			return err
		}

		if challInfo.ProveInfo.ServiceProve.HasValue() {
			_, serviceProve := challInfo.ProveInfo.ServiceProve.Unwrap()
			if len(serviceProve.ServiceProve) > 0 {
				return nil
			}
		}
	}
	return fmt.Errorf("submitServiceProof failed: %v", err)
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

func (n *Node) checkServiceProofRecord(challStart uint32) error {
	serviceProofRecord, err := n.LoadServiceProve()
	if err != nil {
		return err
	}

	if serviceProofRecord.Start != challStart {
		os.Remove(n.GetServiceProve())
		n.Del("info", n.GetServiceProve())
		return errors.New("Local service file challenge record is outdated")
	}

	if !serviceProofRecord.CanSubmitResult {
		return nil
	}

	n.Schal("info", fmt.Sprintf("local service proof file challenge: %v", serviceProofRecord.Start))

	if serviceProofRecord.CanSubmitProof && serviceProofRecord.SignatureHex != "" {
		teeSignBytes, err := hex.DecodeString(serviceProofRecord.SignatureHex)
		if err != nil {
			return err
		}
		for i := 0; i < 5; i++ {
			txhash, err := n.SubmitVerifyServiceResult(
				types.Bool(true),
				teeSignBytes,
				serviceProofRecord.BloomFilter,
				serviceProofRecord.TeePublicKey,
			)
			if err != nil {
				n.Schal("err", fmt.Sprintf("[SubmitServiceProofResult] hash: %s, err: %v", txhash, err))
				time.Sleep(time.Minute)
				continue
			}
			n.Schal("info", fmt.Sprintf("submit service aggr proof result suc: %s", txhash))
			break
		}
		serviceProofRecord.CanSubmitResult = false
		n.SaveServiceProve(serviceProofRecord)
		return nil
	}

	return errors.New("Service proof result not submited")
}

// func (n *Node) batchVerify(
// 	randomIndexList []types.U32,
// 	randomList []chain.Random,
// 	teeEndPoint string,
// 	serviceProofRecord common.ServiceProofInfo,
// ) ([]uint64, []byte, []byte, bool, error) {
// 	var err error
// 	qslice_pb := encodeToRequestBatchVerify_Qslice(randomIndexList, randomList)
// 	var batchVerifyParam = &pb.RequestBatchVerify_BatchVerifyParam{
// 		Names: serviceProofRecord.Names,
// 		Us:    serviceProofRecord.Us,
// 		Mus:   serviceProofRecord.Mus,
// 		Sigma: serviceProofRecord.Sigma,
// 	}
// 	var batchVerifyResult *pb.ResponseBatchVerify
// 	var timeoutStep time.Duration = 10
// 	var timeout time.Duration
// 	var requestBatchVerify = &pb.RequestBatchVerify{
// 		AggProof: batchVerifyParam,
// 		MinerId:  n.GetSignatureAccPulickey(),
// 		Qslices:  qslice_pb,
// 		USigs:    serviceProofRecord.Usig,
// 	}
// 	var dialOptions []grpc.DialOption
// 	if !strings.Contains(teeEndPoint, "443") {
// 		dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
// 	} else {
// 		dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
// 	}
// 	n.Schal("info", fmt.Sprintf("req tee batch verify: %s", teeEndPoint))
// 	n.Schal("info", fmt.Sprintf("serviceProofRecord.Names: %v", serviceProofRecord.Names))
// 	n.Schal("info", fmt.Sprintf("len(serviceProofRecord.Us): %v", len(serviceProofRecord.Us)))
// 	n.Schal("info", fmt.Sprintf("len(serviceProofRecord.Mus): %v", len(serviceProofRecord.Mus)))
// 	n.Schal("info", fmt.Sprintf("Sigma: %v", serviceProofRecord.Sigma))
// 	for i := 0; i < 5; {
// 		timeout = time.Minute * timeoutStep
// 		batchVerifyResult, err = com.RequestBatchVerify(
// 			teeEndPoint,
// 			requestBatchVerify,
// 			timeout,
// 			dialOptions,
// 			nil,
// 		)
// 		if err != nil {
// 			if strings.Contains(err.Error(), configs.Err_ctx_exceeded) {
// 				i++
// 				n.Schal("err", fmt.Sprintf("[RequestBatchVerify] %v", err))
// 				timeoutStep += 10
// 				time.Sleep(time.Minute * 3)
// 				continue
// 			}
// 			if strings.Contains(err.Error(), configs.Err_tee_Busy) {
// 				n.Schal("err", fmt.Sprintf("[RequestBatchVerify] %v", err))
// 				time.Sleep(time.Minute * 3)
// 				continue
// 			}
// 			n.Schal("err", fmt.Sprintf("[RequestBatchVerify] %v", err))
// 			return nil, nil, nil, false, err
// 		}
// 		return batchVerifyResult.ServiceBloomFilter, batchVerifyResult.TeeAccountId, batchVerifyResult.Signature, batchVerifyResult.BatchVerifyResult, err
// 	}
// 	return nil, nil, nil, false, err
// }

// func encodeToRequestBatchVerify_Qslice(randomIndexList []types.U32, randomList []chain.Random) *pb.RequestBatchVerify_Qslice {
// 	var randomIndexList_pb = make([]uint32, len(randomIndexList))
// 	for i := 0; i < len(randomIndexList); i++ {
// 		randomIndexList_pb[i] = uint32(randomIndexList[i])
// 	}
// 	var randomList_pb = make([][]byte, len(randomList))
// 	for i := 0; i < len(randomList); i++ {
// 		randomList_pb[i] = make([]byte, len(randomList[i]))
// 		for j := 0; j < len(randomList[i]); j++ {
// 			randomList_pb[i][j] = byte(randomList[i][j])
// 		}
// 	}
// 	return &pb.RequestBatchVerify_Qslice{
// 		RandomIndexList: randomIndexList_pb,
// 		RandomList:      randomList_pb,
// 	}
// }

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

func (n *Node) batchGenProofAndVerify(challStart uint32, randomIndexList []types.U32, randomList []chain.Random) (chain.WorkerPublicKey, []byte, chain.BloomFilter, []types.U8, error) {
	var ok bool
	var sigma string
	var sigmaOnChian string
	var fid string
	var fragmentPath string
	var serviceTagPath string
	var proveResponse GenProofResponse
	var names = make([]string, 0)
	var us = make([]string, 0)
	var mus = make([]string, 0)
	var usig = make([][]byte, 0)
	var verifyInServiceFileStructureList = make([]*pb.RequestAggregateSignature_VerifyInServiceFileStructure, 0)

	qElement := calcQSlice(randomIndexList, randomList)
	qSlicePb := calcQSlicePb(randomIndexList, randomList)

	serviceRoothashDir, err := utils.Dirs(n.GetFileDir())
	if err != nil {
		n.Schal("err", fmt.Sprintf("[Dirs] %v", err))
		return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, nil, err
	}

	var stackedBloomFilters = make([]uint64, 0)

	timeout := time.NewTicker(time.Duration(time.Minute))
	defer timeout.Stop()

	totalFile := 0
	index := 1
	for i := int(0); i < len(serviceRoothashDir); i++ {
		fid = filepath.Base(serviceRoothashDir[i])

		n.Schal("info", fmt.Sprintf("check the file: %s", fid))

		fragments, err := n.calcChallengeFragments(fid, challStart)
		if err != nil {
			n.Schal("err", fmt.Sprintf("calcChallengeFragments(%s): %v", fid, err))
			return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, nil, err
		}

		n.Schal("info", fmt.Sprintf("number of challenged fragments: %v", len(fragments)))

		for j := 0; j < len(fragments); j++ {
			fragmentPath = filepath.Join(n.GetFileDir(), fid, fragments[j])
			serviceTagPath = filepath.Join(n.GetFileDir(), fid, fragments[j]+".tag")
			tag, err := n.checkTag(fid, fragments[j])
			if err != nil {
				n.Schal("err", fmt.Sprintf("checkTag: %v", err))
				n.GenerateRestoralOrder(fid, fragments[j])
				return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, nil, fmt.Errorf("This challenge has failed due to an invalid tag: %s", fragments[j])
			}

			_, err = os.Stat(fragmentPath)
			if err != nil {
				n.Schal("err", err.Error())
				n.GenerateRestoralOrder(fid, fragments[j])
				return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, nil, fmt.Errorf("This challenge has failed due to missing fragment: %s", fragments[j])
			}

			matrix, _, err := SplitByN(fragmentPath, int64(len(tag.Tag.T.Phi)))
			if err != nil {
				n.Schal("err", fmt.Sprintf("SplitByN %v err: %v", serviceTagPath, err))
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

			if index%2 == 0 {
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

				batchVerifyResponse, err := n.requestBatchVerify(request)
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

		batchVerifyResponse, err := n.requestBatchVerify(request)
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

	if totalFile > 0 {
		request := &pb.RequestAggregateSignature{
			VerifyInserviceFileHistory: verifyInServiceFileStructureList,
			Qslices:                    &qSlicePb,
		}

		//fmt.Println(hex.EncodeToString(request.VerifyInserviceFileHistory[0].Signature))

		aggregateSignatureResponse, err := n.requestAggregateSignature(request)
		if err != nil {
			return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, nil, err
		}

		if len(aggregateSignatureResponse.TeeAccountId) != chain.WorkerPublicKeyLen {
			return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, nil, fmt.Errorf("The length of the tee publickey returned by tee is illegal: %d != %d", len(aggregateSignatureResponse.TeeAccountId), chain.WorkerPublicKeyLen)
		}

		if len(aggregateSignatureResponse.Signature) != chain.TeeSigLen {
			return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, nil, fmt.Errorf("The length of the signature returned by tee is illegal: %d != %d", len(aggregateSignatureResponse.Signature), chain.TeeSigLen)
		}

		n.Schal("info", fmt.Sprintf("Batch verification results of service files: %v", true))

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

	var request = &pb.RequestBatchVerify{
		AggProof: &pb.RequestBatchVerify_BatchVerifyParam{
			Names: names,
			Us:    us,
			Mus:   mus,
			Sigma: sigmaOnChian,
		},
		Qslices: &qSlicePb,
		USigs:   usig,
		MinerId: n.GetSignatureAccPulickey(),
	}

	batchVerifyResponse, err := n.requestBatchVerify(request)
	if err != nil {
		return chain.WorkerPublicKey{}, nil, chain.BloomFilter{}, []types.U8{}, err
	}

	var teePuk chain.WorkerPublicKey
	for i := 0; i < chain.WorkerPublicKeyLen; i++ {
		teePuk[i] = types.U8(batchVerifyResponse.TeeAccountId[i])
	}

	return teePuk, batchVerifyResponse.Signature, chain.BloomFilter{}, []types.U8{}, nil
}

func (n *Node) requestBatchVerify(request *pb.RequestBatchVerify) (*pb.ResponseBatchVerify, error) {
	var dialOptions []grpc.DialOption
	tees := n.GetAllVerifierTeeEndpoint()
	for i := 0; i < len(tees); i++ {
		if !strings.Contains(tees[i], "443") {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		} else {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
		}
		batchVerifyResponse, err := com.RequestBatchVerify(tees[i], request, time.Minute*10, dialOptions, nil)
		if err != nil {
			continue
		}
		return batchVerifyResponse, nil
	}

	return nil, errors.New("RequestBatchVerify failed")
}

func (n *Node) requestAggregateSignature(request *pb.RequestAggregateSignature) (*pb.ResponseAggregateSignature, error) {
	var dialOptions []grpc.DialOption
	tees := n.GetAllVerifierTeeEndpoint()
	for i := 0; i < len(tees); i++ {
		if !strings.Contains(tees[i], "443") {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		} else {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
		}
		responseAggregateSignature, err := com.RequestAggregateSignature(tees[i], request, time.Minute*10, dialOptions, nil)
		if err != nil {
			continue
		}
		return responseAggregateSignature, nil
	}

	return nil, errors.New("RequestAggregateSignature failed")
}
