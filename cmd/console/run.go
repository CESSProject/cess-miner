/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	"github.com/CESSProject/cess-bucket/pkg/cache"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	sdkgo "github.com/CESSProject/sdk-go"
	"github.com/CESSProject/sdk-go/core/rule"
	"github.com/spf13/cobra"
)

// runCmd is used to start the service
//
// Usage:
//
//	bucket run
func runCmd(cmd *cobra.Command, args []string) {
	var (
		err      error
		logDir   string
		cacheDir string
		n        = node.New()
	)

	// Build profile instances
	n.Cfg, err = buildConfigFile(cmd, "", 0)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	//Build client
	n.Cli, err = sdkgo.New(
		configs.Name,
		sdkgo.ConnectRpcAddrs(n.Cfg.GetRpcAddr()),
		sdkgo.ListenPort(n.Cfg.GetServicePort()),
		sdkgo.Workspace(n.Cfg.GetWorkspace()),
		sdkgo.Mnemonic(n.Cfg.GetMnemonic()),
		sdkgo.TransactionTimeout(configs.TimeToWaitEvent),
	)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	token := n.Cfg.GetUseSpace() / (rule.SIZE_1GiB * 1024)
	if n.Cfg.GetUseSpace()%(rule.SIZE_1GiB*1024) != 0 {

		token += 1
	}
	token *= 1000

	_, err = n.Cli.Register(configs.Name, n.Cfg.GetIncomeAcc(), token)
	if err != nil {
		log.Println("Register err: ", err)
		os.Exit(1)
	}

	// Build data directory
	logDir, cacheDir, n.SpaceDir, n.FileDir, n.TmpDir, err = buildDir(n.Cli.Workspace())
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// Build cache instance
	n.Cach, err = buildCache(cacheDir)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	//Build log instance
	n.Log, err = buildLogs(logDir)
	if err != nil {
		log.Println(err)
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

	rpc, err := cmd.Flags().GetString("rpc")
	if err != nil {
		return cfg, err
	}

	workspace, err := cmd.Flags().GetString("ws")
	if err != nil {
		return cfg, err
	}

	ip, err := cmd.Flags().GetString("ip")
	if err != nil {
		return cfg, err
	}

	income, err := cmd.Flags().GetString("income")
	if err != nil {
		return cfg, err
	}

	port, err = cmd.Flags().GetInt("port")
	if err != nil {
		port, err = cmd.Flags().GetInt("p")
		if err != nil {
			return cfg, err
		}
	}

	useSpace, err := cmd.Flags().GetUint64("space")
	if err != nil {
		useSpace, err = cmd.Flags().GetUint64("s")
		if err != nil {
			return cfg, err
		}
	}

	cfg.SetRpcAddr([]string{rpc})
	err = cfg.SetWorkspace(workspace)
	if err != nil {
		return cfg, err
	}
	err = cfg.SetServiceAddr(ip)
	if err != nil {
		return cfg, err
	}
	err = cfg.SetServicePort(port)
	if err != nil {
		return cfg, err
	}
	cfg.SetIncomeAcc(income)
	cfg.SetUseSpace(useSpace)

	mnemonic, err := utils.PasswdWithMask("Please enter your mnemonic and press Enter to end:", "", "")
	if err != nil {
		return cfg, err
	}
	err = cfg.SetMnemonic(mnemonic)
	if err != nil {
		return cfg, err
	}
	return cfg, nil
}

func buildDir(workspace string) (string, string, string, string, string, error) {
	logDir := filepath.Join(workspace, configs.LogDir)
	if err := os.MkdirAll(logDir, configs.DirMode); err != nil {
		return "", "", "", "", "", err
	}

	cacheDir := filepath.Join(workspace, configs.DbDir)
	if err := os.MkdirAll(cacheDir, configs.DirMode); err != nil {
		return "", "", "", "", "", err
	}

	spaceDir := filepath.Join(workspace, configs.SpaceDir)
	if err := os.MkdirAll(spaceDir, configs.DirMode); err != nil {
		return "", "", "", "", "", err
	}

	fileDir := filepath.Join(workspace, rule.FileDir)
	if err := os.MkdirAll(fileDir, configs.DirMode); err != nil {
		return "", "", "", "", "", err
	}

	tmpDir := filepath.Join(workspace, rule.TempDir)
	if err := os.MkdirAll(tmpDir, configs.DirMode); err != nil {
		return "", "", "", "", "", err
	}

	log.Println(workspace)
	return logDir, cacheDir, spaceDir, fileDir, tmpDir, nil
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
