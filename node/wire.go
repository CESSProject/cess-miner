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
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/node/runstatus"
	"github.com/CESSProject/cess-miner/node/web"
	"github.com/CESSProject/cess-miner/node/workspace"
	"github.com/CESSProject/cess-miner/pkg/cache"
	"github.com/CESSProject/cess-miner/pkg/com"
	"github.com/CESSProject/cess-miner/pkg/com/pb"
	out "github.com/CESSProject/cess-miner/pkg/fout"
	"github.com/CESSProject/cess-miner/pkg/logger"
	"github.com/CESSProject/cess-miner/pkg/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	sdkgo "github.com/CESSProject/cess-go-sdk"
	"github.com/CESSProject/cess-go-sdk/chain"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
)

func InitNode() *Node {
	wire.Build(
		GetNode,
		InitWebServer,
		InitMiddlewares,
		web.NewHandler,
		InitChainClient,
		InitRSAKeyPair,
		InitTeeRecord,
		InitMinerPoisInfo,
		InitPois,
		InitRunStatus,
		InitLogs,
		InitCache,
	)
	return &Node{}
}

func InitRunStatus(n *Node, cli *chain.ChainClient, st types.Bytes) runstatus.Runstatus {
	rt := runstatus.NewRunstatus()
	rt.SetPID(os.Getpid())
	rt.SetCpucores(int(n.ReadUseCpu()))
	rt.SetCurrentRpc(cli.GetCurrentRpcAddr())
	rt.SetCurrentRpcst(cli.GetRpcState())
	rt.SetSignAcc(cli.GetSignatureAcc())
	rt.SetState(string(st))
	InitRunstatus(rt)
	return rt
}

func InitCache(n *Node, cli *chain.ChainClient) {
	cace, err := cache.NewCache(n.GetDbDir(), 0, 0, configs.NameSpaces)
	if err != nil {
		out.Err(fmt.Sprintf("[NewCache] %v", err))
		os.Exit(1)
	}
	InitCacher(cace)
}

func InitLogs(n *Node, cli *chain.ChainClient) {
	var logs_info = make(map[string]string)
	for _, v := range logger.LogFiles {
		logs_info[v] = filepath.Join(n.GetLogDir(), v+".log")
	}
	lg, err := logger.NewLogs(logs_info)
	if err != nil {
		out.Err(fmt.Sprintf("[NewLogs] %v", err))
		os.Exit(1)
	}
	InitLogger(lg)
}

func InitChainClient(n *Node, sip string) (*chain.ChainClient, *RSAKeyPair, *pb.MinerPoisInfo, *Pois, *TeeRecord, workspace.Workspace, types.Bytes) {
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

	InitWorkspace(filepath.Join(n.ReadWorkspace(), n.GetSignatureAcc(), configs.Name))
	InitChainclient(cli)

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

	rsakey, poisInfo, pois, teeRecord, st, err := checkMiner(n, sip)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}
	return cli, rsakey, poisInfo, pois, teeRecord, n.Workspace, st
}

func checkMiner(n *Node, sip string) (*RSAKeyPair, *pb.MinerPoisInfo, *Pois, *TeeRecord, types.Bytes, error) {
	var rsakey *RSAKeyPair
	var poisInfo = &pb.MinerPoisInfo{}
	var p *Pois
	var teeRecord *TeeRecord

	register, decTib, oldRegInfo, err := checkRegistrationInfo(n.ChainClient, n.ReadSignatureAccount(), n.ReadStakingAcc(), n.ReadUseSpace())
	if err != nil {
		return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[checkMiner]")
	}

	switch register {
	case Unregistered:
		_, err = registerMiner(n.ChainClient, n.ReadStakingAcc(), n.ReadEarningsAcc(), sip, decTib)
		if err != nil {
			return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[registerMiner]")
		}

		err = n.RemoveAndBuild()
		if err != nil {
			return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[RemoveAndBuild]")
		}

		for i := 0; i < 3; i++ {
			rsakey, poisInfo, teeRecord, err = registerPoisKey(n)
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
			rsakey, poisInfo, teeRecord, err = registerPoisKey(n)
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

		err = updateMinerRegistertionInfo(n.ChainClient, oldRegInfo, n.ReadUseSpace(), n.ReadStakingAcc(), n.ReadEarningsAcc())
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

		err = updateMinerRegistertionInfo(n.ChainClient, oldRegInfo, n.ReadUseSpace(), n.ReadStakingAcc(), n.ReadEarningsAcc())
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

		teeRecord, err = saveAllTees(n.ChainClient)
		if err != nil {
			return rsakey, poisInfo, p, teeRecord, oldRegInfo.State, errors.Wrap(err, "[saveAllTees]")
		}

		buf, err := queryPodr2KeyFromTee(teeRecord.GetAllTeeEndpoint(), n.GetSignatureAccPulickey(), n.ReadPriorityTeeList())
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

func queryPodr2KeyFromTee(teeEndPointList []string, signature_publickey []byte, priorityTeeList []string) ([]byte, error) {
	var err error
	var podr2PubkeyResponse *pb.Podr2PubkeyResponse
	var dialOptions []grpc.DialOption
	delay := time.Duration(30)

	var allTee []string

	if len(priorityTeeList) > 0 {
		allTee = append(allTee, priorityTeeList...)
		allTee = append(allTee, priorityTeeList...)
		allTee = append(allTee, priorityTeeList...)
	}
	allTee = append(allTee, teeEndPointList...)
	for i := 0; i < len(allTee); i++ {
		delay = 30
		out.Tip(fmt.Sprintf("Requesting podr2 public key from tee: %s", allTee[i]))
		for tryCount := uint8(0); tryCount <= 3; tryCount++ {
			if !strings.Contains(allTee[i], "443") {
				dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
			} else {
				dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
			}
			podr2PubkeyResponse, err = com.GetPodr2Pubkey(
				allTee[i],
				&pb.Request{StorageMinerAccountId: signature_publickey},
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
			return podr2PubkeyResponse.Pubkey, nil
		}
	}
	return nil, errors.New("all tee nodes are busy or unavailable")
}

func saveAllTees(cli *chain.ChainClient) (*TeeRecord, error) {
	var (
		err            error
		teeList        []chain.WorkerInfo
		dialOptions    []grpc.DialOption
		teeRecord      = NewTeeRecord()
		chainPublickey = make([]byte, chain.WorkerPublicKeyLen)
	)
	for {
		teeList, err = cli.QueryAllWorkers(-1)
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
		endPoint, err := cli.QueryEndpoints(v.Pubkey, -1)
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
				StorageMinerAccountId: cli.GetSignatureAccPulickey(),
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

func updateMinerRegistertionInfo(cli *chain.ChainClient, oldRegInfo *chain.MinerInfo, useSpace uint64, stakingAcc, earningsAcc string) error {
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

	return nil
}

func registerPoisKey(n *Node) (*RSAKeyPair, *pb.MinerPoisInfo, *TeeRecord, error) {
	var (
		err                    error
		teeList                []chain.WorkerInfo
		dialOptions            []grpc.DialOption
		responseMinerInitParam *pb.ResponseMinerInitParam
		rsakey                 *RSAKeyPair
		poisInfo               = &pb.MinerPoisInfo{}
		teeRecord              = NewTeeRecord()
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
				n.ChainClient,
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

func registerMinerPoisKey(cli *chain.ChainClient, poisKey chain.PoISKeyInfo, teeSignWithAcc types.Bytes, teeSign types.Bytes, teePuk chain.WorkerPublicKey) error {
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

func registerMiner(cli *chain.ChainClient, stakingAcc, earningsAcc, sip string, decTib uint64) (string, error) {
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

func checkRpcSynchronization(cli *chain.ChainClient) error {
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

func checkVersion(cli *chain.ChainClient) error {
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

func checkRegistrationInfo(cli *chain.ChainClient, signatureAcc, stakingAcc string, useSpace uint64) (int, uint64, *chain.MinerInfo, error) {
	minerInfo, err := cli.QueryMinerItems(cli.GetSignatureAccPulickey(), -1)
	if err != nil {
		if err.Error() != chain.ERR_Empty {
			return Unregistered, 0, &minerInfo, err
		}
		decTib := useSpace / chain.SIZE_1KiB
		if useSpace%chain.SIZE_1KiB != 0 {
			decTib += 1
		}
		token := decTib * chain.StakingStakePerTiB
		accInfo, err := cli.QueryAccountInfo(cli.GetSignatureAcc(), -1)
		if err != nil {
			if err.Error() != chain.ERR_Empty {
				return Unregistered, decTib, &minerInfo, fmt.Errorf("failed to query signature account information: %v", err)
			}
			return Unregistered, decTib, &minerInfo, errors.New("signature account does not exist, possible cause: 1.balance is empty 2.wrong rpc address")
		}
		token_cess, _ := new(big.Int).SetString(fmt.Sprintf("%d%s", token, chain.TokenPrecision_CESS), 10)
		if stakingAcc == "" || stakingAcc == signatureAcc {
			if accInfo.Data.Free.CmpAbs(token_cess) < 0 {
				return Unregistered, decTib, &minerInfo, fmt.Errorf("signature account balance less than %d %s", token, cli.GetTokenSymbol())
			}
		} else {
			stakingAccInfo, err := cli.QueryAccountInfo(stakingAcc, -1)
			if err != nil {
				if err.Error() != chain.ERR_Empty {
					return Unregistered, decTib, &minerInfo, fmt.Errorf("failed to query staking account information: %v", err)
				}
				return Unregistered, decTib, &minerInfo, fmt.Errorf("staking account does not exist, possible: 1.balance is empty 2.wrong rpc address")
			}
			if stakingAccInfo.Data.Free.CmpAbs(token_cess) < 0 {
				return Unregistered, decTib, &minerInfo, fmt.Errorf("staking account balance less than %d %s", token, cli.GetTokenSymbol())
			}
		}
		return Unregistered, decTib, &minerInfo, nil
	}
	if !minerInfo.SpaceProofInfo.HasValue() {
		return UnregisteredPoisKey, 0, &minerInfo, nil
	}
	return Registered, 0, &minerInfo, nil
}

func InitWebServer(n *Node, mdls []gin.HandlerFunc, hdl *web.Handler) string {
	n.Engine = gin.Default()
	n.Engine.Use(mdls...)
	hdl.RegisterRoutes(n.Engine)
	go func() {
		err := n.Engine.Run(fmt.Sprintf(":%d", n.ReadServicePort()))
		if err != nil {
			log.Fatal(err)
		}
	}()
	ip, err := GetLocalIP()
	if err != nil {
		log.Fatal(err)
	}
	ip = fmt.Sprintf("http://%s:%d", ip, n.ReadServicePort())
	// TODO
	// sip:=Encrypted(ip)
	return ip
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
			AllowHeaders:    []string{"Content-Type", "Account", "Message", "Signature", "Fid", "Fragment"},
			AllowMethods:    []string{"PUT", "GET", "OPTION"},
		}),
		func(ctx *gin.Context) {
			ok, err := VerifySignature(ctx)
			if !ok || err != nil {
				ctx.AbortWithStatus(http.StatusForbidden)
				return
			}
			ctx.Next()
		},
	}
}

func VerifySignature(ctx *gin.Context) (bool, error) {
	account := ctx.Request.Header.Get("Account")
	message := ctx.Request.Header.Get("Message")
	signature := ctx.Request.Header.Get("Signature")
	return sutils.VerifySR25519WithPublickey(message, []byte(signature), account)
}
