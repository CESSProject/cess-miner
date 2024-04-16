/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"bufio"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	"github.com/CESSProject/cess-bucket/pkg/cache"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/proof"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	sdkgo "github.com/CESSProject/cess-go-sdk"
	"github.com/CESSProject/cess-go-sdk/chain"
	sconfig "github.com/CESSProject/cess-go-sdk/config"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/cess-go-sdk/core/sdk"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/cess_pois/acc"
	"github.com/CESSProject/cess_pois/pois"
	p2pgo "github.com/CESSProject/p2p-go"
	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/p2p-go/out"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/howeyc/gopass"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// runCmd is used to start the service
func runCmd(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	cfg := confile.NewEmptyConfigfile()
	cli := sdk.SDK(&chain.ChainClient{})
	peernode := &core.PeerNode{}
	runningState := node.NewRunningState()
	teeRecord := &node.TeeRecord{}
	minerState := &node.MinerState{}
	wspace := node.NewWorkspace()
	minerPoisInfo := &pb.MinerPoisInfo{}
	rsaKeyPair := &proof.RSAKeyPair{}
	p := &node.Pois{}

	runningState.SetCpuCores(configs.SysInit(cfg.ReadUseCpu()))
	runningState.SetInitStage(node.Stage_Startup, "Service startup")
	runningState.SetPID(int32(os.Getpid()))
	//n.ListenLocal()

	runningState.SetInitStage(node.Stage_ReadConfig, "Parsing configuration file...")
	// parse configuration file
	config_file, err := parseArgs_config(cmd)
	if err != nil {
		cfg, err = buildConfigItems(cmd)
		if err != nil {
			out.Err(fmt.Sprintf("build config items err: %v", err))
			os.Exit(1)
		}
	} else {
		cfg, err = parseConfigFile(config_file)
		if err != nil {
			out.Err(fmt.Sprintf("parse config file err: %v", err))
			os.Exit(1)
		}
	}

	runningState.SetInitStage(node.Stage_ReadConfig, "[ok] Read configuration file")
	runningState.SetInitStage(node.Stage_ConnectRpc, "Connecting to rpc...")

	// new chain client
	cli, err = sdkgo.New(
		ctx,
		sdkgo.Name(sconfig.CharacterName_Bucket),
		sdkgo.ConnectRpcAddrs(cfg.ReadRpcEndpoints()),
		sdkgo.Mnemonic(cfg.ReadMnemonic()),
		sdkgo.TransactionTimeout(configs.TimeToWaitEvent),
	)
	if err != nil {
		out.Err(fmt.Sprintf("[sdkgo.New] %v", err))
		os.Exit(1)
	}
	defer cli.Close()

	runningState.SetInitStage(node.Stage_ConnectRpc, fmt.Sprintf("[ok] Connect rpc: %s", cli.GetCurrentRpcAddr()))
	runningState.SetInitStage(node.Stage_CreateP2p, "Create peer node...")

	// new peer node
	peernode, err = p2pgo.New(
		ctx,
		p2pgo.ListenPort(cfg.ReadServicePort()),
		p2pgo.Workspace(filepath.Join(cfg.ReadWorkspace(), cli.GetSignatureAcc(), cli.GetSDKName())),
		p2pgo.BootPeers(cfg.ReadBootnodes()),
	)
	if err != nil {
		out.Err(fmt.Sprintf("[p2pgo.New] %v", err))
		os.Exit(1)
	}
	defer peernode.Close()

	// check network environment
	err = checkNetworkEnv(cli, peernode.GetBootnode())
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}

	runningState.SetInitStage(node.Stage_CreateP2p, fmt.Sprintf("[ok] Create peer node: %s", peernode.ID().String()))

	out.Tip(fmt.Sprintf("Local peer id: %s", peernode.ID().String()))
	out.Tip(fmt.Sprintf("Network environment: %s", cli.GetNetworkEnv()))
	out.Tip(fmt.Sprintf("Number of cpu cores used: %v", runningState.GetCpuCores()))
	out.Tip(fmt.Sprintf("RPC endpoint used: %v", cli.GetCurrentRpcAddr()))

	runningState.SetInitStage(node.Stage_SyncBlock, "Waiting to synchronize the main chain...")

	err = checkRpcSynchronization(cli)
	if err != nil {
		out.Err("Failed to synchronize the main chain: network connection is down")
		os.Exit(1)
	}

	register, decTib, oldRegInfo, err := checkRegistrationInfo(cfg, cli)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}

	switch register {
	case configs.Unregistered:
		runningState.SetInitStage(node.Stage_Register, "[ok] Registering...")
		_, err = registerMiner(cfg, cli, peernode, decTib)
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}
		runningState.SetInitStage(node.Stage_Register, "[ok] Registration complete")
		runningState.SetInitStage(node.Stage_BuildDir, "[ok] Build workspace...")
		err = wspace.RemoveAndBuild(peernode.Workspace())
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}
		runningState.SetInitStage(node.Stage_BuildDir, "[ok] Build workspace completed")

		time.Sleep(pattern.BlockInterval * 5)

		err = registerPoisKey(cfg, cli, peernode, teeRecord, minerPoisInfo, rsaKeyPair, wspace)
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}

		err = InitPOIS(
			p, cli, wspace, true, 0, 0,
			int64(cfg.ReadUseSpace()*1024), 32,
			*new(big.Int).SetBytes(minerPoisInfo.KeyN),
			*new(big.Int).SetBytes(minerPoisInfo.KeyG),
			runningState.GetCpuCores(),
		)
		if err != nil {
			out.Err(fmt.Sprintf("[Init Pois] %v", err))
			os.Exit(1)
		}
	case configs.UnregisteredPoisKey:
		minerState.SaveMinerState(string(oldRegInfo.State))
		runningState.SetInitStage(node.Stage_Register, "[ok] Registering pois key...")
		err = registerPoisKey(cfg, cli, peernode, teeRecord, minerPoisInfo, rsaKeyPair, wspace)
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}

		err = updateMinerRegistertionInfo(cfg, cli, oldRegInfo)
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}

		err = InitPOIS(
			p, cli, wspace, true, 0, 0,
			int64(cfg.ReadUseSpace()*1024), 32,
			*new(big.Int).SetBytes(minerPoisInfo.KeyN),
			*new(big.Int).SetBytes(minerPoisInfo.KeyG),
			runningState.GetCpuCores(),
		)
		if err != nil {
			out.Err(fmt.Sprintf("[Init Pois] %v", err))
			os.Exit(1)
		}
		err = wspace.Build(peernode.Workspace())
		if err != nil {
			out.Err(fmt.Sprintf("build workspace err: %v", err))
			os.Exit(1)
		}

	case configs.Registered:
		minerState.SaveMinerState(string(oldRegInfo.State))
		err = updateMinerRegistertionInfo(cfg, cli, oldRegInfo)
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}
		_, spaceProofInfo := oldRegInfo.SpaceProofInfo.Unwrap()
		minerPoisInfo.Acc = []byte(string(spaceProofInfo.Accumulator[:]))
		minerPoisInfo.Front = int64(spaceProofInfo.Front)
		minerPoisInfo.Rear = int64(spaceProofInfo.Rear)
		minerPoisInfo.KeyN = []byte(string(spaceProofInfo.PoisKey.N[:]))
		minerPoisInfo.KeyG = []byte(string(spaceProofInfo.PoisKey.G[:]))
		minerPoisInfo.StatusTeeSign = []byte(string(oldRegInfo.TeeSig[:]))

		err = InitPOIS(
			p, cli, wspace, false,
			int64(spaceProofInfo.Front),
			int64(spaceProofInfo.Rear),
			int64(cfg.ReadUseSpace()*1024), 32,
			*new(big.Int).SetBytes(minerPoisInfo.KeyN),
			*new(big.Int).SetBytes(minerPoisInfo.KeyG),
			runningState.GetCpuCores(),
		)
		if err != nil {
			out.Err(fmt.Sprintf("Init POIS err: %v", err))
			os.Exit(1)
		}
		err = wspace.Build(peernode.Workspace())
		if err != nil {
			out.Err(fmt.Sprintf("build workspace err: %v", err))
			os.Exit(1)
		}

		saveAllTees(cli, peernode, teeRecord)

		buf, err := wspace.LoadRsaPublicKey()
		if err != nil {
			buf, _ = queryPodr2KeyFromTee(peernode, teeRecord, cli.GetSignatureAccPulickey())
		}
		if len(buf) > 0 {
			rsaKeyPair, err = InitRsaKey(buf)
			if err != nil {
				out.Err(fmt.Sprintf("Init rsa public key err: %v", err))
				os.Exit(1)
			}
		}
		spaceProofInfo = pattern.SpaceProofInfo{}
		buf = nil

	default:
		out.Err("system err")
		os.Exit(1)
	}
	oldRegInfo = nil

	runningState.SetInitStage(node.Stage_BuildCache, "[ok] Building cache...")
	// build cache instance
	cace, err := buildCache(wspace.GetDbDir())
	if err != nil {
		out.Err(fmt.Sprintf("[buildCache] %v", err))
		os.Exit(1)
	}
	runningState.SetInitStage(node.Stage_BuildCache, "[ok] Build cache completed")

	runningState.SetInitStage(node.Stage_BuildLog, "[ok] Building log...")
	// build log instance
	l, err := buildLogs(wspace.GetLogDir())
	if err != nil {
		out.Err(fmt.Sprintf("[buildLogs] %v", err))
		os.Exit(1)
	}
	runningState.SetInitStage(node.Stage_BuildLog, "[ok] Build log completed")
	out.Tip(fmt.Sprintf("Workspace: %v", wspace.GetRootDir()))

	runningState.SetInitStage(node.Stage_Complete, "[ok] Initialization completed")

	checkWorkSpace(*wspace)

	go subscribe(ctx, peernode.GetBootnode(), peernode.GetHost())

	tick_block := time.NewTicker(pattern.BlockInterval)
	defer tick_block.Stop()

	//node.Run(ctx, cli, peernode, cache, logger)
	out.Ok("Service started successfully")
	chainState := true
	reportFileCh := make(chan bool, 1)
	reportFileCh <- true
	for range tick_block.C {
		chainState = cli.GetChainState()
		if !chainState {
			peernode.DisableRecv()
			connectChain(cli)
			continue
		}
		peernode.EnableRecv()
		syncMinerStatus(cli, l, minerState)
		if minerState.GetMinerState() == pattern.MINER_STATE_EXIT ||
			minerState.GetMinerState() == pattern.MINER_STATE_OFFLINE {
			continue
		}

		if len(reportFileCh) > 0 {
			<-reportFileCh
			go node.ReportFiles(reportFileCh, cli, runningState, wspace, l)
		}

		n.SetTaskPeriod("1m")
		if len(ch_idlechallenge) > 0 || len(ch_servicechallenge) > 0 {
			go n.challengeMgt(ch_idlechallenge, ch_servicechallenge)
		}
		if len(ch_findPeers) > 0 {
			<-ch_findPeers
			go n.subscribe(ctx, ch_findPeers)
		}

		if len(ch_replace) > 0 {
			<-ch_replace
			go n.replaceIdle(ch_replace)
		}

		if len(ch_spaceMgt) > 0 {
			<-ch_spaceMgt
			go n.poisMgt(ch_spaceMgt)
		}

		n.SetTaskPeriod("1m-end")

		if len(ch_syncChainStatus) > 0 {
			<-ch_syncChainStatus
			go n.syncChainStatus(ch_syncChainStatus)
		}

		if len(ch_GenIdleFile) > 0 {
			<-ch_GenIdleFile
			go n.genIdlefile(ch_GenIdleFile)
		}
		if len(ch_calctag) > 0 {
			<-ch_calctag
			go n.calcTag(ch_calctag)
		}

		n.SetTaskPeriod("1h")
		//go n.reportLogsMgt(ch_reportLogs)
		if len(ch_restoreMgt) > 0 {
			<-ch_restoreMgt
			go n.restoreMgt(ch_restoreMgt)
		}
		n.SetTaskPeriod("1h-end")

	}
}

func parseArgs_config(cmd *cobra.Command) (string, error) {
	var err error
	configFile, _ := cmd.Flags().GetString("config")
	if configFile != "" {
		_, err = os.Stat(configFile)
		if err != nil {
			return "", err
		}
		return configFile, nil
	}
	configFile, _ = cmd.Flags().GetString("c")
	if configFile != "" {
		_, err = os.Stat(configFile)
		if err != nil {
			return "", err
		}
		return configFile, nil
	}
	_, err = os.Stat(configs.DefaultConfigFile)
	if err != nil {
		return "", err
	}
	return configs.DefaultConfigFile, nil
}

func parseConfigFile(file string) (*confile.Confile, error) {
	cfg := confile.NewEmptyConfigfile()
	err := cfg.Parse(file)
	return cfg, err
}

func buildConfigItems(cmd *cobra.Command) (*confile.Confile, error) {
	var (
		istips      bool
		lines       string
		rpc         []string
		cfg         = confile.NewEmptyConfigfile()
		inputReader = bufio.NewReader(os.Stdin)
	)
	rpc, err := cmd.Flags().GetStringSlice("rpc")
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

	out.Ok(fmt.Sprintf("%v", cfg.ReadRpcEndpoints()))

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

	out.Ok(fmt.Sprintf("%v", cfg.ReadBootnodes()))

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

	out.Ok(fmt.Sprintf("%v", cfg.ReadWorkspace()))

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

	out.Ok(fmt.Sprintf("%v", cfg.ReadEarningsAcc()))

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

	out.Ok(fmt.Sprintf("%v", cfg.ReadServicePort()))

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

	out.Ok(fmt.Sprintf("%v", cfg.ReadUseSpace()))

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

	out.Ok(fmt.Sprintf("%v", cfg.ReadPriorityTeeList()))

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

func buildAuthenticationConfig(cmd *cobra.Command) (*confile.Confile, error) {
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

	cfg := confile.NewEmptyConfigfile()
	err := cfg.Parse(conFilePath)
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

func checkNetworkEnv(cli sdk.SDK, bootnode string) error {
	chain := cli.GetNetworkEnv()
	if strings.Contains(chain, configs.DevNet) {
		if !strings.Contains(bootnode, configs.DevNet) {
			return errors.New("chain and p2p are not in the same network")
		}
	} else if strings.Contains(chain, configs.TestNet) {
		if !strings.Contains(bootnode, configs.TestNet) {
			return errors.New("chain and p2p are not in the same network")
		}
	} else if strings.Contains(chain, configs.MainNet) {
		if !strings.Contains(bootnode, configs.MainNet) {
			return errors.New("chain and p2p are not in the same network")
		}
	} else {
		return errors.New("unknown chain network")
	}

	chainVersion, err := cli.ChainVersion()
	if err != nil {
		return errors.New("Failed to read chain version: network connection is down")
	}

	if strings.Contains(chain, configs.TestNet) {
		if !strings.Contains(chainVersion, configs.ChainVersion) {
			return fmt.Errorf("The chain version you are using is not %s, please check your rpc service", configs.ChainVersion)
		}
	}

	return nil
}

func checkRpcSynchronization(cli sdk.SDK) error {
	out.Tip("Waiting to synchronize the main chain...")
	var err error
	var syncSt pattern.SysSyncState
	for {
		syncSt, err = cli.SyncState()
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

func checkRegistrationInfo(cfg *confile.Confile, cli sdk.SDK) (int, uint64, *pattern.MinerInfo, error) {
	minerInfo, err := cli.QueryStorageMiner(cli.GetSignatureAccPulickey())
	if err != nil {
		if err.Error() != pattern.ERR_Empty {
			return configs.Unregistered, 0, &minerInfo, err
		}
		decTib := cfg.ReadUseSpace() / pattern.SIZE_1KiB
		if cfg.ReadUseSpace()%pattern.SIZE_1KiB != 0 {
			decTib += 1
		}
		token := decTib * pattern.StakingStakePerTiB
		accInfo, err := cli.QueryAccountInfo(cli.GetSignatureAccPulickey())
		if err != nil {
			if err.Error() != pattern.ERR_Empty {
				return configs.Unregistered, decTib, &minerInfo, fmt.Errorf("Failed to query signature account information: ", err)
			}
			return configs.Unregistered, decTib, &minerInfo, fmt.Errorf("Signature account does not exist, possible: 1.balance is empty 2.rpc address error")
		}
		token_cess, _ := new(big.Int).SetString(fmt.Sprintf("%d%s", token, pattern.TokenPrecision_CESS), 10)
		if cfg.ReadStakingAcc() == "" || cfg.ReadStakingAcc() == cfg.ReadSignatureAccount() {
			if accInfo.Data.Free.CmpAbs(token_cess) < 0 {
				return configs.Unregistered, decTib, &minerInfo, fmt.Errorf("Signature account balance less than %d %s", token, cli.GetTokenSymbol())
			}
		} else {
			stakingAccInfo, err := cli.QueryAccountInfoByAccount(cfg.ReadStakingAcc())
			if err != nil {
				if err.Error() != pattern.ERR_Empty {
					return configs.Unregistered, decTib, &minerInfo, fmt.Errorf("Failed to query staking account information: ", err)
				}
				return configs.Unregistered, decTib, &minerInfo, fmt.Errorf("Staking account does not exist, possible: 1.balance is empty 2.rpc address error")
			}
			if stakingAccInfo.Data.Free.CmpAbs(token_cess) < 0 {
				return configs.Unregistered, decTib, &minerInfo, fmt.Errorf("Staking account balance less than %d %s", token, cli.GetTokenSymbol())
			}
		}
		return configs.Unregistered, decTib, &minerInfo, nil
	}
	if !minerInfo.SpaceProofInfo.HasValue() {
		return configs.UnregisteredPoisKey, 0, &minerInfo, nil
	}
	return configs.Registered, 0, &minerInfo, nil
}

func registerMiner(cfg *confile.Confile, cli sdk.SDK, peernode *core.PeerNode, decTib uint64) (string, error) {
	stakingAcc := cfg.ReadStakingAcc()
	if stakingAcc != "" && stakingAcc != cli.GetSignatureAcc() {
		out.Ok(fmt.Sprintf("Specify staking account: %s", stakingAcc))
		txhash, err := cli.RegisterSminerAssignStaking(cfg.ReadEarningsAcc(), peernode.GetPeerPublickey(), stakingAcc, uint32(decTib))
		if err != nil {
			if txhash != "" {
				err = fmt.Errorf("[%s] %v", txhash, err)
			}
			return txhash, err
		}
		out.Ok(fmt.Sprintf("Storage node registration successful: %s", txhash))
		return txhash, nil
	}

	txhash, err := cli.RegisterSminer(cfg.ReadEarningsAcc(), peernode.GetPeerPublickey(), uint64(decTib*pattern.StakingStakePerTiB), uint32(decTib))
	if err != nil {
		if txhash != "" {
			err = fmt.Errorf("[%s] %v", txhash, err)
		}
		return txhash, err
	}
	out.Ok(fmt.Sprintf("Storage node registration successful: %s", txhash))
	return txhash, nil
}

func saveAllTees(cli sdk.SDK, peernode *core.PeerNode, teeRecord *node.TeeRecord) error {
	var (
		err            error
		teeList        []pattern.TeeWorkerInfo
		dialOptions    []grpc.DialOption
		chainPublickey = make([]byte, pattern.WorkerPublicKeyLen)
	)
	for {
		teeList, err = cli.QueryAllTeeWorkerMap()
		if err != nil {
			if err.Error() == pattern.ERR_Empty {
				out.Err("No tee found, waiting for the next minute's query...")
				time.Sleep(time.Minute)
				continue
			}
			return err
		}
		break
	}

	for _, v := range teeList {
		out.Tip(fmt.Sprintf("Checking the tee: %s", hex.EncodeToString([]byte(string(v.Pubkey[:])))))
		endPoint, err := cli.QueryTeeWorkEndpoint(v.Pubkey)
		if err != nil {
			out.Err(fmt.Sprintf("Failed to query endpoints for this tee: %v", err))
			continue
		}
		endPoint = node.ProcessTeeEndpoint(endPoint)
		if !strings.Contains(endPoint, "443") {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		} else {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
		}
		// verify identity public key
		identityPubkeyResponse, err := peernode.GetIdentityPubkey(endPoint,
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
		if len(identityPubkeyResponse.Pubkey) != pattern.WorkerPublicKeyLen {
			out.Err(fmt.Sprintf("The identity pubkey length of this tee is incorrect: %d", len(identityPubkeyResponse.Pubkey)))
			continue
		}
		for j := 0; j < pattern.WorkerPublicKeyLen; j++ {
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
	return nil
}

func registerPoisKey(
	cfg *confile.Confile,
	cli sdk.SDK,
	peernode *core.PeerNode,
	teeRecord *node.TeeRecord,
	minerPoisInfo *pb.MinerPoisInfo,
	rsaKeyPair *proof.RSAKeyPair,
	key *node.Workspace) error {
	var (
		err                    error
		teeList                []pattern.TeeWorkerInfo
		dialOptions            []grpc.DialOption
		responseMinerInitParam *pb.ResponseMinerInitParam
		chainPublickey         = make([]byte, pattern.WorkerPublicKeyLen)
		teeEndPointList        = cfg.ReadPriorityTeeList()
	)
	for {
		teeList, err = cli.QueryAllTeeWorkerMap()
		if err != nil {
			if err.Error() == pattern.ERR_Empty {
				out.Err("No tee found, waiting for the next minute's query...")
				time.Sleep(time.Minute)
				continue
			}
			return err
		}
		break
	}

	for _, v := range teeList {
		out.Tip(fmt.Sprintf("Checking the tee: %s", hex.EncodeToString([]byte(string(v.Pubkey[:])))))
		endPoint, err := cli.QueryTeeWorkEndpoint(v.Pubkey)
		if err != nil {
			out.Err(fmt.Sprintf("Failed to query endpoints for this tee: %v", err))
			continue
		}
		endPoint = node.ProcessTeeEndpoint(endPoint)
		if !strings.Contains(endPoint, "443") {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		} else {
			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
		}
		// verify identity public key
		identityPubkeyResponse, err := peernode.GetIdentityPubkey(endPoint,
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
		if len(identityPubkeyResponse.Pubkey) != pattern.WorkerPublicKeyLen {
			out.Err(fmt.Sprintf("The identity pubkey length of this tee is incorrect: %d", len(identityPubkeyResponse.Pubkey)))
			continue
		}
		for j := 0; j < pattern.WorkerPublicKeyLen; j++ {
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
			out.Tip(fmt.Sprintf("Requesting registration parameters from tee: %s", teeEndPointList[i]))
			if !strings.Contains(teeEndPointList[i], "443") {
				dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
			} else {
				dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
			}
			responseMinerInitParam, err = peernode.RequestMinerGetNewKey(
				teeEndPointList[i],
				cli.GetSignatureAccPulickey(),
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
					time.Sleep(pattern.BlockInterval * 2)
					continue
				}
				out.Err(fmt.Sprintf("Request err: %v", err))
				break
			}

			rsaKeyPair, err = InitRsaKey(responseMinerInitParam.Podr2Pbk)
			if err != nil {
				out.Err(fmt.Sprintf("Request err: %v", err))
				break
			}

			err = key.SaveRsaPublicKey(responseMinerInitParam.Podr2Pbk)
			if err != nil {
				out.Err(fmt.Sprintf("Save rsa public key err: %v", err))
				break
			}

			minerPoisInfo.Acc = responseMinerInitParam.Acc
			minerPoisInfo.Front = responseMinerInitParam.Front
			minerPoisInfo.Rear = responseMinerInitParam.Rear
			minerPoisInfo.KeyN = responseMinerInitParam.KeyN
			minerPoisInfo.KeyG = responseMinerInitParam.KeyG
			minerPoisInfo.StatusTeeSign = responseMinerInitParam.StatusTeeSign

			workpublickey, err := teeRecord.GetTeeWorkAccount(teeEndPointList[i])
			if err != nil {
				break
			}

			poisKey, err := sutils.BytesToPoISKeyInfo(responseMinerInitParam.KeyG, responseMinerInitParam.KeyN)
			if err != nil {
				out.Err(fmt.Sprintf("Request err: %v", err))
				continue
			}

			teeWorkPubkey, err := sutils.BytesToWorkPublickey([]byte(workpublickey))
			if err != nil {
				out.Err(fmt.Sprintf("Request err: %v", err))
				continue
			}

			err = registerMinerPoisKey(
				cli,
				poisKey,
				responseMinerInitParam.SignatureWithTeeController[:],
				responseMinerInitParam.StatusTeeSign,
				teeWorkPubkey,
			)
			if err != nil {
				out.Err(fmt.Sprintf("Register miner pois key err: %v", err))
				break
			}
			return nil
		}
	}
	return errors.New("All tee nodes are busy or unavailable")
}

func registerMinerPoisKey(cli sdk.SDK, poisKey pattern.PoISKeyInfo, teeSignWithAcc types.Bytes, teeSign types.Bytes, teePuk pattern.WorkerPublicKey) error {
	var err error
	for i := 0; i < 3; i++ {
		_, err = cli.RegisterSminerPOISKey(
			poisKey,
			teeSignWithAcc,
			teeSign,
			teePuk,
		)
		if err != nil {
			time.Sleep(pattern.BlockInterval * 2)
			minerInfo, err := cli.QueryStorageMiner(cli.GetSignatureAccPulickey())
			if err != nil {
				return err
			}
			if minerInfo.SpaceProofInfo.HasValue() {
				return nil
			}
			continue
		}
	}
	return err
}

func InitPOIS(p *node.Pois, cli sdk.SDK, wspace *node.Workspace, register bool, front, rear, freeSpace, count int64, key_n, key_g big.Int, cpus int) error {
	var err error
	expendersInfo, err := cli.QueryExpenders()
	if err != nil {
		return err
	}
	if p == nil {
		p = &node.Pois{}
	}
	p.ExpendersInfo = expendersInfo

	if len(key_n.Bytes()) != len(pattern.PoISKey_N{}) {
		return errors.New("invalid key_n length")
	}

	if len(key_g.Bytes()) != len(pattern.PoISKey_G{}) {
		return errors.New("invalid key_g length")
	}

	p.RsaKey = &acc.RsaKey{N: key_n, G: key_g}
	p.Front = front
	p.Rear = rear
	cfg := pois.Config{
		AccPath:        wspace.GetPoisDir(),
		IdleFilePath:   wspace.GetSpaceDir(),
		ChallAccPath:   wspace.GetPoisAccDir(),
		MaxProofThread: cpus,
	}

	// k,n,d and key are params that needs to be negotiated with the verifier in advance.
	// minerID is storage node's account ID, and space is the amount of physical space available(MiB)
	p.Prover, err = pois.NewProver(
		int64(expendersInfo.K),
		int64(expendersInfo.N),
		int64(expendersInfo.D),
		cli.GetSignatureAccPulickey(),
		freeSpace,
		count,
	)
	if err != nil {
		return err
	}
	if register {
		//Please initialize prover for the first time
		err = p.Prover.Init(*p.RsaKey, cfg)
		if err != nil {
			return err
		}
	} else {
		// If it is downtime recovery, call the recovery method.front and rear are read from minner info on chain
		err = p.Prover.Recovery(*p.RsaKey, front, rear, cfg)
		if err != nil {
			if strings.Contains(err.Error(), "read element data") {
				num := 2
				m, err := utils.GetSysMemAvailable()
				cpuNum := runtime.NumCPU()
				if err == nil {
					m = m * 7 / 10 / (2 * 1024 * 1024 * 1024)
					if int(m) < cpuNum {
						cpuNum = int(m)
					}
					if cpuNum > num {
						num = cpuNum
					}
				}
				log.Println("check and restore idle data, use", num, "threads")
				err = p.Prover.CheckAndRestoreIdleData(front, rear, num)
				//err = n.Prover.CheckAndRestoreSubAccFiles(front, rear)
				if err != nil {
					return err
				}
				log.Println("info", "restore idle data done.")
				err = p.Prover.Recovery(*p.RsaKey, front, rear, cfg)
				if err != nil {
					return err
				}
				log.Println("info", "recovery PoIS status done.")
			} else {
				return err
			}
		}
	}
	p.Prover.AccManager.GetSnapshot()
	return nil
}

func updateMinerRegistertionInfo(cfg *confile.Confile, cli sdk.SDK, oldRegInfo *pattern.MinerInfo) error {
	var err error
	olddecspace := oldRegInfo.DeclarationSpace.Uint64() / pattern.SIZE_1TiB
	if (*oldRegInfo).DeclarationSpace.Uint64()%pattern.SIZE_1TiB != 0 {
		olddecspace = +1
	}
	newDecSpace := cfg.ReadUseSpace() / pattern.SIZE_1KiB
	if cfg.ReadUseSpace()%pattern.SIZE_1KiB != 0 {
		newDecSpace += 1
	}
	if newDecSpace > olddecspace {
		token := (newDecSpace - olddecspace) * pattern.StakingStakePerTiB
		if cfg.ReadStakingAcc() != "" && cfg.ReadStakingAcc() != cli.GetSignatureAcc() {
			signAccInfo, err := cli.QueryAccountInfo(cli.GetSignatureAccPulickey())
			if err != nil {
				if err.Error() != pattern.ERR_Empty {
					out.Err(err.Error())
					os.Exit(1)
				}
				out.Err("Failed to expand space: account does not exist or balance is empty")
				os.Exit(1)
			}
			incToken, _ := new(big.Int).SetString(fmt.Sprintf("%d%s", token, pattern.TokenPrecision_CESS), 10)
			if signAccInfo.Data.Free.CmpAbs(incToken) < 0 {
				return fmt.Errorf("Failed to expand space: signature account balance less than %d %s", incToken, cli.GetTokenSymbol())
			}
			txhash, err := cli.IncreaseStakingAmount(cli.GetSignatureAcc(), incToken)
			if err != nil {
				if txhash != "" {
					return fmt.Errorf("[%s] Failed to expand space: %v", txhash, err)
				}
				return fmt.Errorf("Failed to expand space: %v", err)
			}
			out.Ok(fmt.Sprintf("Successfully increased %dTCESS staking", token))
		} else {
			newToken, _ := new(big.Int).SetString(fmt.Sprintf("%d%s", newDecSpace*pattern.StakingStakePerTiB, pattern.TokenPrecision_CESS), 10)
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

	newPublicKey, err := sutils.ParsingPublickey(cfg.ReadEarningsAcc())
	if err == nil {
		if !sutils.CompareSlice(oldRegInfo.BeneficiaryAccount[:], newPublicKey) {
			txhash, err := cli.UpdateEarningsAccount(cfg.ReadEarningsAcc())
			if err != nil {
				return fmt.Errorf("Update earnings account err: %v, blockhash: %s", err, txhash)
			}
			out.Ok(fmt.Sprintf("[%s] Successfully updated earnings account to %s", txhash, cfg.ReadEarningsAcc()))
		}
	}

	// if !sutils.CompareSlice(peerid, n.GetPeerPublickey()) {
	// 	var peeridChain pattern.PeerId
	// 	pids := n.GetPeerPublickey()
	// 	for i := 0; i < len(pids); i++ {
	// 		peeridChain[i] = types.U8(pids[i])
	// 	}
	// 	txhash, err := n.UpdateSminerPeerId(peeridChain)
	// 	if err != nil {
	// 		out.Err(fmt.Sprintf("[%s] Update PeerId: %v", txhash, err))
	// 		os.Exit(1)
	// 	}
	// 	out.Ok(fmt.Sprintf("[%s] Successfully updated peer ID to %s", txhash, base58.Encode(n.GetPeerPublickey())))
	// }
	return nil
}

func InitRsaKey(pubkey []byte) (*proof.RSAKeyPair, error) {
	rsaPubkey, err := x509.ParsePKCS1PublicKey(pubkey)
	if err != nil {
		return nil, err
	}
	raskey := proof.NewKey()
	raskey.Spk = rsaPubkey
	return raskey, nil
}

func queryPodr2KeyFromTee(peernode *core.PeerNode, teeRecord *node.TeeRecord, signature_publickey []byte) ([]byte, error) {
	var err error
	var podr2PubkeyResponse *pb.Podr2PubkeyResponse
	var dialOptions []grpc.DialOption
	teeEndPointList := teeRecord.GetAllTeeEndpoint()
	delay := time.Duration(30)
	for i := 0; i < len(teeEndPointList); i++ {
		delay = 30
		out.Tip(fmt.Sprintf("Requesting registration parameters from tee: %s", teeEndPointList[i]))
		for tryCount := uint8(0); tryCount <= 3; tryCount++ {
			if !strings.Contains(teeEndPointList[i], "443") {
				dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
			} else {
				dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
			}
			podr2PubkeyResponse, err = peernode.GetPodr2Pubkey(
				teeEndPointList[i],
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
	return nil, errors.New("All tee nodes are busy or unavailable")
}

func checkWorkSpace(wspace node.Workspace) {
	dirfreeSpace, err := utils.GetDirFreeSpace(wspace.GetRootDir())
	if err == nil {
		if dirfreeSpace < pattern.SIZE_1GiB*32 {
			out.Warn("The workspace capacity is less than 32G")
		}
	}
	out.Tip(fmt.Sprintf("Workspace free size: %v G", dirfreeSpace/pattern.SIZE_1GiB))
}

func connectChain(cli sdk.SDK) {
	// n.Log("err", fmt.Sprintf("[%s] %v", n.GetCurrentRpcAddr(), pattern.ERR_RPC_CONNECTION))
	// n.Ichal("err", fmt.Sprintf("[%s] %v", n.GetCurrentRpcAddr(), pattern.ERR_RPC_CONNECTION))
	// n.Schal("err", fmt.Sprintf("[%s] %v", n.GetCurrentRpcAddr(), pattern.ERR_RPC_CONNECTION))
	// out.Err(fmt.Sprintf("[%s] %v", n.GetCurrentRpcAddr(), pattern.ERR_RPC_CONNECTION))
	err := cli.ReconnectRPC()
	if err != nil {
		// n.SetLastReconnectRpcTime(time.Now().Format(time.DateTime))
		// n.Log("err", "All RPCs failed to reconnect")
		// n.Ichal("err", "All RPCs failed to reconnect")
		// n.Schal("err", "All RPCs failed to reconnect")
		// out.Err("All RPCs failed to reconnect")
		return
	}
	// n.SetLastReconnectRpcTime(time.Now().Format(time.DateTime))
	cli.SetChainState(true)
	//peernode.EnableRecv()
	// out.Tip(fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
	// n.Log("info", fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
	// n.Ichal("info", fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
	// n.Schal("info", fmt.Sprintf("[%s] rpc reconnection successful", n.GetCurrentRpcAddr()))
}

func syncMinerStatus(cli sdk.SDK, l logger.Logger, miner *node.MinerState) {
	// var dialOptions []grpc.DialOption
	// var chainPublickey = make([]byte, pattern.WorkerPublicKeyLen)
	// teelist, err := n.QueryAllTeeWorkerMap()
	// if err != nil {
	// 	n.Log("err", err.Error())
	// } else {
	// 	for i := 0; i < len(teelist); i++ {
	// 		n.Log("info", fmt.Sprintf("check tee: %s", hex.EncodeToString([]byte(string(teelist[i].Pubkey[:])))))
	// 		endpoint, err := n.QueryTeeWorkEndpoint(teelist[i].Pubkey)
	// 		if err != nil {
	// 			n.Log("err", err.Error())
	// 			continue
	// 		}
	// 		endpoint = processEndpoint(endpoint)

	// 		if !strings.Contains(endpoint, "443") {
	// 			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	// 		} else {
	// 			dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
	// 		}

	// 		// verify identity public key
	// 		identityPubkeyResponse, err := n.GetIdentityPubkey(endpoint,
	// 			&pb.Request{
	// 				StorageMinerAccountId: n.GetSignatureAccPulickey(),
	// 			},
	// 			time.Duration(time.Minute),
	// 			dialOptions,
	// 			nil,
	// 		)
	// 		if err != nil {
	// 			n.Log("err", err.Error())
	// 			continue
	// 		}
	// 		//n.Log("info", fmt.Sprintf("get identityPubkeyResponse: %v", identityPubkeyResponse.Pubkey))
	// 		if len(identityPubkeyResponse.Pubkey) != pattern.WorkerPublicKeyLen {
	// 			n.DeleteTee(string(teelist[i].Pubkey[:]))
	// 			n.Log("err", fmt.Sprintf("identityPubkeyResponse.Pubkey length err: %d", len(identityPubkeyResponse.Pubkey)))
	// 			continue
	// 		}

	// 		for j := 0; j < pattern.WorkerPublicKeyLen; j++ {
	// 			chainPublickey[j] = byte(teelist[i].Pubkey[j])
	// 		}
	// 		if !sutils.CompareSlice(identityPubkeyResponse.Pubkey, chainPublickey) {
	// 			n.DeleteTee(string(teelist[i].Pubkey[:]))
	// 			n.Log("err", fmt.Sprintf("identityPubkeyResponse.Pubkey: %s", hex.EncodeToString(identityPubkeyResponse.Pubkey)))
	// 			n.Log("err", "identityPubkeyResponse.Pubkey err: not qual to chain")
	// 			continue
	// 		}

	// 		n.Log("info", fmt.Sprintf("Save a tee: %s  %d", endpoint, teelist[i].Role))
	// 		err = n.SaveTee(string(teelist[i].Pubkey[:]), endpoint, uint8(teelist[i].Role))
	// 		if err != nil {
	// 			n.Log("err", err.Error())
	// 		}
	// 	}
	// }
	minerInfo, err := cli.QueryStorageMiner(cli.GetSignatureAccPulickey())
	if err != nil {
		l.Log("err", err.Error())
		if err.Error() == pattern.ERR_Empty {
			err = miner.SaveMinerState(pattern.MINER_STATE_OFFLINE)
			if err != nil {
				l.Log("err", err.Error())
			}
		}
		return
	}
	err = miner.SaveMinerState(string(minerInfo.State))
	if err != nil {
		l.Log("err", err.Error())
	}
	miner.SaveMinerSpaceInfo(
		minerInfo.DeclarationSpace.Uint64(),
		minerInfo.IdleSpace.Uint64(),
		minerInfo.ServiceSpace.Uint64(),
		minerInfo.LockSpace.Uint64(),
	)
	return
}
