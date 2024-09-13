/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/node/common"
	"github.com/CESSProject/cess-miner/node/logger"
	"github.com/CESSProject/cess-miner/node/record"
	"github.com/CESSProject/cess-miner/node/runstatus"
	"github.com/CESSProject/cess-miner/node/web"
	"github.com/CESSProject/cess-miner/pkg/cache"
	"github.com/CESSProject/cess-miner/pkg/com"
	"github.com/CESSProject/cess-miner/pkg/com/pb"
	out "github.com/CESSProject/cess-miner/pkg/fout"
	"github.com/CESSProject/cess-miner/pkg/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	sdkgo "github.com/CESSProject/cess-go-sdk"
	"github.com/CESSProject/cess-go-sdk/chain"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
)

func (n *Node) InitNode() *Node {
	n.InitChainClient()
	n.InitWebServer(
		InitMiddlewares(),
		web.NewHandler(n.Chainer, n.Workspace, n.Runstatus, n.Logger),
	)
	n.InitLogs()
	n.InitCache()
	return n
}

func (n *Node) InitRunStatus(st types.Bytes, addr string, t string) {
	rt := runstatus.NewRunstatus()
	rt.SetPID(os.Getpid())
	rt.SetCpucores(int(n.ReadUseCpu()))
	rt.SetCurrentRpc(n.GetCurrentRpcAddr())
	rt.SetCurrentRpcst(n.GetRpcState())
	rt.SetSignAcc(n.GetSignatureAcc())
	rt.SetState(string(st))
	rt.SetComAddr(addr)
	stakingAcc := n.ReadStakingAcc()
	if stakingAcc == "" {
		rt.SetStakingAcc(n.GetSignatureAcc())
	} else {
		rt.SetStakingAcc(stakingAcc)
	}
	rt.SetEarningsAcc(n.ReadEarningsAcc())
	rt.SetLastConnectedTime(t)
	n.InitRunstatus(rt)
}

func (n *Node) InitCache() {
	cace, err := cache.NewCache(n.GetDbDir(), 0, 0, configs.NameSpaces)
	if err != nil {
		out.Err(fmt.Sprintf("[NewCache] %v", err))
		os.Exit(1)
	}
	n.InitCacher(cace)
}

func (n *Node) InitLogs() {
	var logs_info = make(map[string]string)
	for _, v := range logger.LogFiles {
		logs_info[v] = filepath.Join(n.GetLogDir(), v+".log")
	}
	lg, err := logger.NewLogger(logs_info)
	if err != nil {
		out.Err(fmt.Sprintf("[NewLogs] %v", err))
		os.Exit(1)
	}
	n.InitLogger(lg)
}

func (n *Node) InitChainClient() {
	addr, err := GetLocalIP()
	if err != nil {
		out.Err(fmt.Sprintf("[GetLocalIP] %v", err))
		os.Exit(1)
	}

	if !utils.FreeLocalPort(uint32(n.ReadServicePort())) {
		out.Err(fmt.Sprintf("[FreeLocalPort] %v", err))
		os.Exit(1)
	}
	addr = fmt.Sprintf("%s:%d", addr, n.ReadServicePort())

	cli, err := sdkgo.New(
		context.Background(),
		sdkgo.Name(configs.Name),
		sdkgo.ConnectRpcAddrs(n.ReadRpcEndpoints()),
		sdkgo.Mnemonic(n.ReadMnemonic()),
		sdkgo.TransactionTimeout(configs.TimeToWaitEvent),
	)
	if err != nil {
		out.Err(fmt.Sprintf("[sdkgo.New] %v", err))
		os.Exit(1)
	}
	t := time.Now().Format(time.DateTime)
	n.InitChainclient(cli)
	n.InitWorkspace(filepath.Join(n.ReadWorkspace(), n.GetSignatureAcc(), configs.Name))

	err = n.InitExtrinsicsNameForMiner()
	if err != nil {
		out.Err("The rpc address does not match the software version, please check the rpc address.")
		os.Exit(1)
	}

	err = checkRpcSynchronization(cli)
	if err != nil {
		out.Err("Failed to sync block: network error")
		os.Exit(1)
	}

	err = checkVersion(cli)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}

	n.ExpendersInfo, err = cli.QueryExpenders(-1)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}

	rsakey, poisInfo, pois, teeRecord, st, err := n.checkMiner(addr)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}

	n.InitRSAKeyPair(rsakey)
	n.InitPois(pois)
	n.InitMinerPoisInfo(poisInfo)
	n.InitTeeRecord(teeRecord)
	n.InitRunStatus(st, addr, t)
}

func (n *Node) checkMiner(addr string) (*RSAKeyPair, *pb.MinerPoisInfo, *Pois, record.TeeRecorder, types.Bytes, error) {
	var rsakey *RSAKeyPair
	var poisInfo = &pb.MinerPoisInfo{}
	var p *Pois
	var teeRecord record.TeeRecorder

	register, decTib, oldRegInfo, err := n.checkRegistrationInfo()
	if err != nil {
		return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[checkMiner]")
	}

	switch register {
	case Unregistered:
		_, err = registerMiner(n.Chainer, n.ReadStakingAcc(), n.ReadEarningsAcc(), addr, decTib)
		if err != nil {
			return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[registerMiner]")
		}

		err = n.RemoveAndBuild()
		if err != nil {
			return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[RemoveAndBuild]")
		}

		for i := 0; i < 3; i++ {
			rsakey, poisInfo, teeRecord, err = n.registerPoisKey()
			if err != nil {
				if !strings.Contains(err.Error(), "storage miner is not registered") {
					return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[registerPoisKey]")
				}
				time.Sleep(chain.BlockInterval)
				continue
			}
			break
		}
		if err != nil {
			return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[registerPoisKey]")
		}

		p, err = NewPOIS(
			n.GetPoisDir(),
			n.GetSpaceDir(),
			n.GetPoisAccDir(),
			n.ExpendersInfo,
			true, 0, 0,
			int64(n.ReadUseSpace()*1024), 32,
			int(n.ReadUseCpu()),
			poisInfo.KeyN,
			poisInfo.KeyG,
			n.GetSignatureAccPulickey(),
		)
		if err != nil {
			return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[NewPOIS]")
		}
		return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, nil

	case UnregisteredPoisKey:
		err = n.Build()
		if err != nil {
			return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[Build]")
		}
		// runtime.SetMinerState(string(oldRegInfo.State))
		for i := 0; i < 3; i++ {
			rsakey, poisInfo, teeRecord, err = n.registerPoisKey()
			if err != nil {
				if !strings.Contains(err.Error(), "storage miner is not registered") {
					return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[registerPoisKey]")
				}
				time.Sleep(chain.BlockInterval)
				continue
			}
			break
		}
		if err != nil {
			return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[registerPoisKey]")
		}

		err = updateMinerRegistertionInfo(n.Chainer, oldRegInfo, n.ReadUseSpace(), n.ReadStakingAcc(), n.ReadEarningsAcc(), addr)
		if err != nil {
			return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[updateMinerRegistertionInfo]")
		}

		p, err = NewPOIS(
			n.GetPoisDir(),
			n.GetSpaceDir(),
			n.GetPoisAccDir(),
			n.ExpendersInfo,
			true, 0, 0,
			int64(n.ReadUseSpace()*1024), 32,
			int(n.ReadUseCpu()),
			poisInfo.KeyN,
			poisInfo.KeyG,
			n.GetSignatureAccPulickey(),
		)
		if err != nil {
			return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[NewPOIS]")
		}
		return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, nil

	case Registered:
		err = n.Build()
		if err != nil {
			return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[NewPOIS]")
		}

		err = updateMinerRegistertionInfo(n.Chainer, oldRegInfo, n.ReadUseSpace(), n.ReadStakingAcc(), n.ReadEarningsAcc(), addr)
		if err != nil {
			return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[updateMinerRegistertionInfo]")
		}

		ok, spaceProofInfo := oldRegInfo.SpaceProofInfo.Unwrap()
		if !ok {
			return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.New("SpaceProofInfo unwrap failed")
		}

		poisInfo.Acc = []byte(string(spaceProofInfo.Accumulator[:]))
		poisInfo.Front = int64(spaceProofInfo.Front)
		poisInfo.Rear = int64(spaceProofInfo.Rear)
		poisInfo.KeyN = []byte(string(spaceProofInfo.PoisKey.N[:]))
		poisInfo.KeyG = []byte(string(spaceProofInfo.PoisKey.G[:]))
		poisInfo.StatusTeeSign = []byte(string(oldRegInfo.TeeSig[:]))

		p, err = NewPOIS(
			n.GetPoisDir(),
			n.GetSpaceDir(),
			n.GetPoisAccDir(),
			n.ExpendersInfo, false,
			int64(spaceProofInfo.Front),
			int64(spaceProofInfo.Rear),
			int64(n.ReadUseSpace()*1024), 32,
			int(n.ReadUseCpu()),
			poisInfo.KeyN,
			poisInfo.KeyG,
			n.GetSignatureAccPulickey(),
		)
		if err != nil {
			return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[NewPOIS]")
		}

		// teeRecord, err = n.saveAllTees()
		// if err != nil {
		// 	return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[saveAllTees]")
		// }
		var buf []byte
		buf, teeRecord, err = n.queryPodr2KeyFromTee()
		if err != nil {
			return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[queryPodr2KeyFromTee]")
		}

		rsakey, err = NewRsaKey(buf)
		if err != nil {
			return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[NewRsaKey]")
		}
		return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, nil
	}
	return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.New("system err")
}

func (n *Node) queryPodr2KeyFromTee() ([]byte, record.TeeRecorder, error) {
	var err error
	var podr2PubkeyResponse *pb.Podr2PubkeyResponse
	var dialOptions []grpc.DialOption
	var teeRecord = record.NewTeeRecord()
	delay := time.Duration(30)

	ptee := n.ReadPriorityTeeList()
	for i := 0; i < len(ptee); i++ {
		if !strings.Contains(ptee[i], "443") {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		} else {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
		}
		podr2PubkeyResponse, err = com.GetPodr2Pubkey(
			ptee[i],
			&pb.Request{StorageMinerAccountId: n.GetSignatureAccPulickey()},
			time.Duration(time.Second*delay),
			dialOptions,
			nil,
		)
		if err != nil {
			if strings.Contains(err.Error(), configs.Err_ctx_exceeded) {
				delay += 30
				continue
			}
			if strings.Contains(err.Error(), configs.Err_tee_Busy) {
				delay += 10
				continue
			}
			continue
		}
		return podr2PubkeyResponse.Pubkey, teeRecord, nil
	}

	teelist, err := n.QueryAllWorkers(-1)
	if err != nil {
		return nil, teeRecord, errors.New("rpc err: failed to query tee list")
	}
	for i := 0; i < len(teelist); i++ {
		delay = 30
		out.Tip(fmt.Sprintf("Requesting podr2 public key from tee: %s", hex.EncodeToString([]byte(string(teelist[i].Pubkey[:])))))
		for tryCount := uint8(0); tryCount <= 3; tryCount++ {
			endpoint, err := n.QueryEndpoints(teelist[i].Pubkey, -1)
			if err != nil {
				n.Log("err", err.Error())
				continue
			}
			endpoint = ProcessTeeEndpoint(endpoint)
			if !strings.Contains(endpoint, "443") {
				dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
			} else {
				dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
			}
			podr2PubkeyResponse, err = com.GetPodr2Pubkey(
				endpoint,
				&pb.Request{StorageMinerAccountId: n.GetSignatureAccPulickey()},
				time.Duration(time.Second*delay),
				dialOptions,
				nil,
			)
			if err != nil {
				if strings.Contains(err.Error(), configs.Err_ctx_exceeded) {
					delay += 30
					continue
				}
				if strings.Contains(err.Error(), configs.Err_tee_Busy) {
					delay += 10
					continue
				}
				continue
			}
			n.SaveTee(string(teelist[i].Pubkey[:]), endpoint, uint8(teelist[i].Role))
			return podr2PubkeyResponse.Pubkey, teeRecord, nil
		}
	}
	return nil, teeRecord, errors.New("all tee nodes are busy or unavailable")
}

func (n *Node) saveAllTees() (record.TeeRecorder, error) {
	var (
		err            error
		teeList        []chain.WorkerInfo
		dialOptions    []grpc.DialOption
		teeRecord      = record.NewTeeRecord()
		chainPublickey = make([]byte, chain.WorkerPublicKeyLen)
	)
	for {
		teeList, err = n.QueryAllWorkers(-1)
		if err != nil {
			if err.Error() == chain.ERR_Empty {
				out.Err("No tee found, waiting for the next minute's query...")
				time.Sleep(time.Minute)
				continue
			}
			return teeRecord, err
		}
		break
	}

	for _, v := range teeList {
		out.Tip(fmt.Sprintf("Checking the tee: %s", hex.EncodeToString([]byte(string(v.Pubkey[:])))))
		endPoint, err := n.QueryEndpoints(v.Pubkey, -1)
		if err != nil {
			out.Err(fmt.Sprintf("Failed to query endpoints for this tee: %v", err))
			continue
		}
		endPoint = ProcessTeeEndpoint(endPoint)
		if !strings.Contains(endPoint, "443") {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		} else {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
		}
		// verify identity public key
		identityPubkeyResponse, err := com.GetIdentityPubkey(endPoint,
			&pb.Request{
				StorageMinerAccountId: n.GetSignatureAccPulickey(),
			},
			time.Duration(time.Minute),
			dialOptions,
			nil,
		)
		if err != nil {
			out.Err(fmt.Sprintf("Failed to query the identity pubkey for this tee: %v", err))
			continue
		}
		if len(identityPubkeyResponse.Pubkey) != chain.WorkerPublicKeyLen {
			out.Err(fmt.Sprintf("The identity pubkey length of this tee is incorrect: %d", len(identityPubkeyResponse.Pubkey)))
			continue
		}
		for j := 0; j < chain.WorkerPublicKeyLen; j++ {
			chainPublickey[j] = byte(v.Pubkey[j])
		}
		if !sutils.CompareSlice(identityPubkeyResponse.Pubkey, chainPublickey) {
			out.Err("The IdentityPubkey returned by this tee doesn't match the one in the chain")
			continue
		}
		err = teeRecord.SaveTee(string(v.Pubkey[:]), endPoint, uint8(v.Role))
		if err != nil {
			out.Err(fmt.Sprintf("Save tee err: %v", err))
			continue
		}
	}
	return teeRecord, nil
}

func updateMinerRegistertionInfo(cli chain.Chainer, oldRegInfo *chain.MinerInfo, useSpace uint64, stakingAcc, earningsAcc, addr string) error {
	var err error
	olddecspace := oldRegInfo.DeclarationSpace.Uint64() / chain.SIZE_1TiB
	if (*oldRegInfo).DeclarationSpace.Uint64()%chain.SIZE_1TiB != 0 {
		olddecspace = +1
	}
	newDecSpace := useSpace / chain.SIZE_1KiB
	if useSpace%chain.SIZE_1KiB != 0 {
		newDecSpace += 1
	}
	if newDecSpace > olddecspace {
		token := (newDecSpace - olddecspace) * chain.StakingStakePerTiB
		if stakingAcc != "" && stakingAcc != cli.GetSignatureAcc() {
			signAccInfo, err := cli.QueryAccountInfo(cli.GetSignatureAcc(), -1)
			if err != nil {
				if err.Error() != chain.ERR_Empty {
					out.Err(err.Error())
					os.Exit(1)
				}
				out.Err("Failed to expand space: account does not exist or balance is empty")
				os.Exit(1)
			}
			incToken, _ := new(big.Int).SetString(fmt.Sprintf("%d%s", token, chain.TokenPrecision_CESS), 10)
			if signAccInfo.Data.Free.CmpAbs(incToken) < 0 {
				return fmt.Errorf("Failed to expand space: signature account balance less than %d %s", incToken, cli.GetTokenSymbol())
			}
			txhash, err := cli.IncreaseCollateral(cli.GetSignatureAccPulickey(), fmt.Sprintf("%d%s", token, chain.TokenPrecision_CESS))
			if err != nil {
				if txhash != "" {
					return fmt.Errorf("[%s] Failed to expand space: %v", txhash, err)
				}
				return fmt.Errorf("Failed to expand space: %v", err)
			}
			out.Ok(fmt.Sprintf("Successfully increased %dTCESS staking", token))
		} else {
			newToken, _ := new(big.Int).SetString(fmt.Sprintf("%d%s", newDecSpace*chain.StakingStakePerTiB, chain.TokenPrecision_CESS), 10)
			if oldRegInfo.Collaterals.CmpAbs(newToken) < 0 {
				return fmt.Errorf("Please let the staking account add the staking for you first before expande space")
			}
		}
		_, err = cli.IncreaseDeclarationSpace(uint32(newDecSpace - olddecspace))
		if err != nil {
			return err
		}
		out.Ok(fmt.Sprintf("Successfully expanded %dTiB space", newDecSpace-olddecspace))
	}

	newPublicKey, err := sutils.ParsingPublickey(earningsAcc)
	if err == nil {
		if !sutils.CompareSlice(oldRegInfo.BeneficiaryAccount[:], newPublicKey) {
			txhash, err := cli.UpdateBeneficiary(earningsAcc)
			if err != nil {
				return fmt.Errorf("Update earnings account err: %v, blockhash: %s", err, txhash)
			}
			out.Ok(fmt.Sprintf("[%s] Successfully updated earnings account to %s", txhash, earningsAcc))
		}
	}

	if string(oldRegInfo.PeerId[:]) != addr {
		txhash, err := cli.UpdateSminerAddr([]byte(addr))
		if err != nil {
			return fmt.Errorf("Update address err: %v, blockhash: %s", err, txhash)
		}
		out.Ok(fmt.Sprintf("[%s] Successfully updated address to %s", txhash, addr))
	}

	return nil
}

func (n *Node) registerPoisKey() (*RSAKeyPair, *pb.MinerPoisInfo, record.TeeRecorder, error) {
	var (
		err                    error
		teeList                []chain.WorkerInfo
		dialOptions            []grpc.DialOption
		responseMinerInitParam *pb.ResponseMinerInitParam
		rsakey                 *RSAKeyPair
		poisInfo               = &pb.MinerPoisInfo{}
		teeRecord              = record.NewTeeRecord()
		chainPublickey         = make([]byte, chain.WorkerPublicKeyLen)
	)

	teeEndPointList := n.ReadPriorityTeeList()

	for {
		teeList, err = n.QueryAllWorkers(-1)
		if err != nil {
			if err.Error() == chain.ERR_Empty {
				out.Err("No tee found, waiting for the next minute's query...")
				time.Sleep(time.Minute)
				continue
			}
			return rsakey, poisInfo, teeRecord, err
		}
		break
	}

	for _, v := range teeList {
		out.Tip(fmt.Sprintf("Checking the tee: %s", hex.EncodeToString([]byte(string(v.Pubkey[:])))))
		endPoint, err := n.QueryEndpoints(v.Pubkey, -1)
		if err != nil {
			out.Err(fmt.Sprintf("Failed to query endpoints for this tee: %v", err))
			continue
		}
		endPoint = ProcessTeeEndpoint(endPoint)
		if !strings.Contains(endPoint, "443") {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		} else {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
		}
		// verify identity public key
		identityPubkeyResponse, err := com.GetIdentityPubkey(endPoint,
			&pb.Request{
				StorageMinerAccountId: n.GetSignatureAccPulickey(),
			},
			time.Duration(time.Minute),
			dialOptions,
			nil,
		)
		if err != nil {
			out.Err(fmt.Sprintf("Failed to query the identity pubkey for this tee: %v", err))
			continue
		}
		if len(identityPubkeyResponse.Pubkey) != chain.WorkerPublicKeyLen {
			out.Err(fmt.Sprintf("The identity pubkey length of this tee is incorrect: %d", len(identityPubkeyResponse.Pubkey)))
			continue
		}
		for j := 0; j < chain.WorkerPublicKeyLen; j++ {
			chainPublickey[j] = byte(v.Pubkey[j])
		}
		if !sutils.CompareSlice(identityPubkeyResponse.Pubkey, chainPublickey) {
			out.Err("The IdentityPubkey returned by this tee doesn't match the one in the chain")
			continue
		}
		err = teeRecord.SaveTee(string(v.Pubkey[:]), endPoint, uint8(v.Role))
		if err != nil {
			out.Err(fmt.Sprintf("Save tee err: %v", err))
			continue
		}
		teeEndPointList = append(teeEndPointList, endPoint)
	}

	delay := time.Duration(30)
	for i := 0; i < len(teeEndPointList); i++ {
		delay = 30
		for tryCount := uint8(0); tryCount <= 5; tryCount++ {
			out.Tip(fmt.Sprintf("Requesting registration parameters to tee: %s", teeEndPointList[i]))
			if !strings.Contains(teeEndPointList[i], "443") {
				dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
			} else {
				dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
			}
			responseMinerInitParam, err = com.RequestMinerGetNewKey(
				teeEndPointList[i],
				n.GetSignatureAccPulickey(),
				time.Duration(time.Second*delay),
				dialOptions,
				nil,
			)
			if err != nil {
				if strings.Contains(err.Error(), configs.Err_ctx_exceeded) {
					out.Err(fmt.Sprintf("Request err: %v", err))
					delay += 30
					continue
				}
				if strings.Contains(err.Error(), configs.Err_miner_not_exists) {
					out.Err(fmt.Sprintf("Request err: %v", err))
					time.Sleep(chain.BlockInterval * 2)
					continue
				}
				out.Err(fmt.Sprintf("Request err: %v", err))
				break
			}

			rsakey, err = NewRsaKey(responseMinerInitParam.Podr2Pbk)
			if err != nil {
				out.Err(fmt.Sprintf("Request err: %v", err))
				break
			}

			poisInfo.Acc = responseMinerInitParam.Acc
			poisInfo.Front = responseMinerInitParam.Front
			poisInfo.Rear = responseMinerInitParam.Rear
			poisInfo.KeyN = responseMinerInitParam.KeyN
			poisInfo.KeyG = responseMinerInitParam.KeyG
			poisInfo.StatusTeeSign = responseMinerInitParam.StatusTeeSign

			workpublickey, err := teeRecord.GetTeeWorkAccount(teeEndPointList[i])
			if err != nil {
				break
			}

			poisKey, err := chain.BytesToPoISKeyInfo(responseMinerInitParam.KeyG, responseMinerInitParam.KeyN)
			if err != nil {
				out.Err(fmt.Sprintf("Request err: %v", err))
				continue
			}

			teeWorkPubkey, err := chain.BytesToWorkPublickey([]byte(workpublickey))
			if err != nil {
				out.Err(fmt.Sprintf("Request err: %v", err))
				continue
			}

			err = registerMinerPoisKey(
				n.Chainer,
				poisKey,
				responseMinerInitParam.SignatureWithTeeController[:],
				responseMinerInitParam.StatusTeeSign,
				teeWorkPubkey,
			)
			if err != nil {
				out.Err(fmt.Sprintf("register miner pois key err: %v", err))
				break
			}
			return rsakey, poisInfo, teeRecord, err
		}
	}
	return rsakey, poisInfo, teeRecord, errors.New("all tee nodes are busy or unavailable")
}

func registerMinerPoisKey(cli chain.Chainer, poisKey chain.PoISKeyInfo, teeSignWithAcc types.Bytes, teeSign types.Bytes, teePuk chain.WorkerPublicKey) error {
	var err error
	for i := 0; i < 3; i++ {
		_, err = cli.RegisterPoisKey(
			poisKey,
			teeSignWithAcc,
			teeSign,
			teePuk,
		)
		if err != nil {
			time.Sleep(chain.BlockInterval * 2)
			minerInfo, err := cli.QueryMinerItems(cli.GetSignatureAccPulickey(), -1)
			if err != nil {
				return err
			}
			if minerInfo.SpaceProofInfo.HasValue() {
				return nil
			}
			continue
		}
		return nil
	}
	return err
}

func registerMiner(cli chain.Chainer, stakingAcc, earningsAcc, sip string, decTib uint64) (string, error) {
	if stakingAcc != "" && stakingAcc != cli.GetSignatureAcc() {
		out.Ok(fmt.Sprintf("Specify staking account: %s", stakingAcc))
		txhash, err := cli.RegnstkAssignStaking(earningsAcc, []byte(sip), stakingAcc, uint32(decTib))
		if err != nil {
			if txhash != "" {
				err = fmt.Errorf("[%s] %v", txhash, err)
			}
			return txhash, err
		}
		out.Ok(fmt.Sprintf("Storage miner registration successful: %s", txhash))
		return txhash, nil
	}

	txhash, err := cli.RegnstkSminer(earningsAcc, []byte(sip), uint64(decTib*chain.StakingStakePerTiB), uint32(decTib))
	if err != nil {
		if txhash != "" {
			err = fmt.Errorf("[%s] %v", txhash, err)
		}
		return txhash, err
	}
	out.Ok(fmt.Sprintf("Storage miner registration successful: %s", txhash))
	return txhash, nil
}

func checkRpcSynchronization(cli chain.Chainer) error {
	out.Tip("Waiting to synchronize the main chain...")
	var err error
	var syncSt chain.SysSyncState
	for {
		syncSt, err = cli.SystemSyncState()
		if err != nil {
			return err
		}
		if syncSt.CurrentBlock == syncSt.HighestBlock {
			out.Ok(fmt.Sprintf("Synchronization the main chain completed: %d", syncSt.CurrentBlock))
			break
		}
		out.Tip(fmt.Sprintf("In the synchronization main chain: %d ...", syncSt.CurrentBlock))
		time.Sleep(time.Second * time.Duration(utils.Ternary(int64(syncSt.HighestBlock-syncSt.CurrentBlock)*6, 30)))
	}
	return nil
}

func checkVersion(cli chain.Chainer) error {
	chainVersion, err := cli.SystemVersion()
	if err != nil {
		return errors.New("failed to query the chain version: rpc connection is down")
	}
	chain := cli.GetNetworkEnv()
	if strings.Contains(chain, configs.TestNet) {
		tmps := strings.Split(chainVersion, "-")
		for _, v := range tmps {
			if strings.Contains(v, ".") {
				values := strings.Split(v, ".")
				if len(values) == 3 {
					if values[0] != configs.ChainVersionStr[0] || values[1] != configs.ChainVersionStr[1] {
						return fmt.Errorf("chain version number is not v%d.%d, please check your rpc service", configs.ChainVersionInt[0], configs.ChainVersionInt[1])
					}
					versionI, err := strconv.Atoi(values[2])
					if err == nil {
						if versionI < configs.ChainVersionInt[2] {
							return fmt.Errorf("chain version number is lower than v%d.%d.%d, please check your rpc service", configs.ChainVersionInt[0], configs.ChainVersionInt[1], configs.ChainVersionInt[2])
						}
					}
				}
			}
		}
	}
	return nil
}

func (n *Node) checkRegistrationInfo() (int, uint64, *chain.MinerInfo, error) {
	minerInfo, err := n.QueryMinerItems(n.GetSignatureAccPulickey(), -1)
	if err != nil {
		if err.Error() != chain.ERR_Empty {
			return Unregistered, 0, &minerInfo, err
		}
		useSpace := uint64(n.ReadUseCpu())
		stakingAcc := n.ReadStakingAcc()
		signatureAcc := n.ReadSignatureAccount()
		decTib := useSpace / chain.SIZE_1KiB
		if useSpace%chain.SIZE_1KiB != 0 {
			decTib += 1
		}
		token := decTib * chain.StakingStakePerTiB
		accInfo, err := n.QueryAccountInfo(n.GetSignatureAcc(), -1)
		if err != nil {
			if err.Error() != chain.ERR_Empty {
				return Unregistered, decTib, &minerInfo, fmt.Errorf("failed to query signature account information: %v", err)
			}
			return Unregistered, decTib, &minerInfo, errors.New("signature account does not exist, possible cause: 1.balance is empty 2.wrong rpc address")
		}
		token_cess, _ := new(big.Int).SetString(fmt.Sprintf("%d%s", token, chain.TokenPrecision_CESS), 10)
		if stakingAcc == "" || stakingAcc == signatureAcc {
			if accInfo.Data.Free.CmpAbs(token_cess) < 0 {
				return Unregistered, decTib, &minerInfo, fmt.Errorf("signature account balance less than %d %s", token, n.GetTokenSymbol())
			}
		} else {
			stakingAccInfo, err := n.QueryAccountInfo(stakingAcc, -1)
			if err != nil {
				if err.Error() != chain.ERR_Empty {
					return Unregistered, decTib, &minerInfo, fmt.Errorf("failed to query staking account information: %v", err)
				}
				return Unregistered, decTib, &minerInfo, fmt.Errorf("staking account does not exist, possible: 1.balance is empty 2.wrong rpc address")
			}
			if stakingAccInfo.Data.Free.CmpAbs(token_cess) < 0 {
				return Unregistered, decTib, &minerInfo, fmt.Errorf("staking account balance less than %d %s", token, n.GetTokenSymbol())
			}
		}
		return Unregistered, decTib, &minerInfo, nil
	}
	if !minerInfo.SpaceProofInfo.HasValue() {
		return UnregisteredPoisKey, 0, &minerInfo, nil
	}
	return Registered, 0, &minerInfo, nil
}

func (n *Node) InitWebServer(mdls []gin.HandlerFunc, hdl *web.Handler) {
	gin.SetMode(gin.ReleaseMode)
	n.Engine = gin.Default()
	n.Engine.Use(mdls...)
	hdl.RegisterRoutes(n.Engine)
	go func() {
		err := n.Engine.Run(fmt.Sprintf(":%d", n.ReadServicePort()))
		if err != nil {
			log.Fatal(err)
		}
	}()
}

func GetLocalIP() (string, error) {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		return "", errors.Wrap(err, "[net.Interfaces]")
	}
	for i := 0; i < len(netInterfaces); i++ {
		if (netInterfaces[i].Flags & net.FlagUp) != 0 {
			addrs, _ := netInterfaces[i].Addrs()
			for _, address := range addrs {
				if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && !ipnet.IP.IsPrivate() && !ipnet.IP.IsUnspecified() {
					// IPv4
					if ipnet.IP.To4() != nil {
						return ipnet.IP.String(), nil
					}
					// IPv6
					if ipnet.IP.To16() != nil {
						return ipnet.IP.String(), nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("No available ip address found")
}

func InitMiddlewares() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		cors.New(cors.Config{
			AllowAllOrigins: true,
			AllowHeaders: []string{
				common.Header_Account,
				common.Header_Message,
				common.Header_Signature,
				common.Header_Fid,
				common.Header_Fragment,
				common.Header_ContentType,
				common.Header_X_Forwarded_For,
			},
			AllowMethods: []string{"PUT", "GET", "OPTION"},
		}),
	}
}
