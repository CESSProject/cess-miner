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

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	"github.com/CESSProject/cess-bucket/pkg/cache"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	cess "github.com/CESSProject/cess-go-sdk"
	sconfig "github.com/CESSProject/cess-go-sdk/config"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	p2pgo "github.com/CESSProject/p2p-go"
	"github.com/CESSProject/p2p-go/config"
	"github.com/CESSProject/p2p-go/out"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/howeyc/gopass"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mr-tron/base58/base58"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
)

// runCmd is used to start the service
func runCmd(cmd *cobra.Command, args []string) {
	var (
		firstReg       bool
		err            error
		bootEnv        string
		token          uint64
		protocolPrefix string
		syncSt         pattern.SysSyncState
		n              = node.New()
	)

	ctx := context.Background()

	// parse configuration file
	n.Confile, err = buildConfigFile(cmd, 0)
	if err != nil {
		out.Err(fmt.Sprintf("[buildConfigFile] %v", err))
		os.Exit(1)
	}

	n.SaveCpuCore(configs.SysInit(n.GetUseCpu()))

	out.Tip(fmt.Sprintf("Rpc addresses: %v", n.GetRpcAddr()))

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

	// build client
	n.SDK, err = cess.New(
		ctx,
		sconfig.CharacterName_Bucket,
		cess.ConnectRpcAddrs(n.GetRpcAddr()),
		cess.Mnemonic(n.GetMnemonic()),
		cess.TransactionTimeout(configs.TimeToWaitEvent),
	)
	if err != nil {
		out.Err(fmt.Sprintf("[cess.New] %v", err))
		os.Exit(1)
	}

	n.P2P, err = p2pgo.New(
		ctx,
		p2pgo.ListenPort(n.GetServicePort()),
		p2pgo.Workspace(filepath.Join(n.GetWorkspace(), n.GetSignatureAcc(), n.GetSdkName())),
		p2pgo.BootPeers(n.GetBootNodes()),
		p2pgo.ProtocolPrefix(protocolPrefix),
	)
	if err != nil {
		out.Err(fmt.Sprintf("[p2pgo.New] %v", err))
		os.Exit(1)
	}

	out.Tip(fmt.Sprintf("Local peer id: %s", n.ID().Pretty()))
	out.Tip(fmt.Sprintf("Chain network: %s", n.GetNetworkEnv()))
	out.Tip(fmt.Sprintf("P2P network: %s", bootEnv))

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

	for {
		syncSt, err = n.SyncState()
		if err != nil {
			out.Err("Invalid chain node: rpc service failure")
			os.Exit(1)
		}
		if syncSt.CurrentBlock == syncSt.HighestBlock {
			out.Ok(fmt.Sprintf("Synchronization main chain completed: %d", syncSt.CurrentBlock))
			break
		}
		out.Tip(fmt.Sprintf("In the synchronization main chain: %d ...", syncSt.CurrentBlock))
		time.Sleep(time.Second * time.Duration(utils.Ternary(int64(syncSt.HighestBlock-syncSt.CurrentBlock)*6, 30)))
	}

	n.ExpendersInfo, err = n.Expenders()
	if err != nil {
		if err.Error() == pattern.ERR_Empty {
			out.Err("Invalid chain node: file specification is empty")
		} else {
			out.Err("Invalid chain node: rpc service failure")
		}
		os.Exit(1)
	}

	minerInfo, err := n.QueryStorageMiner(n.GetStakingPublickey())
	if err != nil {
		if err.Error() == pattern.ERR_Empty {
			firstReg = true
			token = n.GetUseSpace() / pattern.SIZE_1KiB
			if n.GetUseSpace()%pattern.SIZE_1KiB != 0 {
				token += 1
			}
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
			token_cess, _ := new(big.Int).SetString(fmt.Sprintf("%d%s", token, pattern.TokenPrecision_CESS), 10)
			if accInfo.Data.Free.CmpAbs(token_cess) < 0 {
				out.Err(fmt.Sprintf("Account balance less than %d %s", token, n.GetTokenSymbol()))
				os.Exit(1)
			}
		} else {
			out.Err(pattern.ERR_RPC_CONNECTION.Error())
			os.Exit(1)
		}
	}

	// build data directory
	n.DataDir, err = buildDir(n.Workspace())
	if err != nil {
		out.Err(fmt.Sprintf("[buildDir] %v", err))
		os.Exit(1)
	}

	// load peers
	err = n.LoadPeersFromDisk(n.DataDir.PeersFile)
	if err != nil {
		n.UpdatePeerFirst()
	}

	var bootPeerID []peer.ID
	for _, b := range boots {
		multiaddr, err := sutils.ParseMultiaddrs(b)
		if err != nil {
			n.Log("err", fmt.Sprintf("[ParseMultiaddrs %v] %v", b, err))
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
			n.Connect(n.GetCtxQueryFromCtxCancel(), *addrInfo)
			bootPeerID = append(bootPeerID, addrInfo.ID)
			n.SavePeer(addrInfo.ID.Pretty(), *addrInfo)
		}
	}

	teelist, err := n.QueryTeeWorkerList()
	if err != nil {
		out.Err(fmt.Sprintf("[QueryTeeWorkerList] %v", err))
		os.Exit(1)
	}

	var teemap = make(map[string]string, 0)
	for _, v := range teelist {
		teemap[base58.Encode(v.Peer_id)] = v.Controller_account
	}

	var suc bool
	if firstReg {
		txhash, err := n.RegisterSminer(n.GetPeerPublickey(), n.GetEarningsAcc(), token)
		if err != nil {
			out.Err(fmt.Sprintf("[%s] Register failed: %v", txhash, err))
			os.Exit(1)
		}
		n.SetEarningsAcc(n.GetEarningsAcc())
		n.RebuildDirs()
		time.Sleep(pattern.BlockInterval)
		var useTee string
		for j := 0; j < 10; j++ {
			if suc {
				break
			}
			for i := 0; i < len(bootPeerID); i++ {
				out.Tip(fmt.Sprintf("Will request miner init param to %v", bootPeerID[i]))
				responseMinerInitParam, err := n.PoisGetMinerInitParamP2P(bootPeerID[i], n.GetSignatureAccPulickey(), time.Duration(time.Second*30))
				if err != nil {
					out.Err(fmt.Sprintf("[PoisGetMinerInitParamP2P] %v", err))
					continue
				}
				n.MinerPoisInfo = &pb.MinerPoisInfo{
					Acc:           responseMinerInitParam.Acc,
					Front:         responseMinerInitParam.Front,
					Rear:          responseMinerInitParam.Rear,
					KeyN:          responseMinerInitParam.KeyN,
					KeyG:          responseMinerInitParam.KeyG,
					StatusTeeSign: responseMinerInitParam.Signature,
				}
				useTee = bootPeerID[i].Pretty()
				suc = true
				break
			}
		}

		if !suc {
			out.Err("All trusted nodes are busy, program exits.")
			os.Exit(1)
		}

		var key pattern.PoISKeyInfo
		if len(n.MinerPoisInfo.KeyG) != len(pattern.PoISKey_G{}) {
			out.Err("invalid tee key_g")
			os.Exit(1)
		}

		if len(n.MinerPoisInfo.KeyN) != len(pattern.PoISKey_N{}) {
			out.Err("invalid tee key_n")
			os.Exit(1)
		}
		for i := 0; i < len(n.MinerPoisInfo.KeyG); i++ {
			key.G[i] = types.U8(n.MinerPoisInfo.KeyG[i])
		}
		for i := 0; i < len(n.MinerPoisInfo.KeyN); i++ {
			key.N[i] = types.U8(n.MinerPoisInfo.KeyN[i])
		}

		if _, ok := teemap[useTee]; !ok {
			out.Err(fmt.Sprintf("Unregistered tee: %v", useTee))
			os.Exit(1)
		}
		pubkey, err := sutils.ParsingPublickey(teemap[useTee])
		if err != nil {
			out.Err(fmt.Sprintf("Invalid account: %s", teemap[useTee]))
			os.Exit(1)
		}
		teeAcc, err := types.NewAccountID(pubkey)
		if err != nil {
			out.Err(fmt.Sprintf("Invalid account: %s", teemap[useTee]))
			os.Exit(1)
		}
		key.Acc = *teeAcc
		var sign pattern.TeeSignature
		if len(n.MinerPoisInfo.StatusTeeSign) != pattern.TeeSignatureLen {
			out.Err("invalid tee signature")
			os.Exit(1)
		}
		for i := 0; i < len(n.MinerPoisInfo.StatusTeeSign); i++ {
			sign[i] = types.U8(n.MinerPoisInfo.StatusTeeSign[i])
		}
		txhash, err = n.RegisterSminerPOISKey(key, sign)
		if err != nil {
			out.Err(fmt.Sprintf("[%s] Register POIS key failed: %v", txhash, err))
			os.Exit(1)
		}
		err = n.InitPois(0, 0, int64(n.GetUseSpace()*1024), 32, *new(big.Int).SetBytes(n.MinerPoisInfo.KeyN), *new(big.Int).SetBytes(n.MinerPoisInfo.KeyG))
		if err != nil {
			out.Err(fmt.Sprintf("[Init Pois] %v", err))
			os.Exit(1)
		}
	} else {
		var spaceProofInfo pattern.SpaceProofInfo
		var teeSign []byte
		var earningsAcc string
		var peerid []byte
		if !minerInfo.SpaceProofInfo.HasValue() {
			var useTee string
			for j := 0; j < 10; j++ {
				if suc {
					break
				}
				for i := 0; i < len(bootPeerID); i++ {
					out.Tip(fmt.Sprintf("Will request miner init param to %v", bootPeerID[i]))
					responseMinerInitParam, err := n.PoisGetMinerInitParamP2P(bootPeerID[i], n.GetSignatureAccPulickey(), time.Duration(time.Second*30))
					if err != nil {
						out.Err(fmt.Sprintf("[PoisGetMinerInitParamP2P] %v", err))
						continue
					}
					n.MinerPoisInfo = &pb.MinerPoisInfo{
						Acc:           responseMinerInitParam.Acc,
						Front:         responseMinerInitParam.Front,
						Rear:          responseMinerInitParam.Rear,
						KeyN:          responseMinerInitParam.KeyN,
						KeyG:          responseMinerInitParam.KeyG,
						StatusTeeSign: responseMinerInitParam.Signature,
					}
					useTee = bootPeerID[i].Pretty()
					suc = true
					break
				}
			}

			if !suc {
				out.Err("All trusted nodes are busy, program exits.")
				os.Exit(1)
			}

			var key pattern.PoISKeyInfo
			if len(n.MinerPoisInfo.KeyG) != len(pattern.PoISKey_G{}) {
				out.Err("invalid tee key_g")
				os.Exit(1)
			}

			if len(n.MinerPoisInfo.KeyN) != len(pattern.PoISKey_N{}) {
				out.Err("invalid tee key_n")
				os.Exit(1)
			}
			for i := 0; i < len(n.MinerPoisInfo.KeyG); i++ {
				key.G[i] = types.U8(n.MinerPoisInfo.KeyG[i])
			}
			for i := 0; i < len(n.MinerPoisInfo.KeyN); i++ {
				key.N[i] = types.U8(n.MinerPoisInfo.KeyN[i])
			}

			if _, ok := teemap[useTee]; !ok {
				out.Err(fmt.Sprintf("Unregistered tee: %v", useTee))
				os.Exit(1)
			}
			pubkey, err := sutils.ParsingPublickey(teemap[useTee])
			if err != nil {
				out.Err(fmt.Sprintf("Invalid account: %s", teemap[useTee]))
				os.Exit(1)
			}
			teeAcc, err := types.NewAccountID(pubkey)
			if err != nil {
				out.Err(fmt.Sprintf("Invalid account: %s", teemap[useTee]))
				os.Exit(1)
			}
			key.Acc = *teeAcc
			var sign pattern.TeeSignature
			if len(n.MinerPoisInfo.StatusTeeSign) != len(pattern.TeeSignature{}) {
				out.Err("invalid tee signature")
				os.Exit(1)
			}
			for i := 0; i < len(n.MinerPoisInfo.StatusTeeSign); i++ {
				sign[i] = types.U8(n.MinerPoisInfo.StatusTeeSign[i])
			}
			txhash, err := n.RegisterSminerPOISKey(key, sign)
			if err != nil {
				out.Err(fmt.Sprintf("[%s] Register POIS key failed: %v", txhash, err))
				os.Exit(1)
			}
			time.Sleep(pattern.BlockInterval)
			var count uint8 = 0
			for {
				count++
				if count > 5 {
					out.Err("Invalid chain node: rpc service failure")
					os.Exit(1)
				}
				minerInfo, err = n.QueryStorageMiner(n.GetStakingPublickey())
				if err != nil {
					time.Sleep(pattern.BlockInterval)
					continue
				}
				if !minerInfo.SpaceProofInfo.HasValue() {
					time.Sleep(pattern.BlockInterval)
					continue
				}
				_, spaceProofInfo = minerInfo.SpaceProofInfo.Unwrap()
				teeSign = []byte(string(minerInfo.TeeSignature[:]))
				peerid = []byte(string(minerInfo.PeerId[:]))
				earningsAcc, _ = sutils.EncodePublicKeyAsCessAccount(minerInfo.BeneficiaryAcc[:])
				break
			}
		} else {
			_, spaceProofInfo = minerInfo.SpaceProofInfo.Unwrap()
			teeSign = []byte(string(minerInfo.TeeSignature[:]))
			peerid = []byte(string(minerInfo.PeerId[:]))
			earningsAcc, _ = sutils.EncodePublicKeyAsCessAccount(minerInfo.BeneficiaryAcc[:])
		}

		n.MinerPoisInfo = &pb.MinerPoisInfo{
			Acc:           []byte(string(spaceProofInfo.Accumulator[:])),
			Front:         int64(spaceProofInfo.Front),
			Rear:          int64(spaceProofInfo.Rear),
			KeyN:          []byte(string(spaceProofInfo.PoisKey.N[:])),
			KeyG:          []byte(string(spaceProofInfo.PoisKey.G[:])),
			StatusTeeSign: teeSign,
		}
		token = n.GetUseSpace() / pattern.SIZE_1KiB
		if n.GetUseSpace()%pattern.SIZE_1KiB != 0 {
			token += 1
		}
		token *= pattern.StakingStakePerTiB
		newToken, _ := new(big.Int).SetString(fmt.Sprintf("%d%s", token, pattern.TokenPrecision_CESS), 10)
		if newToken.Uint64() > minerInfo.Collaterals.Uint64() {
			if newToken.Uint64()-minerInfo.Collaterals.Uint64() >= pattern.StakingStakePerTiB {
				accInfo, err := n.QueryAccountInfo(n.GetSignatureAccPulickey())
				if err != nil {
					if err.Error() != pattern.ERR_Empty {
						out.Err("Invalid chain node: rpc service failure")
						os.Exit(1)
					}
					out.Err("Account does not exist or balance is empty")
					os.Exit(1)
				}
				stakes, ok := new(big.Int).SetString(fmt.Sprintf("%d", newToken.Uint64()-minerInfo.Collaterals.Uint64()), 10)
				if !ok {
					out.Err(fmt.Sprintf("Failed to calculate staking"))
					os.Exit(1)
				}
				if accInfo.Data.Free.CmpAbs(stakes) < 0 {
					out.Err(fmt.Sprintf("Account balance less than %d %s, unable to staking.", (token - minerInfo.Collaterals.Uint64()), n.GetTokenSymbol()))
					os.Exit(1)
				}
				txhash, err := n.IncreaseStakingAmount(stakes)
				if err != nil {
					out.Err(fmt.Sprintf("[%s] Invalid chain node: rpc service failure", txhash))
					os.Exit(1)
				}
			}
		}

		if earningsAcc != n.GetEarningsAcc() {
			txhash, err := n.UpdateEarningsAccount(n.GetEarningsAcc())
			if err != nil {
				out.Err(fmt.Sprintf("[%s] UpdateEarningsAccount: %v", txhash, err))
				os.Exit(1)
			}
		}

		if !sutils.CompareSlice(peerid, n.GetPeerPublickey()) {
			var peeridChain pattern.PeerId
			pids := n.GetPeerPublickey()
			for i := 0; i < len(pids); i++ {
				peeridChain[i] = types.U8(pids[i])
			}
			txhash, err := n.UpdateSminerPeerId(peeridChain)
			if err != nil {
				out.Err(fmt.Sprintf("[%s] UpdateSminerPeerId: %v", txhash, err))
				os.Exit(1)
			}
		}
		n.SetEarningsAcc(n.GetEarningsAcc())
		err = n.InitPois(
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

	// build cache instance
	n.Cache, err = buildCache(n.DataDir.DbDir)
	if err != nil {
		out.Err(fmt.Sprintf("[buildCache] %v", err))
		os.Exit(1)
	}

	// build log instance
	n.Logger, err = buildLogs(n.DataDir.LogDir)
	if err != nil {
		out.Err(fmt.Sprintf("[buildLogs] %v", err))
		os.Exit(1)
	}

	out.Tip(fmt.Sprintf("Workspace: %v", n.Workspace()))

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
