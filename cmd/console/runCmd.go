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
	p2pgo "github.com/CESSProject/p2p-go"
	sdkgo "github.com/CESSProject/sdk-go"
	"github.com/CESSProject/sdk-go/core/pattern"
	"github.com/howeyc/gopass"
	"github.com/spf13/cobra"
)

// runCmd is used to start the service
func runCmd(cmd *cobra.Command, args []string) {
	var (
		err       error
		logDir    string
		cacheDir  string
		earnings  string
		bootstrap []string
		n         = node.New()
	)

	// Build profile instances
	n.Confile, err = buildConfigFile(cmd, 0)
	if err != nil {
		configs.Err(fmt.Sprintf("[buildConfigFile] %v", err))
		os.Exit(1)
	}

	//Build client
	n.SDK, err = sdkgo.New(
		configs.Name,
		sdkgo.ConnectRpcAddrs(n.GetRpcAddr()),
		sdkgo.Mnemonic(n.GetMnemonic()),
		sdkgo.TransactionTimeout(configs.TimeToWaitEvent),
	)
	if err != nil {
		configs.Err(fmt.Sprintf("[sdkgo.New] %v", err))
		os.Exit(1)
	}

	pubkey, err := n.QueryTeePodr2Puk()
	if err == nil {
		n.Key.SetKeyN(pubkey)
	}

	boot, _ := cmd.Flags().GetString("boot")
	if boot == "" {
		configs.Warn("Empty boot node")
	} else {
		bootstrap, _ = utils.ParseMultiaddrs(boot)
		if len(bootstrap) > 0 {
			configs.Tip(fmt.Sprintf("Bootstrap node: %v", bootstrap))
		}
	}

	//  else {
	// 	peerid, err := n.AddMultiaddrToPearstore(boot, peerstore.PermanentAddrTTL)
	// 	if err != nil {
	// 		configs.Err(fmt.Sprintf("Failed to connect to the boot node: %s", boot))
	// 	} else {
	// 		configs.BootPeerId = peerid.String()
	// 		configs.Ok(fmt.Sprintf("Successfully connected to the boot node: %s", peerid.String()))
	// 	}
	// }

	n.P2P, err = p2pgo.New(
		context.Background(),
		"",
		p2pgo.ListenPort(n.GetServicePort()),
		p2pgo.Workspace(filepath.Join(n.GetWorkspace(), n.GetStakingAcc(), configs.Name)),
		p2pgo.BootPeers(bootstrap),
	)

	for {
		syncSt, err := n.SyncState()
		if err != nil {
			configs.Err(err.Error())
			os.Exit(1)
		}
		if syncSt.CurrentBlock == syncSt.HighestBlock {
			configs.Ok(fmt.Sprintf("Synchronization main chain completed: %d", syncSt.CurrentBlock))
			break
		}
		configs.Tip(fmt.Sprintf("In the synchronization main chain: %d ...", syncSt.CurrentBlock))
		time.Sleep(time.Second * time.Duration(utils.Ternary(int64(syncSt.HighestBlock-syncSt.CurrentBlock)*6, 30)))
	}

	token := n.GetUseSpace() / (pattern.SIZE_1GiB * 1024)
	if n.GetUseSpace()%(pattern.SIZE_1GiB*1024) != 0 {
		token += 1
	}
	token *= 1000

	_, earnings, err = n.Register(configs.Name, n.GetPeerPublickey(), n.GetEarningsAcc(), token)
	if err != nil {
		configs.Err(fmt.Sprintf("[Register] %v", err))
		os.Exit(1)
	}
	n.SetEarningsAcc(earnings)

	// Build data directory
	logDir, cacheDir, err = buildDir(n.Workspace())
	if err != nil {
		configs.Err(fmt.Sprintf("[buildDir] %v", err))
		os.Exit(1)
	}

	// Build cache instance
	n.Cache, err = buildCache(cacheDir)
	if err != nil {
		configs.Err(fmt.Sprintf("[buildCache] %v", err))
		os.Exit(1)
	}

	//Build log instance
	n.Logger, err = buildLogs(logDir)
	if err != nil {
		configs.Err(fmt.Sprintf("[buildLogs] %v", err))
		os.Exit(1)
	}

	configs.Tip(fmt.Sprintf("Local peer: %s", n.Multiaddr()))

	// run
	n.Run()
}

func buildConfigFile(cmd *cobra.Command, port int) (confile.Confile, error) {
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
	err := cfg.Parse(conFilePath, port)
	if err == nil {
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
				configs.Input(fmt.Sprintf("Enter the rpc address of the chain, multiple addresses are separated by spaces, press Enter to skip\nto use [%s, %s] as default rpc address:", configs.DefaultRpcAddr1, configs.DefaultRpcAddr2))
				istips = true
			}
			lines, err = inputReader.ReadString('\n')
			if err != nil {
				configs.Err(err.Error())
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

	configs.Ok(fmt.Sprintf("%v", cfg.GetRpcAddr()))

	workspace, err := cmd.Flags().GetString("ws")
	if err != nil {
		return cfg, err
	}
	istips = false
	if workspace == "" {
		for {
			if !istips {
				configs.Input(fmt.Sprintf("Enter the workspace path, press Enter to skip to use %s as default workspace:", configs.DefaultWorkspace))
				istips = true
			}
			lines, err = inputReader.ReadString('\n')
			if err != nil {
				configs.Err(err.Error())
				continue
			} else {
				workspace = strings.ReplaceAll(lines, "\n", "")
			}
			if workspace != "" {
				if workspace[0] != configs.DefaultWorkspace[0] {
					workspace = ""
					configs.Err(fmt.Sprintf("Enter the full path of the workspace starting with %s :", configs.DefaultWorkspace))
					continue
				}
			} else {
				workspace = configs.DefaultWorkspace
			}
			err = cfg.SetWorkspace(workspace)
			if err != nil {
				configs.Err(err.Error())
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

	configs.Ok(fmt.Sprintf("%v", cfg.GetWorkspace()))

	var earnings string
	earnings, err = cmd.Flags().GetString("earnings")
	if err != nil {
		return cfg, err
	}
	istips = false
	if earnings == "" {
		for {
			if !istips {
				configs.Input("Enter the earnings account, if you have already registered and don't want to update, press Enter to skip:")
				istips = true
			}
			lines, err = inputReader.ReadString('\n')
			if err != nil {
				configs.Err(err.Error())
				continue
			}
			earnings = strings.ReplaceAll(lines, "\n", "")
			err = cfg.SetEarningsAcc(earnings)
			if err != nil {
				earnings = ""
				configs.Err("Invalid account, please check and re-enter:")
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

	configs.Ok(fmt.Sprintf("%v", cfg.GetEarningsAcc()))

	var listenPort int
	listenPort, err = cmd.Flags().GetInt("port")
	if err != nil {
		listenPort, err = cmd.Flags().GetInt("p")
		if err != nil {
			return cfg, err
		}
	}
	istips = false
	for listenPort < 1024 {
		if !istips {
			configs.Input(fmt.Sprintf("Enter the service port, press Enter to skip to use %d as default port:", configs.DefaultServicePort))
			istips = true
		}
		lines, err = inputReader.ReadString('\n')
		if err != nil {
			configs.Err(err.Error())
			continue
		}
		lines = strings.ReplaceAll(lines, "\n", "")
		if lines == "" {
			listenPort = configs.DefaultServicePort
		} else {
			listenPort, err = strconv.Atoi(lines)
			if err != nil || listenPort < 1024 {
				listenPort = 0
				configs.Err("Please enter a number between 1024~65535:")
				continue
			}
		}

		err = cfg.SetServicePort(listenPort)
		if err != nil {
			listenPort = 0
			configs.Err("Please enter a number between 1024~65535:")
			continue
		}
	}

	configs.Ok(fmt.Sprintf("%v", cfg.GetServicePort()))

	useSpace, err := cmd.Flags().GetUint64("space")
	if err != nil {
		useSpace, err = cmd.Flags().GetUint64("s")
		if err != nil {
			return cfg, err
		}
	}
	istips = false
	for useSpace == 0 {
		if !istips {
			configs.Input("Please enter the maximum space used by the storage node in GiB:")
			istips = true
		}
		lines, err = inputReader.ReadString('\n')
		if err != nil {
			configs.Err(err.Error())
			continue
		}
		lines = strings.ReplaceAll(lines, "\n", "")
		if lines == "" {
			configs.Err("Please enter an integer greater than or equal to 0:")
			continue
		}
		useSpace, err = strconv.ParseUint(lines, 10, 64)
		if err != nil {
			useSpace = 0
			configs.Err("Please enter an integer greater than or equal to 0:")
			continue
		}
		cfg.SetUseSpace(useSpace)
		break
	}

	configs.Ok(fmt.Sprintf("%v", cfg.GetUseSpace()))

	//var mnemonic string
	istips = false
	for {
		if !istips {
			configs.Input("Please enter the mnemonic of the staking account:")
			istips = true
		}
		pwd, err := gopass.GetPasswdMasked()
		if err != nil {
			if err.Error() == "interrupted" || err.Error() == "interrupt" || err.Error() == "killed" {
				os.Exit(0)
			}
			configs.Err("Invalid mnemonic, please check and re-enter:")
			continue
		}
		if len(pwd) == 0 {
			configs.Err("The mnemonic you entered is empty, please re-enter:")
			continue
		}
		err = cfg.SetMnemonic(string(pwd))
		if err != nil {
			configs.Err("Invalid mnemonic, please check and re-enter:")
			continue
		}
		break
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
				configs.Input(fmt.Sprintf("Enter the rpc address of the chain, multiple addresses are separated by spaces, press Enter to skip\nto use [%s, %s] as default rpc address:", configs.DefaultRpcAddr1, configs.DefaultRpcAddr2))
				istips = true
			}
			lines, err = inputReader.ReadString('\n')
			if err != nil {
				configs.Err(err.Error())
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
			configs.Input("Please enter the mnemonic of the staking account:")
			istips = true
		}
		pwd, err := gopass.GetPasswdMasked()
		if err != nil {
			if err.Error() == "interrupted" || err.Error() == "interrupt" || err.Error() == "killed" {
				os.Exit(0)
			}
			configs.Err("Invalid mnemonic, please check and re-enter:")
			continue
		}
		if len(pwd) == 0 {
			configs.Err("The mnemonic you entered is empty, please re-enter:")
			continue
		}
		err = cfg.SetMnemonic(string(pwd))
		if err != nil {
			configs.Err("Invalid mnemonic, please check and re-enter:")
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

	configs.Ok(workspace)
	return logDir, cacheDir, nil
}

func buildCache(cacheDir string) (cache.Cache, error) {
	return cache.NewCache(cacheDir, 0, 0, configs.NameSpace)
}

func buildLogs(logDir string) (logger.Logger, error) {
	var logs_info = make(map[string]string)
	for _, v := range configs.LogFiles {
		logs_info[v] = filepath.Join(logDir, v+".log")
	}
	return logger.NewLogs(logs_info)
}
