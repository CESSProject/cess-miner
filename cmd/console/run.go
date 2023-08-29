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
	"github.com/CESSProject/cess-go-sdk/config"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/CESSProject/p2p-go/out"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/howeyc/gopass"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
)

// runCmd is used to start the service
func runCmd(cmd *cobra.Command, args []string) {
	var (
		firstReg       bool
		err            error
		logDir         string
		cacheDir       string
		earnings       string
		bootEnv        string
		token          uint64
		syncSt         pattern.SysSyncState
		protocolPrefix string
		n              = node.New()
	)

	// Build profile instances
	n.Confile, err = buildConfigFile(cmd, 0)
	if err != nil {
		out.Err(fmt.Sprintf("[buildConfigFile] %v", err))
		os.Exit(1)
	}
	//out.Ok("Configuration file parsing completed")
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

	//Build client
	n.SDK, err = cess.New(
		context.Background(),
		config.CharacterName_Bucket,
		cess.ConnectRpcAddrs(n.GetRpcAddr()),
		cess.Mnemonic(n.GetMnemonic()),
		cess.TransactionTimeout(configs.TimeToWaitEvent),
		cess.Workspace(n.GetWorkspace()),
		cess.P2pPort(n.GetServicePort()),
		cess.Bootnodes(n.GetBootNodes()),
		cess.ProtocolPrefix(protocolPrefix),
	)
	if err != nil {
		out.Err(fmt.Sprintf("[cess.New] %v", err))
		os.Exit(1)
	}

	// out.Tip(fmt.Sprintf("P2P protocol version: %v", n.GetProtocolVersion()))
	// out.Tip(fmt.Sprintf("DHT protocol version: %v", n.GetDhtProtocolVersion()))
	// out.Tip(fmt.Sprintf("GRPC protocol version: %v", n.GetGrpcProtocolVersion()))
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
			out.Err(err.Error())
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
		out.Err("Weak network signal or rpc service failure")
		os.Exit(1)
	}

	minerInfo_V2, err := n.QueryStorageMiner_V2(n.GetStakingPublickey())
	if err != nil {
		if err.Error() == pattern.ERR_Empty {
			firstReg = true
			token = n.GetUseSpace() / 1024
			if n.GetUseSpace()%1024 != 0 {
				token += 1
			}
			token *= 2000
			accInfo, err := n.QueryAccountInfo(n.GetSignatureAccPulickey())
			if err != nil {
				if err.Error() != pattern.ERR_Empty {
					out.Err("Weak network signal or rpc service failure")
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

	if firstReg {
		var bootPeerID []peer.ID
		var minerInitParam *pb.ResponseMinerInitParam
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
				err = n.Connect(n.GetCtxQueryFromCtxCancel(), *addrInfo)
				if err != nil {
					continue
				}
				bootPeerID = append(bootPeerID, addrInfo.ID)
				n.SavePeer(addrInfo.ID.Pretty(), *addrInfo)
			}
		}

		for i := 0; i < len(bootPeerID); i++ {
			minerInitParam, err = n.PoisGetMinerInitParamP2P(bootPeerID[i], n.GetSignatureAccPulickey(), time.Duration(time.Second*15))
			if err != nil {
				out.Err(fmt.Sprintf("[PoisGetMinerInitParam] %v", err))
				continue
			}
			//out.Ok("Get the initial proof key")
			break
		}

		var key pattern.PoISKeyInfo
		if len(minerInitParam.KeyG) != len(pattern.PoISKey_G{}) {
			out.Err("invalid tee key_g")
			os.Exit(1)
		}

		if len(minerInitParam.KeyN) != len(pattern.PoISKey_N{}) {
			out.Err("invalid tee key_n")
			os.Exit(1)
		}
		for i := 0; i < len(minerInitParam.KeyG); i++ {
			key.G[i] = types.U8(minerInitParam.KeyG[i])
		}
		for i := 0; i < len(minerInitParam.KeyN); i++ {
			key.N[i] = types.U8(minerInitParam.KeyN[i])
		}
		var sign pattern.TeeSignature
		if len(minerInitParam.Signature) != len(pattern.TeeSignature{}) {
			out.Err("invalid tee signature")
			os.Exit(1)
		}
		for i := 0; i < len(minerInitParam.Signature); i++ {
			sign[i] = types.U8(minerInitParam.Signature[i])
		}

		//out.Tip("Start registering storage node")
		_, earnings, err = n.RegisterOrUpdateSminer_V2(n.GetPeerPublickey(), n.GetEarningsAcc(), token, key, sign)
		if err != nil {
			out.Err(fmt.Sprintf("Register failed: %v", err))
			os.Exit(1)
		}
		n.SetEarningsAcc(earnings)
		n.RebuildDirs()
		err = n.InitPois(0, 0, int64(n.GetUseSpace()*1024), 32, *new(big.Int).SetBytes(minerInitParam.KeyN), *new(big.Int).SetBytes(minerInitParam.KeyG))
		if err != nil {
			out.Err(fmt.Sprintf("[Init Pois] %v", err))
			os.Exit(1)
		}
	} else {
		//out.Tip("Update storage node information")
		_, earnings, err = n.RegisterOrUpdateSminer_V2(n.GetPeerPublickey(), n.GetEarningsAcc(), 0, pattern.PoISKeyInfo{}, pattern.TeeSignature{})
		if err != nil {
			out.Err(fmt.Sprintf("Update failed: %v", err))
			os.Exit(1)
		}
		n.SetEarningsAcc(earnings)
		err = n.InitPois(int64(minerInfo_V2.SpaceProofInfo.Front), int64(minerInfo_V2.SpaceProofInfo.Rear), int64(n.GetUseSpace()*1024), 32, *new(big.Int).SetBytes([]byte(string(minerInfo_V2.SpaceProofInfo.PoisKey.N[:]))), *new(big.Int).SetBytes([]byte(string(minerInfo_V2.SpaceProofInfo.PoisKey.G[:]))))
		if err != nil {
			out.Err(fmt.Sprintf("[Init Pois-2] %v", err))
			os.Exit(1)
		}
	}

	// Build data directory
	logDir, cacheDir, err = buildDir(n.Workspace())
	if err != nil {
		out.Err(fmt.Sprintf("[buildDir] %v", err))
		os.Exit(1)
	}
	//out.Tip("Initialize the workspace")

	// Build cache instance
	n.Cache, err = buildCache(cacheDir)
	if err != nil {
		out.Err(fmt.Sprintf("[buildCache] %v", err))
		os.Exit(1)
	}
	//out.Tip("Initialize the cache")

	//Build log instance
	n.Logger, err = buildLogs(logDir)
	if err != nil {
		out.Err(fmt.Sprintf("[buildLogs] %v", err))
		os.Exit(1)
	}
	//out.Tip("Initialize the logger")

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

func buildDir(workspace string) (string, string, error) {
	logDir := filepath.Join(workspace, configs.LogDir)
	if err := os.MkdirAll(logDir, pattern.DirMode); err != nil {
		return "", "", err
	}

	cacheDir := filepath.Join(workspace, configs.DbDir)
	if err := os.MkdirAll(cacheDir, pattern.DirMode); err != nil {
		return "", "", err
	}

	return logDir, cacheDir, nil
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
