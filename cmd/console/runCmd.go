/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	"github.com/CESSProject/cess-bucket/pkg/cache"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	sdkgo "github.com/CESSProject/sdk-go"
	"github.com/CESSProject/sdk-go/core/client"
	"github.com/CESSProject/sdk-go/core/rule"
	"github.com/spf13/cobra"
)

// runCmd is used to start the service
func runCmd(cmd *cobra.Command, args []string) {
	var (
		ok       bool
		err      error
		logDir   string
		cacheDir string
		n        = node.New()
	)

	// Build profile instances
	n.Cfg, err = buildConfigFile(cmd, "", 0)
	if err != nil {
		logERR(fmt.Sprintf("[buildConfigFile] %v", err))
		os.Exit(1)
	}

	//Build client
	cli, err := sdkgo.New(
		configs.Name,
		sdkgo.ConnectRpcAddrs(n.Cfg.GetRpcAddr()),
		sdkgo.ListenPort(n.Cfg.GetServicePort()),
		sdkgo.Workspace(n.Cfg.GetWorkspace()),
		sdkgo.Mnemonic(n.Cfg.GetMnemonic()),
		sdkgo.TransactionTimeout(configs.TimeToWaitEvent),
	)
	if err != nil {
		logERR(fmt.Sprintf("[sdkgo.New] %v", err))
		os.Exit(1)
	}

	n.Cli, ok = cli.(*client.Cli)
	if !ok {
		logERR("Invalid client type")
		os.Exit(1)
	}

	token := n.Cfg.GetUseSpace() / (rule.SIZE_1GiB * 1024)
	if n.Cfg.GetUseSpace()%(rule.SIZE_1GiB*1024) != 0 {
		token += 1
	}
	token *= 1000

	_, err = n.Cli.RegisterRole(configs.Name, n.Cfg.GetIncomeAcc(), token)
	if err != nil {
		logERR(fmt.Sprintf("[RegisterRole] %v", err))
		os.Exit(1)
	}

	// Build data directory
	logDir, cacheDir, err = buildDir(n.Cli.Workspace())
	if err != nil {
		logERR(fmt.Sprintf("[buildDir] %v", err))
		os.Exit(1)
	}

	// Build cache instance
	n.Cach, err = buildCache(cacheDir)
	if err != nil {
		logERR(fmt.Sprintf("[buildCache] %v", err))
		os.Exit(1)
	}

	//Build log instance
	n.Log, err = buildLogs(logDir)
	if err != nil {
		logERR(fmt.Sprintf("[buildLogs] %v", err))
		os.Exit(1)
	}

	// run
	n.Run()
}

func buildConfigFile(cmd *cobra.Command, ip4 string, port int) (confile.Confile, error) {
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
	err := cfg.Parse(conFilePath, ip4, port)
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
	for len(rpc) == 0 {
		if !istips {
			logTip("Please enter the rpc address of the chain, multiple addresses are separated by spaces:")
			istips = true
		}
		lines, err = inputReader.ReadString('\n')
		if err != nil {
			logERR(err.Error())
			continue
		} else {
			rpc = strings.Split(strings.ReplaceAll(lines, "\n", ""), " ")
		}
	}
	cfg.SetRpcAddr(rpc)

	workspace, err := cmd.Flags().GetString("ws")
	if err != nil {
		return cfg, err
	}
	istips = false
	for workspace == "" {
		if !istips {
			logTip(fmt.Sprintf("Please enter the workspace, press enter to use %v by default workspace:", configs.DefaultWorkspace))
			istips = true
		}
		lines, err = inputReader.ReadString('\n')
		if err != nil {
			logERR(err.Error())
			continue
		} else {
			workspace = strings.ReplaceAll(lines, "\n", "")
		}
		if workspace != "" {
			if workspace[0] != configs.DefaultWorkspace {
				logERR(fmt.Sprintf("Please enter the full path of the workspace starting with %v :", configs.DefaultWorkspace))
				continue
			}
		} else {
			workspace = string(configs.DefaultWorkspace)
		}
		err = cfg.SetWorkspace(workspace)
		if err != nil {
			logERR(err.Error())
			continue
		}
		break
	}

	var income string
	income, err = cmd.Flags().GetString("earnings")
	if err != nil {
		return cfg, err
	}
	istips = false
	for income == "" {
		if !istips {
			logTip("Please enter your earnings account, if you are already registered and do not want to update, please press enter to skip:")
			istips = true
		}
		lines, err = inputReader.ReadString('\n')
		if err != nil {
			logERR(err.Error())
			continue
		}
		income = strings.ReplaceAll(lines, "\n", "")
		err = cfg.SetIncomeAcc(income)
		if err != nil {
			logERR(err.Error())
			continue
		}
		break
	}

	var listenPort int
	listenPort, err = cmd.Flags().GetInt("port")
	if err != nil {
		listenPort, err = cmd.Flags().GetInt("p")
		if err != nil {
			return cfg, err
		}
	}
	istips = false
	for listenPort == 0 {
		if !istips {
			logTip("Please enter your service port:")
			istips = true
		}
		lines, err = inputReader.ReadString('\n')
		if err != nil {
			logERR(err.Error())
			continue
		}
		listenPort, err = strconv.Atoi(strings.ReplaceAll(lines, "\n", ""))
		if err != nil {
			logERR("Please enter a number between 1024~65535:")
			continue
		}
		if listenPort != 0 {
			err = cfg.SetServicePort(listenPort)
			if err != nil {
				logERR(err.Error())
				continue
			}
		}
		break
	}

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
			logTip("Please enter the maximum space used by the storage node in GiB:")
			istips = true
		}
		lines, err = inputReader.ReadString('\n')
		if err != nil {
			logERR(err.Error())
			continue
		}
		useSpace, err = strconv.ParseUint(strings.ReplaceAll(lines, "\n", ""), 10, 64)
		if err != nil {
			logERR("Please enter an integer greater than 0:")
			continue
		}
		cfg.SetUseSpace(useSpace)
		break
	}

	var mnemonic string
	istips = false
	for {
		if !istips {
			logTip("Please enter the mnemonic of the staking account:")
			istips = true
		}
		mnemonic, err = utils.PasswdWithMask("", "", "")
		if err != nil {
			logERR(err.Error())
			continue
		}
		if mnemonic == "" {
			logERR("The mnemonic you entered is empty, please re-enter:")
			continue
		}
		err = cfg.SetMnemonic(mnemonic)
		if err != nil {
			logERR(err.Error())
			continue
		}
		break
	}
	return cfg, nil
}

func buildDir(workspace string) (string, string, error) {
	logDir := filepath.Join(workspace, configs.LogDir)
	if err := os.MkdirAll(logDir, configs.DirMode); err != nil {
		return "", "", err
	}

	cacheDir := filepath.Join(workspace, configs.DbDir)
	if err := os.MkdirAll(cacheDir, configs.DirMode); err != nil {
		return "", "", err
	}

	log.Println(workspace)
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
