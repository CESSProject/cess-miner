/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"bufio"
	"context"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AstaFrode/go-libp2p/core/peer"
	"github.com/AstaFrode/go-libp2p/core/peerstore"
	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	"github.com/CESSProject/cess-bucket/pkg/cache"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	cess "github.com/CESSProject/cess-go-sdk"
	sconfig "github.com/CESSProject/cess-go-sdk/config"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	p2pgo "github.com/CESSProject/p2p-go"
	"github.com/CESSProject/p2p-go/config"
	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/p2p-go/out"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/howeyc/gopass"
	"github.com/mr-tron/base58"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// runCmd is used to start the service
func runCmd(cmd *cobra.Command, args []string) {
	var (
		firstReg       bool
		err            error
		bootEnv        string
		token          uint64
		spaceTiB       uint32
		protocolPrefix string
		syncSt         pattern.SysSyncState
		n              = node.New()
	)
	n.SetInitStage(node.Stage_Startup, "Program startup")
	n.SetPID(int32(os.Getpid()))
	ctx := context.Background()
	n.ListenLocal()
	n.SetInitStage(node.Stage_ReadConfig, "Reading configuration...")
	// parse configuration file
	n.Confile, err = buildConfigFile(cmd, 0)
	if err != nil {
		out.Err(fmt.Sprintf("[buildConfigFile] %v", err))
		n.SetInitStage(node.Stage_ReadConfig, fmt.Sprintf("[err] %v", err))
		os.Exit(1)
	}
	n.SetInitStage(node.Stage_ReadConfig, "[ok] Read configuration file")
	n.SetCpuCores(configs.SysInit(n.GetUseCpu()))

	out.Tip(fmt.Sprintf("RPC addresses: %v", n.GetRpcAddr()))

	boots := n.GetBootNodes()
	for _, v := range boots {
		if strings.Contains(v, "testnet") {
			bootEnv = "cess-testnet"
			protocolPrefix = config.TestnetProtocolPrefix
			break
		} else if strings.Contains(v, "mainnet") {
			bootEnv = "cess-mainnet"
			protocolPrefix = config.MainnetProtocolPrefix
			break
		} else if strings.Contains(v, "devnet") {
			bootEnv = "cess-devnet"
			protocolPrefix = config.DevnetProtocolPrefix
			break
		} else {
			bootEnv = "unknown"
		}
	}
	out.Tip(fmt.Sprintf("Bootnodes: %v", boots))

	n.SetInitStage(node.Stage_ConnectRpc, "Connecting to rpc...")
	// build client
	n.SDK, err = cess.New(
		ctx,
		cess.Name(sconfig.CharacterName_Bucket),
		cess.ConnectRpcAddrs(n.GetRpcAddr()),
		cess.Mnemonic(n.GetMnemonic()),
		cess.TransactionTimeout(configs.TimeToWaitEvent),
	)
	if err != nil {
		out.Err(fmt.Sprintf("[cess.New] %v", err))
		n.SetInitStage(node.Stage_ConnectRpc, fmt.Sprintf("[err] %v", err))
		os.Exit(1)
	}
	n.SetInitStage(node.Stage_ConnectRpc, fmt.Sprintf("[ok] Connect rpc: %s", n.GetCurrentRpcAddr()))

	n.SetInitStage(node.Stage_CreateP2p, "Create p2p node...")
	n.P2P, err = p2pgo.New(
		ctx,
		p2pgo.ListenPort(n.GetServicePort()),
		p2pgo.Workspace(filepath.Join(n.GetWorkspace(), n.GetSignatureAcc(), n.GetSDKName())),
		p2pgo.BootPeers(n.GetBootNodes()),
		p2pgo.ProtocolPrefix(protocolPrefix),
	)
	if err != nil {
		out.Err(fmt.Sprintf("[p2pgo.New] %v", err))
		n.SetInitStage(node.Stage_CreateP2p, fmt.Sprintf("[err] %v", err))
		os.Exit(1)
	}
	n.SetInitStage(node.Stage_CreateP2p, fmt.Sprintf("[ok] Create p2p node: %s", n.ID().Pretty()))

	out.Tip(fmt.Sprintf("Local peer id: %s", n.ID().Pretty()))
	out.Tip(fmt.Sprintf("Chain network: %s", n.GetNetworkEnv()))
	out.Tip(fmt.Sprintf("P2P network: %s", bootEnv))
	out.Tip(fmt.Sprintf("Number of cpu cores used: %v", n.GetCpuCores()))
	out.Tip(fmt.Sprintf("RPC address used: %v", n.GetCurrentRpcAddr()))
	//
	// out.Tip(fmt.Sprintf("Local account publickey: %v", n.GetSignatureAccPulickey()))
	// out.Tip(fmt.Sprintf("Protocol version: %s", n.GetProtocolVersion()))
	// out.Tip(fmt.Sprintf("DHT protocol version: %s", n.GetDhtProtocolVersion()))
	// out.Tip(fmt.Sprintf("Rendezvous version: %s", n.GetRendezvousVersion()))

	if strings.Contains(bootEnv, "test") {
		if !strings.Contains(n.GetNetworkEnv(), "test") {
			out.Warn("Chain and p2p are not in the same network")
		}
	}

	if strings.Contains(bootEnv, "main") {
		if !strings.Contains(n.GetNetworkEnv(), "main") {
			out.Warn("Chain and p2p are not in the same network")
		}
	}

	if strings.Contains(bootEnv, "dev") {
		if !strings.Contains(n.GetNetworkEnv(), "dev") {
			out.Warn("Chain and p2p are not in the same network")
		}
	}

	n.SetInitStage(node.Stage_SyncBlock, "Sync block...")
	for {
		syncSt, err = n.SyncState()
		if err != nil {
			out.Err("Invalid chain node: rpc service failure")
			n.SetInitStage(node.Stage_SyncBlock, fmt.Sprintf("[err] %v", err))
			os.Exit(1)
		}
		if syncSt.CurrentBlock == syncSt.HighestBlock {
			out.Ok(fmt.Sprintf("Synchronization main chain completed: %d", syncSt.CurrentBlock))
			break
		}
		out.Tip(fmt.Sprintf("In the synchronization main chain: %d ...", syncSt.CurrentBlock))
		time.Sleep(time.Second * time.Duration(utils.Ternary(int64(syncSt.HighestBlock-syncSt.CurrentBlock)*6, 30)))
	}
	n.SetInitStage(node.Stage_SyncBlock, fmt.Sprintf("[ok] Latest block: %d", syncSt.HighestBlock))
	n.SetInitStage(node.Stage_QueryChain, "Querying chain...")
	chainVersion, err := n.ChainVersion()
	if err != nil {
		out.Err("[SysVersion] Invalid chain node: rpc service failure")
		os.Exit(1)
	}

	if strings.Contains(n.GetNetworkEnv(), "test") {
		if !strings.Contains(chainVersion, configs.ChainVersion) {
			out.Err(fmt.Sprintf("The chain version is not %v", configs.ChainVersion))
			os.Exit(1)
		}
	}

	n.ExpendersInfo, err = n.QueryExpenders()
	if err != nil {
		if err.Error() == pattern.ERR_Empty {
			out.Err("chain err: expenders is empty")
		} else {
			out.Err(err.Error())
		}
		os.Exit(1)
	}

	var teeAcc string
	var teeEndPointList = make([]string, 0)
	for {
		teeList, err := n.QueryAllTeeWorkerMap()
		if err != nil {
			if err.Error() == pattern.ERR_Empty {
				out.Err("No TEE was found, waiting for the next query...")
				time.Sleep(time.Minute)
				continue
			}
			out.Err(err.Error())
			os.Exit(1)
		}

		for _, v := range teeList {
			endPoint, err := n.QueryTeeWorkEndpoint(v.Pubkey)
			if err != nil {
				continue
			}

			err = n.SaveTee(string(v.Pubkey[:]), endPoint, uint8(v.Role))
			if err != nil {
				out.Err(fmt.Sprintf("[SaveTee] %v", err))
				continue
			}
		}
		break
	}
	teeEndPointList = n.GetPriorityTeeList()
	teeEndPointList = append(teeEndPointList, n.GetAllTeeEndpoint()...)

	minerInfo, err := n.QueryStorageMiner(n.GetSignatureAccPulickey())
	if err != nil {
		if err.Error() == pattern.ERR_Empty {
			firstReg = true
			n.SetInitStage(node.Stage_QueryChain, "[ok] Complete query")
			token = n.GetUseSpace() / pattern.SIZE_1KiB
			if n.GetUseSpace()%pattern.SIZE_1KiB != 0 {
				token += 1
			}
			spaceTiB = uint32(token)
			token *= pattern.StakingStakePerTiB
			accInfo, err := n.QueryAccountInfo(n.GetSignatureAccPulickey())
			if err != nil {
				if err.Error() != pattern.ERR_Empty {
					out.Err("Invalid chain node: rpc service failure")
					os.Exit(1)
				}
				out.Err("Account does not exist or balance is empty")
				os.Exit(1)
			}
			if n.GetStakingAcc() == "" {
				token_cess, _ := new(big.Int).SetString(fmt.Sprintf("%d%s", token, pattern.TokenPrecision_CESS), 10)
				if accInfo.Data.Free.CmpAbs(token_cess) < 0 {
					out.Err(fmt.Sprintf("Account balance less than %d %s", token, n.GetTokenSymbol()))
					os.Exit(1)
				}
			}
		} else {
			out.Err(pattern.ERR_RPC_CONNECTION.Error())
			os.Exit(1)
		}
	} else {
		n.SetInitStage(node.Stage_QueryChain, "[ok] Complete query")
		err = n.SaveMinerState(string(minerInfo.State))
		if err != nil {
			out.Err(err.Error())
		}
		n.SaveMinerSpaceInfo(
			minerInfo.DeclarationSpace.Uint64(),
			minerInfo.IdleSpace.Uint64(),
			minerInfo.ServiceSpace.Uint64(),
			minerInfo.LockSpace.Uint64(),
		)
	}

	n.SetInitStage(node.Stage_BuildDir, "[ok] Build directory...")
	n.DataDir, err = buildDir(n.Workspace())
	if err != nil {
		n.SetInitStage(node.Stage_BuildDir, fmt.Sprintf("[err] %v", err))
		out.Err(fmt.Sprintf("[buildDir] %v", err))
		os.Exit(1)
	}
	n.SetInitStage(node.Stage_BuildDir, "[ok] Build directory completed")

	for _, b := range boots {
		multiaddr, err := core.ParseMultiaddrs(b)
		if err != nil {
			out.Err(fmt.Sprintf("[ParseMultiaddrs] %v", err))
			continue
		}
		for _, v := range multiaddr {
			maAddr, err := ma.NewMultiaddr(v)
			if err != nil {
				continue
			}
			addrInfo, err := peer.AddrInfoFromP2pAddr(maAddr)
			if err != nil {
				continue
			}
			if addrInfo.ID == n.ID() {
				continue
			}
			err = n.Connect(n.GetCtxQueryFromCtxCancel(), *addrInfo)
			if err != nil {
				out.Err(fmt.Sprintf("Failed to connect to %s: %v", addrInfo.ID.Pretty(), err))
			} else {
				out.Tip(fmt.Sprintf("Connected to %s successfully", addrInfo.ID.Pretty()))
			}
			n.GetDht().RoutingTable().TryAddPeer(addrInfo.ID, true, true)
			n.Peerstore().AddAddr(addrInfo.ID, maAddr, peerstore.PermanentAddrTTL)
			n.SavePeer(*addrInfo)
		}
	}

	var suc bool
	var dialOptions []grpc.DialOption
	var responseMinerInitParam *pb.ResponseMinerInitParam
	var delay time.Duration
	if firstReg {
		n.SetInitStage(node.Stage_Register, "[ok] Registering...")
		stakingAcc := n.GetStakingAcc()
		if stakingAcc != "" {
			out.Ok(fmt.Sprintf("Specify staking account: %s", stakingAcc))
			txhash, err := n.RegisterSminerAssignStaking(n.GetEarningsAcc(), n.GetPeerPublickey(), stakingAcc, spaceTiB)
			if err != nil {
				if txhash != "" {
					err = fmt.Errorf("[%s] %v", txhash, err)
				}
				out.Err(err.Error())
				os.Exit(1)
			}
			out.Ok(fmt.Sprintf("Storage node registration successful: %s", txhash))
		} else {
			txhash, err := n.RegisterSminer(n.GetEarningsAcc(), n.GetPeerPublickey(), token, spaceTiB)
			if err != nil {
				if txhash != "" {
					err = fmt.Errorf("[%s] %v", txhash, err)
				}
				out.Err(err.Error())
				os.Exit(1)
			}
			out.Ok(fmt.Sprintf("Storage node registration successful: %s", txhash))
		}
		n.SetInitStage(node.Stage_Register, "[ok] Registration is complete")
		n.RebuildDirs()

		time.Sleep(pattern.BlockInterval * 5)

		for i := 0; i < len(teeEndPointList); i++ {
			delay = 20
			suc = false
			for tryCount := uint8(0); tryCount <= 5; tryCount++ {
				out.Tip(fmt.Sprintf("Will request miner init param to %v", teeEndPointList[i]))
				if !strings.Contains(teeEndPointList[i], "443") {
					dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
				} else {
					dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
				}
				responseMinerInitParam, err = n.RequestMinerGetNewKey(
					teeEndPointList[i],
					n.GetSignatureAccPulickey(),
					time.Duration(time.Second*delay),
					dialOptions,
					nil,
				)
				if err != nil {
					if strings.Contains(err.Error(), configs.Err_ctx_exceeded) {
						delay += 30
						continue
					}
					if strings.Contains(err.Error(), configs.Err_miner_not_exists) {
						time.Sleep(pattern.BlockInterval * 2)
						continue
					}
					out.Err(fmt.Sprintf("[RequestMinerGetNewKey] %v", err))
					break
				}
				teeAcc, _ = n.GetTeeWorkAccount(teeEndPointList[i])
				suc = true
				break
			}
			if suc {
				n.MinerPoisInfo = &pb.MinerPoisInfo{
					Acc:           responseMinerInitParam.Acc,
					Front:         responseMinerInitParam.Front,
					Rear:          responseMinerInitParam.Rear,
					KeyN:          responseMinerInitParam.KeyN,
					KeyG:          responseMinerInitParam.KeyG,
					StatusTeeSign: responseMinerInitParam.StatusTeeSign,
				}
				err = n.SetPublickey(responseMinerInitParam.Podr2Pbk)
				if err != nil {
					out.Err("invalid podr2 public key")
					os.Exit(1)
				}
				err = os.WriteFile(n.DataDir.Podr2PubkeyFile, responseMinerInitParam.Podr2Pbk, os.ModePerm)
				if err != nil {
					out.Err(fmt.Sprintf("write %v to Podr2PubkeyFile failed: %v", responseMinerInitParam.Podr2Pbk, err))
					os.Exit(1)
				}
				break
			}
		}

		if !suc {
			out.Err("All tee nodes are busy or unavailable, program exits.")
			os.Exit(1)
		}
		poisKey, err := sutils.BytesToPoISKeyInfo(n.MinerPoisInfo.KeyG, n.MinerPoisInfo.KeyN)
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}

		teeWorkPubkey, err := sutils.BytesToWorkPublickey([]byte(teeAcc))
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}
		txhash, err := n.RegisterSminerPOISKey(
			poisKey,
			responseMinerInitParam.SignatureWithTeeController[:],
			n.MinerPoisInfo.StatusTeeSign[:],
			teeWorkPubkey,
		)
		if err != nil {
			if txhash != "" {
				out.Err(fmt.Sprintf("[%s] Register POIS key failed: %v", txhash, err))
			} else {
				out.Err(fmt.Sprintf("Register POIS key failed: %v", err))
			}
			os.Exit(1)
		}
		err = n.InitPois(
			firstReg,
			0,
			0,
			int64(n.GetUseSpace()*1024),
			32,
			*new(big.Int).SetBytes(n.MinerPoisInfo.KeyN),
			*new(big.Int).SetBytes(n.MinerPoisInfo.KeyG),
		)
		if err != nil {
			out.Err(fmt.Sprintf("[Init Pois] %v", err))
			os.Exit(1)
		}
	} else {
		n.SetInitStage(node.Stage_Register, "[ok] Registered")
		var spaceProofInfo pattern.SpaceProofInfo
		var teeSign []byte
		var earningsAcc string
		var peerid []byte
		if !minerInfo.SpaceProofInfo.HasValue() {
			firstReg = true
			for i := 0; i < len(teeEndPointList); i++ {
				delay = 30
				suc = false
				for tryCount := uint8(0); tryCount <= 3; tryCount++ {
					out.Tip(fmt.Sprintf("Will request miner init param to %v", teeEndPointList[i]))
					if !strings.Contains(teeEndPointList[i], "443") {
						dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
					} else {
						dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
					}
					responseMinerInitParam, err = n.RequestMinerGetNewKey(
						teeEndPointList[i],
						n.GetSignatureAccPulickey(),
						time.Duration(time.Second*delay),
						dialOptions,
						nil,
					)
					if err != nil {
						if strings.Contains(err.Error(), configs.Err_ctx_exceeded) {
							delay += 50
							continue
						}
						out.Err(fmt.Sprintf("[RequestMinerGetNewKey] %v", err))
						break
					}
					teeAcc, _ = n.GetTeeWorkAccount(teeEndPointList[i])
					suc = true
					break
				}
				if suc {
					n.MinerPoisInfo = &pb.MinerPoisInfo{
						Acc:           responseMinerInitParam.Acc,
						Front:         responseMinerInitParam.Front,
						Rear:          responseMinerInitParam.Rear,
						KeyN:          responseMinerInitParam.KeyN,
						KeyG:          responseMinerInitParam.KeyG,
						StatusTeeSign: responseMinerInitParam.StatusTeeSign,
					}
					err = n.SetPublickey(responseMinerInitParam.Podr2Pbk)
					if err != nil {
						out.Err("invalid podr2 public key")
						os.Exit(1)
					}
					err = os.WriteFile(n.DataDir.Podr2PubkeyFile, responseMinerInitParam.Podr2Pbk, os.ModePerm)
					if err != nil {
						out.Err(fmt.Sprintf("write %v to Podr2PubkeyFile failed: %v", responseMinerInitParam.Podr2Pbk, err))
						os.Exit(1)
					}
					break
				}
			}

			if !suc {
				out.Err("All tee nodes are busy or unavailable, program exits.")
				os.Exit(1)
			}

			poisKey, err := sutils.BytesToPoISKeyInfo(n.MinerPoisInfo.KeyG, n.MinerPoisInfo.KeyN)
			if err != nil {
				out.Err(err.Error())
				os.Exit(1)
			}

			teeWorkPubkey, err := sutils.BytesToWorkPublickey([]byte(teeAcc))
			if err != nil {
				out.Err(err.Error())
				os.Exit(1)
			}
			txhash, err := n.RegisterSminerPOISKey(
				poisKey,
				responseMinerInitParam.SignatureWithTeeController[:],
				n.MinerPoisInfo.StatusTeeSign[:],
				teeWorkPubkey,
			)
			if err != nil {
				out.Err(fmt.Sprintf("[%s] Register POIS key failed: %v", txhash, err))
				os.Exit(1)
			}
			time.Sleep(pattern.BlockInterval * 2)
			var count uint8 = 0
			for {
				count++
				if count > 5 {
					out.Err("Invalid chain node: rpc service failure")
					os.Exit(1)
				}
				minerInfo, err = n.QueryStorageMiner(n.GetSignaturePublickey())
				if err != nil {
					time.Sleep(pattern.BlockInterval)
					continue
				}
				if !minerInfo.SpaceProofInfo.HasValue() {
					time.Sleep(pattern.BlockInterval)
					continue
				}
				_, spaceProofInfo = minerInfo.SpaceProofInfo.Unwrap()
				teeSign = []byte(string(minerInfo.TeeSig[:]))
				peerid = []byte(string(minerInfo.PeerId[:]))
				earningsAcc, _ = sutils.EncodePublicKeyAsCessAccount(minerInfo.BeneficiaryAccount[:])
				break
			}
		} else {
			firstReg = false
			_, spaceProofInfo = minerInfo.SpaceProofInfo.Unwrap()
			teeSign = []byte(string(minerInfo.TeeSig[:]))
			peerid = []byte(string(minerInfo.PeerId[:]))
			earningsAcc, _ = sutils.EncodePublicKeyAsCessAccount(minerInfo.BeneficiaryAccount[:])
		}

		n.MinerPoisInfo = &pb.MinerPoisInfo{
			Acc:           []byte(string(spaceProofInfo.Accumulator[:])),
			Front:         int64(spaceProofInfo.Front),
			Rear:          int64(spaceProofInfo.Rear),
			KeyN:          []byte(string(spaceProofInfo.PoisKey.N[:])),
			KeyG:          []byte(string(spaceProofInfo.PoisKey.G[:])),
			StatusTeeSign: teeSign,
		}

		oldDecSpace := minerInfo.DeclarationSpace.Uint64() / pattern.SIZE_1TiB
		if minerInfo.DeclarationSpace.Uint64()%pattern.SIZE_1TiB != 0 {
			oldDecSpace = +1
		}
		newDecSpace := n.GetUseSpace() / pattern.SIZE_1KiB
		if n.GetUseSpace()%pattern.SIZE_1KiB != 0 {
			newDecSpace += 1
		}
		if newDecSpace > oldDecSpace {
			txhash, err := n.IncreaseDeclarationSpace(uint32(newDecSpace - oldDecSpace))
			if err != nil {
				if txhash != "" {
					out.Err(fmt.Sprintf("[%s] %v", txhash, err))
				} else {
					out.Err(err.Error())
				}
				os.Exit(1)
			}
			out.Ok(fmt.Sprintf("Successfully expanded %dTiB space", newDecSpace-oldDecSpace))
			stakingAcc := n.GetStakingAcc()
			if stakingAcc != "" && stakingAcc != n.GetSignatureAcc() {
				accInfo, err := n.QueryAccountInfo(n.GetSignatureAccPulickey())
				if err != nil {
					if err.Error() != pattern.ERR_Empty {
						out.Err(err.Error())
						os.Exit(1)
					}
					out.Err("Failed to expand space: account does not exist or balance is empty")
					os.Exit(1)
				}
				token = (newDecSpace - oldDecSpace) * pattern.StakingStakePerTiB
				incToken, ok := new(big.Int).SetString(fmt.Sprintf("%d%s", token, pattern.TokenPrecision_CESS), 10)
				if !ok {
					out.Err("Failed to calculate staking")
					os.Exit(1)
				}
				if accInfo.Data.Free.CmpAbs(incToken) < 0 {
					out.Err(fmt.Sprintf("Failed to expand space: signature account balance less than %d %s",
						incToken, n.GetTokenSymbol()))
					os.Exit(1)
				}
				txhash, err := n.IncreaseStakingAmount(n.GetSignatureAcc(), incToken)
				if err != nil {
					if txhash != "" {
						out.Err(fmt.Sprintf("[%s] Failed to expand space: %v", txhash, err))
					} else {
						out.Err(fmt.Sprintf("Failed to expand space: %v", err))
					}
					os.Exit(1)
				}
				out.Ok(fmt.Sprintf("Successfully added %dTCESS staking", token))
			}
		}

		if earningsAcc != n.GetEarningsAcc() {
			txhash, err := n.UpdateEarningsAccount(n.GetEarningsAcc())
			if err != nil {
				out.Err(fmt.Sprintf("[%s] Update earnings account: %v", txhash, err))
				os.Exit(1)
			}
			out.Ok(fmt.Sprintf("[%s] Successfully updated earnings account to %s", txhash, n.GetEarningsAcc()))
		}

		if !sutils.CompareSlice(peerid, n.GetPeerPublickey()) {
			var peeridChain pattern.PeerId
			pids := n.GetPeerPublickey()
			for i := 0; i < len(pids); i++ {
				peeridChain[i] = types.U8(pids[i])
			}
			txhash, err := n.UpdateSminerPeerId(peeridChain)
			if err != nil {
				out.Err(fmt.Sprintf("[%s] Update PeerId: %v", txhash, err))
				os.Exit(1)
			}
			out.Ok(fmt.Sprintf("[%s] Successfully updated peer ID to %s", txhash, base58.Encode(n.GetPeerPublickey())))
		}

		err = n.InitPois(
			firstReg,
			int64(spaceProofInfo.Front),
			int64(spaceProofInfo.Rear),
			int64(n.GetUseSpace()*1024),
			32,
			*new(big.Int).SetBytes([]byte(string(spaceProofInfo.PoisKey.N[:]))),
			*new(big.Int).SetBytes([]byte(string(spaceProofInfo.PoisKey.G[:]))),
		)
		if err != nil {
			out.Err(fmt.Sprintf("[Init Pois-2] %v", err))
			os.Exit(1)
		}
	}

	if n.GetPodr2Key().Spk == nil || n.GetPodr2Key().Spk.N == nil {
		buf, err := os.ReadFile(n.DataDir.Podr2PubkeyFile)
		if err != nil {
			out.Err(fmt.Sprintf("[ReadFile Podr2PubkeyFile] %v", err))
			os.Exit(1)
		}
		err = n.SetPublickey(buf)
		if err != nil {
			out.Err("invalid podr2 public key in the file")
			os.Exit(1)
		}
	}

	n.SetInitStage(node.Stage_BuildCache, "[ok] Building cache...")
	// build cache instance
	n.Cache, err = buildCache(n.DataDir.DbDir)
	if err != nil {
		out.Err(fmt.Sprintf("[buildCache] %v", err))
		os.Exit(1)
	}
	n.SetInitStage(node.Stage_BuildCache, "[ok] Build cache completed")

	n.SetInitStage(node.Stage_BuildLog, "[ok] Building log...")
	// build log instance
	n.Logger, err = buildLogs(n.DataDir.LogDir)
	if err != nil {
		out.Err(fmt.Sprintf("[buildLogs] %v", err))
		os.Exit(1)
	}
	n.SetInitStage(node.Stage_BuildLog, "[ok] Build log completed")
	out.Tip(fmt.Sprintf("Workspace: %v", n.Workspace()))

	n.SetInitStage(node.Stage_Complete, "[ok] Initialization completed")

	dirfreeSpace, err := utils.GetDirFreeSpace(n.Workspace())
	if err == nil {
		if dirfreeSpace < pattern.SIZE_1GiB*32 {
			out.Warn("The workspace capacity is less than 32G")
		}
	}
	out.Tip(fmt.Sprintf("Workspace free size: %v", dirfreeSpace))
	// run
	n.Run()
}

func buildConfigFile(cmd *cobra.Command, port int) (confile.Confile, error) {
	var err error
	var conFilePath string
	configpath1, _ := cmd.Flags().GetString("config")
	configpath2, _ := cmd.Flags().GetString("c")
	if configpath1 != "" {
		_, err = os.Stat(configpath1)
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}
		conFilePath = configpath1
	} else if configpath2 != "" {
		_, err = os.Stat(configpath2)
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}
		conFilePath = configpath2
	} else {
		conFilePath = configs.DefaultConfigFile
	}

	cfg := confile.NewConfigfile()
	err = cfg.Parse(conFilePath, port)
	if err == nil {
		return cfg, nil
	} else {
		if configpath1 != "" || configpath2 != "" {
			return cfg, err
		}
	}

	if !strings.Contains(err.Error(), "stat") {
		out.Err(err.Error())
	}

	var istips bool
	var inputReader = bufio.NewReader(os.Stdin)
	var lines string
	var rpc []string
	rpc, err = cmd.Flags().GetStringSlice("rpc")
	if err != nil {
		return cfg, err
	}

	var rpcValus = make([]string, 0)
	if len(rpc) == 0 {
		for {
			if !istips {
				out.Input(fmt.Sprintf("Enter the rpc address of the chain, multiple addresses are separated by spaces, press Enter to skip\nto use [%s, %s] as default rpc address:", configs.DefaultRpcAddr1, configs.DefaultRpcAddr2))
				istips = true
			}
			lines, err = inputReader.ReadString('\n')
			if err != nil {
				out.Err(err.Error())
				continue
			} else {
				lines = strings.ReplaceAll(lines, "\n", "")
			}

			if lines != "" {
				inputrpc := strings.Split(lines, " ")
				for i := 0; i < len(inputrpc); i++ {
					temp := strings.ReplaceAll(inputrpc[i], " ", "")
					if temp != "" {
						rpcValus = append(rpcValus, temp)
					}
				}
			}
			if len(rpcValus) == 0 {
				rpcValus = []string{configs.DefaultRpcAddr1, configs.DefaultRpcAddr2}
			}
			cfg.SetRpcAddr(rpcValus)
			break
		}
	} else {
		cfg.SetRpcAddr(rpc)
	}

	out.Ok(fmt.Sprintf("%v", cfg.GetRpcAddr()))

	var boots []string
	boots, err = cmd.Flags().GetStringSlice("boot")
	if err != nil {
		return cfg, err
	}
	var bootValus = make([]string, 0)
	istips = false
	if len(boots) == 0 {
		for {
			if !istips {
				out.Input(fmt.Sprintf("Enter the boot node address, multiple addresses are separated by spaces, press Enter to skip\nto use [%s] as default boot node address:", configs.DefaultBootNodeAddr))
				istips = true
			}
			lines, err = inputReader.ReadString('\n')
			if err != nil {
				out.Err(err.Error())
				continue
			} else {
				lines = strings.ReplaceAll(lines, "\n", "")
			}

			if lines != "" {
				inputrpc := strings.Split(lines, " ")
				for i := 0; i < len(inputrpc); i++ {
					temp := strings.ReplaceAll(inputrpc[i], " ", "")
					if temp != "" {
						bootValus = append(bootValus, temp)
					}
				}
			}
			if len(bootValus) == 0 {
				bootValus = []string{configs.DefaultBootNodeAddr}
			}
			cfg.SetBootNodes(bootValus)
			break
		}
	} else {
		cfg.SetBootNodes(boots)
	}

	out.Ok(fmt.Sprintf("%v", cfg.GetBootNodes()))

	workspace, err := cmd.Flags().GetString("ws")
	if err != nil {
		return cfg, err
	}
	istips = false
	if workspace == "" {
		for {
			if !istips {
				out.Input(fmt.Sprintf("Enter the workspace path, press Enter to skip to use %s as default workspace:", configs.DefaultWorkspace))
				istips = true
			}
			lines, err = inputReader.ReadString('\n')
			if err != nil {
				out.Err(err.Error())
				continue
			} else {
				workspace = strings.ReplaceAll(lines, "\n", "")
			}
			if workspace != "" {
				if workspace[0] != configs.DefaultWorkspace[0] {
					workspace = ""
					out.Err(fmt.Sprintf("Enter the full path of the workspace starting with %s :", configs.DefaultWorkspace))
					continue
				}
			} else {
				workspace = configs.DefaultWorkspace
			}
			err = cfg.SetWorkspace(workspace)
			if err != nil {
				out.Err(err.Error())
				continue
			}
			break
		}
	} else {
		err = cfg.SetWorkspace(workspace)
		if err != nil {
			return cfg, err
		}
	}

	out.Ok(fmt.Sprintf("%v", cfg.GetWorkspace()))

	var earnings string
	earnings, err = cmd.Flags().GetString("earnings")
	if err != nil {
		return cfg, err
	}
	istips = false
	if earnings == "" {
		for {
			if !istips {
				out.Input("Enter the earnings account, if you have already registered and don't want to update, press Enter to skip:")
				istips = true
			}
			lines, err = inputReader.ReadString('\n')
			if err != nil {
				out.Err(err.Error())
				continue
			}
			earnings = strings.ReplaceAll(lines, "\n", "")
			err = cfg.SetEarningsAcc(earnings)
			if err != nil {
				earnings = ""
				out.Err("Invalid account, please check and re-enter:")
				continue
			}
			break
		}
	} else {
		err = cfg.SetEarningsAcc(earnings)
		if err != nil {
			return cfg, err
		}
	}

	out.Ok(fmt.Sprintf("%v", cfg.GetEarningsAcc()))

	var listenPort int
	listenPort, err = cmd.Flags().GetInt("port")
	if err != nil {
		listenPort, err = cmd.Flags().GetInt("p")
		if err != nil {
			return cfg, err
		}
	}
	istips = false
	if listenPort == 0 {
		for {
			if !istips {
				out.Input(fmt.Sprintf("Enter the service port, press Enter to skip to use %d as default port:", configs.DefaultServicePort))
				istips = true
			}
			lines, err = inputReader.ReadString('\n')
			if err != nil {
				out.Err(err.Error())
				continue
			}
			lines = strings.ReplaceAll(lines, "\n", "")
			if lines == "" {
				listenPort = configs.DefaultServicePort
			} else {
				listenPort, err = strconv.Atoi(lines)
				if err != nil || listenPort < 1024 {
					listenPort = 0
					out.Err("Please enter a number between 1024~65535:")
					continue
				}
			}

			err = cfg.SetServicePort(listenPort)
			if err != nil {
				listenPort = 0
				out.Err("Please enter a number between 1024~65535:")
				continue
			}
			break
		}
	} else {
		err = cfg.SetServicePort(listenPort)
		if err != nil {
			return cfg, err
		}
	}

	out.Ok(fmt.Sprintf("%v", cfg.GetServicePort()))

	useSpace, err := cmd.Flags().GetUint64("space")
	if err != nil {
		useSpace, err = cmd.Flags().GetUint64("s")
		if err != nil {
			return cfg, err
		}
	}
	istips = false
	if useSpace == 0 {
		for {
			if !istips {
				out.Input("Please enter the maximum space used by the storage node in GiB:")
				istips = true
			}
			lines, err = inputReader.ReadString('\n')
			if err != nil {
				out.Err(err.Error())
				continue
			}
			lines = strings.ReplaceAll(lines, "\n", "")
			if lines == "" {
				out.Err("Please enter an integer greater than or equal to 0:")
				continue
			}
			useSpace, err = strconv.ParseUint(lines, 10, 64)
			if err != nil {
				useSpace = 0
				out.Err("Please enter an integer greater than or equal to 0:")
				continue
			}
			cfg.SetUseSpace(useSpace)
			break
		}
	} else {
		cfg.SetUseSpace(useSpace)
	}

	out.Ok(fmt.Sprintf("%v", cfg.GetUseSpace()))

	var priorityTeeList []string
	priorityTeeList, err = cmd.Flags().GetStringSlice("tees")
	if err != nil {
		return cfg, err
	}
	var priorityTeeListValues = make([]string, 0)
	istips = false
	if len(priorityTeeList) == 0 {
		for {
			if !istips {
				out.Input(fmt.Sprintf("Enter priority tee address, multiple addresses are separated by spaces, press Enter to skip:"))
				istips = true
			}
			lines, err = inputReader.ReadString('\n')
			if err != nil {
				out.Err(err.Error())
				continue
			} else {
				lines = strings.ReplaceAll(lines, "\n", "")
			}

			if lines != "" {
				inputrpc := strings.Split(lines, " ")
				for i := 0; i < len(inputrpc); i++ {
					temp := strings.ReplaceAll(inputrpc[i], " ", "")
					if temp != "" {
						priorityTeeListValues = append(priorityTeeListValues, temp)
					}
				}
			}
			cfg.SetBootNodes(priorityTeeListValues)
			break
		}
	} else {
		cfg.SetBootNodes(priorityTeeList)
	}

	out.Ok(fmt.Sprintf("%v", cfg.GetPriorityTeeList()))

	var mnemonic string
	mnemonic, err = cmd.Flags().GetString("mnemonic")
	if err != nil {
		mnemonic, err = cmd.Flags().GetString("m")
	}
	if mnemonic == "" {
		istips = false
		for {
			if !istips {
				out.Input("Please enter the mnemonic of the staking account:")
				istips = true
			}
			pwd, err := gopass.GetPasswdMasked()
			if err != nil {
				if err.Error() == "interrupted" || err.Error() == "interrupt" || err.Error() == "killed" {
					os.Exit(0)
				}
				out.Err("Invalid mnemonic, please check and re-enter:")
				continue
			}
			if len(pwd) == 0 {
				out.Err("The mnemonic you entered is empty, please re-enter:")
				continue
			}
			err = cfg.SetMnemonic(string(pwd))
			if err != nil {
				out.Err("Invalid mnemonic, please check and re-enter:")
				continue
			}
			break
		}
	} else {
		err = cfg.SetMnemonic(mnemonic)
		if err != nil {
			out.Err("invalid mnemonic")
			return cfg, err
		}
	}
	return cfg, nil
}

func buildAuthenticationConfig(cmd *cobra.Command) (confile.Confile, error) {
	var conFilePath string
	configpath1, _ := cmd.Flags().GetString("config")
	configpath2, _ := cmd.Flags().GetString("c")
	if configpath1 != "" {
		conFilePath = configpath1
	} else if configpath2 != "" {
		conFilePath = configpath2
	} else {
		conFilePath = configs.DefaultConfigFile
	}

	cfg := confile.NewConfigfile()
	err := cfg.Parse(conFilePath, 0)
	if err == nil {
		return cfg, err
	}

	if configpath1 != "" || configpath2 != "" {
		return cfg, err
	}

	var istips bool
	var inputReader = bufio.NewReader(os.Stdin)
	var lines string
	var rpc []string
	rpc, err = cmd.Flags().GetStringSlice("rpc")
	if err != nil {
		return cfg, err
	}
	var rpcValus = make([]string, 0)
	if len(rpc) == 0 {
		for {
			if !istips {
				out.Input(fmt.Sprintf("Enter the rpc address of the chain, multiple addresses are separated by spaces, press Enter to skip\nto use [%s, %s] as default rpc address:", configs.DefaultRpcAddr1, configs.DefaultRpcAddr2))
				istips = true
			}
			lines, err = inputReader.ReadString('\n')
			if err != nil {
				out.Err(err.Error())
				continue
			} else {
				lines = strings.ReplaceAll(lines, "\n", "")
			}

			if lines != "" {
				inputrpc := strings.Split(lines, " ")
				for i := 0; i < len(inputrpc); i++ {
					temp := strings.ReplaceAll(inputrpc[i], " ", "")
					if temp != "" {
						rpcValus = append(rpcValus, temp)
					}
				}
			}
			if len(rpcValus) == 0 {
				rpcValus = []string{configs.DefaultRpcAddr1, configs.DefaultRpcAddr2}
			}
			cfg.SetRpcAddr(rpcValus)
			break
		}
	} else {
		cfg.SetRpcAddr(rpc)
	}

	istips = false
	for {
		if !istips {
			out.Input("Please enter the mnemonic of the staking account:")
			istips = true
		}
		pwd, err := gopass.GetPasswdMasked()
		if err != nil {
			if err.Error() == "interrupted" || err.Error() == "interrupt" || err.Error() == "killed" {
				os.Exit(0)
			}
			out.Err("Invalid mnemonic, please check and re-enter:")
			continue
		}
		if len(pwd) == 0 {
			out.Err("The mnemonic you entered is empty, please re-enter:")
			continue
		}
		err = cfg.SetMnemonic(string(pwd))
		if err != nil {
			out.Err("Invalid mnemonic, please check and re-enter:")
			continue
		}
		break
	}
	return cfg, nil
}

func buildDir(workspace string) (*node.DataDir, error) {
	var dir = &node.DataDir{}
	dir.LogDir = filepath.Join(workspace, configs.LogDir)
	if err := os.MkdirAll(dir.LogDir, pattern.DirMode); err != nil {
		return dir, err
	}

	dir.DbDir = filepath.Join(workspace, configs.DbDir)
	if err := os.MkdirAll(dir.DbDir, pattern.DirMode); err != nil {
		return dir, err
	}

	dir.AccDir = filepath.Join(workspace, configs.AccDir)
	if err := os.MkdirAll(dir.AccDir, pattern.DirMode); err != nil {
		return dir, err
	}

	dir.PoisDir = filepath.Join(workspace, configs.PoisDir)
	if err := os.MkdirAll(dir.PoisDir, pattern.DirMode); err != nil {
		return dir, err
	}

	dir.RandomDir = filepath.Join(workspace, configs.RandomDir)
	if err := os.MkdirAll(dir.RandomDir, pattern.DirMode); err != nil {
		return dir, err
	}

	dir.SpaceDir = filepath.Join(workspace, configs.SpaceDir)
	if err := os.MkdirAll(dir.SpaceDir, pattern.DirMode); err != nil {
		return dir, err
	}

	dir.PeersFile = filepath.Join(workspace, configs.PeersFile)
	dir.Podr2PubkeyFile = filepath.Join(workspace, configs.Podr2PubkeyFile)
	return dir, nil
}

func buildCache(cacheDir string) (cache.Cache, error) {
	return cache.NewCache(cacheDir, 0, 0, configs.NameSpaces)
}

func buildLogs(logDir string) (logger.Logger, error) {
	var logs_info = make(map[string]string)
	for _, v := range logger.LogFiles {
		logs_info[v] = filepath.Join(logDir, v+".log")
	}
	return logger.NewLogs(logs_info)
}
