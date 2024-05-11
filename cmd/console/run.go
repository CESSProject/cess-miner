/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	sdkgo "github.com/CESSProject/cess-go-sdk"
	"github.com/CESSProject/cess-go-sdk/chain"
	sconfig "github.com/CESSProject/cess-go-sdk/config"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/node"
	"github.com/CESSProject/cess-miner/pkg/cache"
	"github.com/CESSProject/cess-miner/pkg/confile"
	"github.com/CESSProject/cess-miner/pkg/logger"
	"github.com/CESSProject/cess-miner/pkg/utils"
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
	cfg := confile.NewConfigFile()
	runtime := node.NewRunTime()
	teeRecord := node.NewTeeRecord()
	wspace := node.NewWorkspace()
	minerRecord := node.NewPeerRecord()

	runtime.ListenLocal()
	runtime.SetPID(os.Getpid())

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
	fmt.Println("config: ", cfg.ReadUseCpu())
	runtime.SetCpuCores(configs.SysInit(cfg.ReadUseCpu()))

	// new chain client
	cli, err := sdkgo.New(
		ctx,
		sdkgo.Name(configs.Name),
		sdkgo.ConnectRpcAddrs(cfg.ReadRpcEndpoints()),
		sdkgo.Mnemonic(cfg.ReadMnemonic()),
		sdkgo.TransactionTimeout(configs.TimeToWaitEvent),
	)
	if err != nil {
		out.Err(fmt.Sprintf("[sdkgo.New] %v", err))
		os.Exit(1)
	}
	defer cli.Close()

	runtime.SetCurrentRpc(cli.GetCurrentRpcAddr())
	runtime.SetChainStatus(true)
	runtime.SetMinerSignAcc(cli.GetSignatureAcc())

	err = checkRpcSynchronization(cli)
	if err != nil {
		out.Err("Failed to synchronize the main chain: network connection is down")
		os.Exit(1)
	}

	expender, err := cli.QueryExpenders(-1)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}

	register, decTib, oldRegInfo, err := checkRegistrationInfo(cli, cfg.ReadSignatureAccount(), cfg.ReadStakingAcc(), cfg.ReadUseSpace())
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}

	// new peer node
	peernode, err := p2pgo.New(
		ctx,
		p2pgo.ListenPort(cfg.ReadServicePort()),
		p2pgo.Workspace(filepath.Join(cfg.ReadWorkspace(), cli.GetSignatureAcc(), configs.Name)),
		p2pgo.BootPeers(cfg.ReadBootnodes()),
	)
	if err != nil {
		out.Err(fmt.Sprintf("[p2pgo.New] %v", err))
		os.Exit(1)
	}
	defer peernode.Close()
	runtime.SetReceiveFlag(true)

	// ok
	go node.Subscribe(ctx, peernode.GetHost(), minerRecord, peernode.GetBootnode())
	time.Sleep(time.Second)

	// check network environment
	err = checkNetworkEnv(cli, peernode.GetNetEnv())
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}

	var p *node.Pois
	var rsakey *node.RSAKeyPair
	var minerPoisInfo = &pb.MinerPoisInfo{}
	switch register {
	case configs.Unregistered:
		_, err = registerMiner(cli, cfg.ReadStakingAcc(), cfg.ReadEarningsAcc(), peernode.GetPeerPublickey(), decTib)
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}
		err = wspace.RemoveAndBuild(peernode.Workspace())
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}

		time.Sleep(chain.BlockInterval * 10)

		for i := 0; i < 3; i++ {
			rsakey, err = registerPoisKey(cli, peernode, teeRecord, minerPoisInfo, wspace, cfg.ReadPriorityTeeList())
			if err != nil {
				if !strings.Contains(err.Error(), "storage miner is not registered") {
					out.Err(err.Error())
					os.Exit(1)
				}
				time.Sleep(chain.BlockInterval)
				continue
			}
			break
		}
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}

		p, err = node.NewPOIS(
			wspace.GetPoisDir(),
			wspace.GetSpaceDir(),
			wspace.GetPoisAccDir(),
			expender,
			true, 0, 0,
			int64(cfg.ReadUseSpace()*1024), 32,
			runtime.GetCpuCores(),
			minerPoisInfo.KeyN,
			minerPoisInfo.KeyG,
			cli.GetSignatureAccPulickey(),
		)
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}
	case configs.UnregisteredPoisKey:
		err = wspace.Build(peernode.Workspace())
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}
		runtime.SetMinerState(string(oldRegInfo.State))
		for i := 0; i < 3; i++ {
			rsakey, err = registerPoisKey(cli, peernode, teeRecord, minerPoisInfo, wspace, cfg.ReadPriorityTeeList())
			if err != nil {
				if !strings.Contains(err.Error(), "storage miner is not registered") {
					out.Err(err.Error())
					os.Exit(1)
				}
				time.Sleep(chain.BlockInterval)
				continue
			}
			break
		}
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}

		err = updateMinerRegistertionInfo(cli, oldRegInfo, cfg.ReadUseSpace(), cfg.ReadStakingAcc(), cfg.ReadEarningsAcc())
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}

		p, err = node.NewPOIS(
			wspace.GetPoisDir(),
			wspace.GetSpaceDir(),
			wspace.GetPoisAccDir(),
			expender,
			true, 0, 0,
			int64(cfg.ReadUseSpace()*1024), 32,
			runtime.GetCpuCores(),
			minerPoisInfo.KeyN,
			minerPoisInfo.KeyG,
			cli.GetSignatureAccPulickey(),
		)
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}

	case configs.Registered:
		err = wspace.Build(peernode.Workspace())
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}

		runtime.SetMinerState(string(oldRegInfo.State))
		err = updateMinerRegistertionInfo(cli, oldRegInfo, cfg.ReadUseSpace(), cfg.ReadStakingAcc(), cfg.ReadEarningsAcc())
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

		p, err = node.NewPOIS(
			wspace.GetPoisDir(),
			wspace.GetSpaceDir(),
			wspace.GetPoisAccDir(),
			expender, false,
			int64(spaceProofInfo.Front),
			int64(spaceProofInfo.Rear),
			int64(cfg.ReadUseSpace()*1024), 32,
			runtime.GetCpuCores(),
			minerPoisInfo.KeyN,
			minerPoisInfo.KeyG,
			cli.GetSignatureAccPulickey(),
		)
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}

		saveAllTees(cli, peernode, teeRecord)

		buf, err := wspace.LoadRsaPublicKey()
		if err != nil {
			buf, _ = queryPodr2KeyFromTee(peernode, teeRecord.GetAllTeeEndpoint(), cli.GetSignatureAccPulickey())
		}

		rsakey, err = node.NewRsaKey(buf)
		if err != nil {
			out.Err(fmt.Sprintf("Init rsa public key err: %v", err))
			os.Exit(1)
		}

		spaceProofInfo = chain.SpaceProofInfo{}
		buf = nil

	default:
		out.Err("system err")
		os.Exit(1)
	}
	oldRegInfo = nil

	// build logger
	l, err := buildLogs(wspace.GetLogDir())
	if err != nil {
		out.Err(fmt.Sprintf("[buildLogs] %v", err))
		os.Exit(1)
	}

	// build cache
	cace, err := buildCache(wspace.GetDbDir())
	if err != nil {
		out.Err(fmt.Sprintf("[buildCache] %v", err))
		os.Exit(1)
	}

	out.Tip(fmt.Sprintf("Local peer id: %s", peernode.ID().String()))
	out.Tip(fmt.Sprintf("Network environment: %s", cli.GetNetworkEnv()))
	out.Tip(fmt.Sprintf("Number of cpu cores used: %v", runtime.GetCpuCores()))
	out.Tip(fmt.Sprintf("RPC endpoint used: %v", cli.GetCurrentRpcAddr()))
	out.Tip(fmt.Sprintf("Workspace: %v", wspace.GetRootDir()))

	err = wspace.Check()
	if err != nil {
		out.Err(err.Error())
	}

	chainState := true
	reportFileCh := make(chan bool, 1)
	reportFileCh <- true
	idleChallCh := make(chan bool, 1)
	idleChallCh <- true
	serviceChallCh := make(chan bool, 1)
	serviceChallCh <- true
	replaceIdleCh := make(chan bool, 1)
	replaceIdleCh <- true
	genIdleCh := make(chan bool, 1)
	genIdleCh <- true
	attestationIdleCh := make(chan bool, 1)
	attestationIdleCh <- true
	syncTeeCh := make(chan bool, 1)
	syncTeeCh <- true
	calcTagCh := make(chan bool, 1)
	calcTagCh <- true
	restoreCh := make(chan bool, 1)
	restoreCh <- true

	tick_block := time.NewTicker(chain.BlockInterval)
	defer tick_block.Stop()

	tick_Minute := time.NewTicker(time.Second * time.Duration(57))
	defer tick_Minute.Stop()

	tick_Hour := time.NewTicker(time.Second * time.Duration(3597))
	defer tick_Hour.Stop()

	out.Ok("Service started successfully")
	for {
		select {
		case <-tick_block.C:
			chainState = cli.GetRpcState()
			if !chainState {
				runtime.SetChainStatus(false)
				runtime.SetReceiveFlag(false)
				peernode.DisableRecv()
				err = cli.ReconnectRpc()
				l.Log("err", fmt.Sprintf("[%s] %v", cli.GetCurrentRpcAddr(), chain.ERR_RPC_CONNECTION))
				l.Ichal("err", fmt.Sprintf("[%s] %v", cli.GetCurrentRpcAddr(), chain.ERR_RPC_CONNECTION))
				l.Schal("err", fmt.Sprintf("[%s] %v", cli.GetCurrentRpcAddr(), chain.ERR_RPC_CONNECTION))
				out.Err(fmt.Sprintf("[%s] %v", cli.GetCurrentRpcAddr(), chain.ERR_RPC_CONNECTION))
				if err != nil {
					runtime.SetLastReconnectRpcTime(time.Now().Format(time.DateTime))
					l.Log("err", "All RPCs failed to reconnect")
					l.Ichal("err", "All RPCs failed to reconnect")
					l.Schal("err", "All RPCs failed to reconnect")
					out.Err("All RPCs failed to reconnect")
					break
				}
				runtime.SetLastReconnectRpcTime(time.Now().Format(time.DateTime))
				out.Ok(fmt.Sprintf("[%s] rpc reconnection successful", cli.GetCurrentRpcAddr()))
				l.Log("info", fmt.Sprintf("[%s] rpc reconnection successful", cli.GetCurrentRpcAddr()))
				l.Ichal("info", fmt.Sprintf("[%s] rpc reconnection successful", cli.GetCurrentRpcAddr()))
				l.Schal("info", fmt.Sprintf("[%s] rpc reconnection successful", cli.GetCurrentRpcAddr()))
				runtime.SetCurrentRpc(cli.GetCurrentRpcAddr())
				runtime.SetChainStatus(true)
				runtime.SetReceiveFlag(true)
				peernode.EnableRecv()
			}

		case <-tick_Minute.C:
			if !chainState {
				break
			}

			syncMinerStatus(cli, l, runtime)
			if runtime.GetMinerState() == chain.MINER_STATE_EXIT ||
				runtime.GetMinerState() == chain.MINER_STATE_OFFLINE {
				break
			}

			if len(syncTeeCh) > 0 {
				<-syncTeeCh
				go node.SyncTeeInfo(cli, l, peernode, teeRecord, syncTeeCh)
			}

			if len(reportFileCh) > 0 {
				<-reportFileCh
				go node.ReportFiles(reportFileCh, cli, runtime, l, wspace.GetFileDir(), wspace.GetTmpDir())
			}

			if len(attestationIdleCh) > 0 {
				<-attestationIdleCh
				go node.AttestationIdle(cli, peernode, p, runtime, minerPoisInfo, teeRecord, l, attestationIdleCh)
			}

			if len(calcTagCh) > 0 {
				<-calcTagCh
				go node.CalcTag(cli, cace, l, runtime, teeRecord, wspace.GetFileDir(), calcTagCh)
			}

			if len(idleChallCh) > 0 || len(serviceChallCh) > 0 {
				go node.ChallengeMgt(cli, l, wspace, runtime, teeRecord, peernode, minerPoisInfo, rsakey, p, cace, idleChallCh, serviceChallCh)
				time.Sleep(chain.BlockInterval)
			}

			if len(genIdleCh) > 0 && !runtime.GetServiceChallengeFlag() && !runtime.GetIdleChallengeFlag() {
				<-genIdleCh
				go node.GenIdle(l, p.Prover, runtime, peernode.Workspace(), cfg.ReadUseSpace(), genIdleCh)
			}

		case <-tick_Hour.C:
			if runtime.GetMinerState() == chain.MINER_STATE_EXIT ||
				runtime.GetMinerState() == chain.MINER_STATE_OFFLINE {
				break
			}

			// go n.reportLogsMgt(ch_reportLogs)

			if !chainState {
				break
			}

			if len(replaceIdleCh) > 0 {
				<-replaceIdleCh
				go node.ReplaceIdle(cli, l, p, minerPoisInfo, teeRecord, peernode, replaceIdleCh)
			}

			if len(restoreCh) > 0 {
				<-restoreCh
				go node.RestoreFiles(cli, cace, l, wspace.GetFileDir(), restoreCh)
			}
		}
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
	cfg := confile.NewConfigFile()
	err := cfg.Parse(file)
	return cfg, err
}

func buildConfigItems(cmd *cobra.Command) (*confile.Confile, error) {
	var (
		istips      bool
		lines       string
		rpc         []string
		cfg         = confile.NewConfigFile()
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

	cfg := confile.NewConfigFile()
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

func buildLogs(logDir string) (*logger.Lg, error) {
	var logs_info = make(map[string]string)
	for _, v := range logger.LogFiles {
		logs_info[v] = filepath.Join(logDir, v+".log")
	}
	return logger.NewLogs(logs_info)
}

func checkNetworkEnv(cli *chain.ChainClient, netenv string) error {
	chain := cli.GetNetworkEnv()
	if strings.Contains(chain, configs.DevNet) {
		if !strings.Contains(netenv, configs.DevNet) {
			return errors.New("chain and p2p are not in the same network")
		}
	} else if strings.Contains(chain, configs.TestNet) {
		if !strings.Contains(netenv, configs.TestNet) {
			return errors.New("chain and p2p are not in the same network")
		}
	} else if strings.Contains(chain, configs.MainNet) {
		if !strings.Contains(netenv, configs.MainNet) {
			return errors.New("chain and p2p are not in the same network")
		}
	} else {
		return errors.New("unknown chain network")
	}

	chainVersion, err := cli.SystemVersion()
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

func checkRegistrationInfo(cli *chain.ChainClient, signatureAcc, stakingAcc string, useSpace uint64) (int, uint64, *chain.MinerInfo, error) {
	minerInfo, err := cli.QueryMinerItems(cli.GetSignatureAccPulickey(), -1)
	if err != nil {
		if err.Error() != chain.ERR_Empty {
			return configs.Unregistered, 0, &minerInfo, err
		}
		decTib := useSpace / sconfig.SIZE_1KiB
		if useSpace%sconfig.SIZE_1KiB != 0 {
			decTib += 1
		}
		token := decTib * chain.StakingStakePerTiB
		accInfo, err := cli.QueryAccountInfo(cli.GetSignatureAcc(), -1)
		if err != nil {
			if err.Error() != chain.ERR_Empty {
				return configs.Unregistered, decTib, &minerInfo, fmt.Errorf("failed to query signature account information: %v", err)
			}
			return configs.Unregistered, decTib, &minerInfo, errors.New("signature account does not exist, possible: 1.balance is empty 2.rpc address error")
		}
		token_cess, _ := new(big.Int).SetString(fmt.Sprintf("%d%s", token, chain.TokenPrecision_CESS), 10)
		if stakingAcc == "" || stakingAcc == signatureAcc {
			if accInfo.Data.Free.CmpAbs(token_cess) < 0 {
				return configs.Unregistered, decTib, &minerInfo, fmt.Errorf("signature account balance less than %d %s", token, cli.GetTokenSymbol())
			}
		} else {
			stakingAccInfo, err := cli.QueryAccountInfo(stakingAcc, -1)
			if err != nil {
				if err.Error() != chain.ERR_Empty {
					return configs.Unregistered, decTib, &minerInfo, fmt.Errorf("failed to query staking account information: %v", err)
				}
				return configs.Unregistered, decTib, &minerInfo, fmt.Errorf("staking account does not exist, possible: 1.balance is empty 2.rpc address error")
			}
			if stakingAccInfo.Data.Free.CmpAbs(token_cess) < 0 {
				return configs.Unregistered, decTib, &minerInfo, fmt.Errorf("staking account balance less than %d %s", token, cli.GetTokenSymbol())
			}
		}
		return configs.Unregistered, decTib, &minerInfo, nil
	}
	if !minerInfo.SpaceProofInfo.HasValue() {
		return configs.UnregisteredPoisKey, 0, &minerInfo, nil
	}
	return configs.Registered, 0, &minerInfo, nil
}

func registerMiner(cli *chain.ChainClient, stakingAcc, earningsAcc string, peer_publickey []byte, decTib uint64) (string, error) {
	if stakingAcc != "" && stakingAcc != cli.GetSignatureAcc() {
		out.Ok(fmt.Sprintf("Specify staking account: %s", stakingAcc))
		txhash, err := cli.RegnstkAssignStaking(earningsAcc, peer_publickey, stakingAcc, uint32(decTib))
		if err != nil {
			if txhash != "" {
				err = fmt.Errorf("[%s] %v", txhash, err)
			}
			return txhash, err
		}
		out.Ok(fmt.Sprintf("Storage node registration successful: %s", txhash))
		return txhash, nil
	}

	txhash, err := cli.RegnstkSminer(earningsAcc, peer_publickey, uint64(decTib*chain.StakingStakePerTiB), uint32(decTib))
	if err != nil {
		if txhash != "" {
			err = fmt.Errorf("[%s] %v", txhash, err)
		}
		return txhash, err
	}
	out.Ok(fmt.Sprintf("Storage node registration successful: %s", txhash))
	return txhash, nil
}

func saveAllTees(cli *chain.ChainClient, peernode *core.PeerNode, teeRecord *node.TeeRecord) error {
	var (
		err            error
		teeList        []chain.WorkerInfo
		dialOptions    []grpc.DialOption
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
			return err
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
	return nil
}

func registerPoisKey(
	cli *chain.ChainClient,
	peernode *core.PeerNode,
	teeRecord *node.TeeRecord,
	minerPoisInfo *pb.MinerPoisInfo,
	ws *node.Workspace,
	priorityTeeList []string,
) (*node.RSAKeyPair, error) {
	var (
		err                    error
		teeList                []chain.WorkerInfo
		dialOptions            []grpc.DialOption
		responseMinerInitParam *pb.ResponseMinerInitParam
		rsakey                 *node.RSAKeyPair
		chainPublickey         = make([]byte, chain.WorkerPublicKeyLen)
		teeEndPointList        = make([]string, len(priorityTeeList))
	)
	copy(teeEndPointList, priorityTeeList)
	for {
		teeList, err = cli.QueryAllWorkers(-1)
		if err != nil {
			if err.Error() == chain.ERR_Empty {
				out.Err("No tee found, waiting for the next minute's query...")
				time.Sleep(time.Minute)
				continue
			}
			return rsakey, err
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
					time.Sleep(chain.BlockInterval * 2)
					continue
				}
				out.Err(fmt.Sprintf("Request err: %v", err))
				break
			}

			rsakey, err = node.NewRsaKey(responseMinerInitParam.Podr2Pbk)
			if err != nil {
				out.Err(fmt.Sprintf("Request err: %v", err))
				break
			}

			err = ws.SaveRsaPublicKey(responseMinerInitParam.Podr2Pbk)
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
				cli,
				poisKey,
				responseMinerInitParam.SignatureWithTeeController[:],
				responseMinerInitParam.StatusTeeSign,
				teeWorkPubkey,
			)
			if err != nil {
				out.Err(fmt.Sprintf("register miner pois key err: %v", err))
				break
			}
			return rsakey, nil
		}
	}
	return rsakey, errors.New("all tee nodes are busy or unavailable")
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
	}
	return err
}

func updateMinerRegistertionInfo(cli *chain.ChainClient, oldRegInfo *chain.MinerInfo, useSpace uint64, stakingAcc, earningsAcc string) error {
	var err error
	olddecspace := oldRegInfo.DeclarationSpace.Uint64() / sconfig.SIZE_1TiB
	if (*oldRegInfo).DeclarationSpace.Uint64()%sconfig.SIZE_1TiB != 0 {
		olddecspace = +1
	}
	newDecSpace := useSpace / sconfig.SIZE_1KiB
	if useSpace%sconfig.SIZE_1KiB != 0 {
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

	// if !sutils.CompareSlice(peerid, n.GetPeerPublickey()) {
	// 	var peeridChain chain.PeerId
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

func queryPodr2KeyFromTee(peernode *core.PeerNode, teeEndPointList []string, signature_publickey []byte) ([]byte, error) {
	var err error
	var podr2PubkeyResponse *pb.Podr2PubkeyResponse
	var dialOptions []grpc.DialOption
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
	return nil, errors.New("all tee nodes are busy or unavailable")
}

func syncMinerStatus(cli *chain.ChainClient, l *logger.Lg, r *node.RunningState) {
	l.Log("info", "will QueryStorageMiner")
	minerInfo, err := cli.QueryMinerItems(cli.GetSignatureAccPulickey(), -1)
	if err != nil {
		l.Log("err", err.Error())
		if err.Error() == chain.ERR_Empty {
			r.SetMinerState(chain.MINER_STATE_OFFLINE)
			err = r.SetMinerState(chain.MINER_STATE_OFFLINE)
			if err != nil {
				l.Log("err", err.Error())
			}
		}
		return
	}
	l.Log("info", fmt.Sprintf("StorageMiner state: %s", minerInfo.State))
	r.SetMinerState(string(minerInfo.State))
	err = r.SetMinerState(string(minerInfo.State))
	if err != nil {
		l.Log("err", err.Error())
	}
	r.SetMinerSpaceInfo(
		minerInfo.DeclarationSpace.Uint64(),
		minerInfo.IdleSpace.Uint64(),
		minerInfo.ServiceSpace.Uint64(),
		minerInfo.LockSpace.Uint64(),
	)
}
