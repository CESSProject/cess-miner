/*
   Copyright 2022 CESS (Cumulus Encrypted Storage System) authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package console

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/CESSProject/cess-bucket/pkg/db"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/serve"
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
		node     = node.New()
	)

	// Build profile instances
	node.Cfile, err = buildConfigFile(cmd)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// Build chain instance
	node.Chn, err = buildChain(node.Cfile, configs.TimeOut_WaitBlock)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// Build data directory
	logDir, cacheDir, node.FillerDir, node.FileDir, node.TmpDir, err = buildDir(node.Cfile, node.Chn)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// Build cache instance
	node.Cach, err = buildCache(cacheDir)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	//Build log instance
	node.Logs, err = buildLogs(logDir)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	//Build server instance
	node.Ser = buildServer(
		configs.Name,
		node.Cfile.GetServicePortNum(),
		node.Chn,
		node.Logs,
		node.Cach,
		node.FileDir,
		node.TmpDir,
	)

	// run
	node.Run()
}

func buildConfigFile(cmd *cobra.Command) (confile.IConfile, error) {
	var conFilePath string
	configpath1, _ := cmd.Flags().GetString("config")
	configpath2, _ := cmd.Flags().GetString("c")
	if configpath1 != "" {
		conFilePath = configpath1
	} else {
		conFilePath = configpath2
	}

	cfg := confile.NewConfigfile()
	if err := cfg.Parse(conFilePath); err != nil {
		return nil, err
	}
	return cfg, nil
}

func buildChain(cfg confile.IConfile, timeout time.Duration) (chain.IChain, error) {
	// connecting chain
	client, err := chain.NewChainClient(cfg.GetRpcAddr(), cfg.GetCtrlPrk(), cfg.GetIncomeAcc(), timeout)
	if err != nil {
		return nil, err
	}

	// judge the balance
	accountinfo, err := client.GetAccountInfo(client.GetPublicKey())
	if err != nil {
		return nil, err
	}

	if accountinfo.Data.Free.CmpAbs(new(big.Int).SetUint64(configs.MinimumBalance)) == -1 {
		return nil, fmt.Errorf("Account balance is less than %v pico\n", configs.MinimumBalance)
	}

	// sync block
	for {
		ok, err := client.GetSyncStatus()
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		log.Println("In sync block...")
		time.Sleep(configs.BlockInterval)
	}
	log.Println("Complete synchronization of primary network block data")

	// whether to register
	_, err = client.GetMinerInfo(client.GetPublicKey())
	if err != nil {
		if err.Error() == chain.ERR_Empty {
			err = register(client, cfg)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return client, nil
}

func register(chn chain.IChain, cfg confile.IConfile) error {
	//Calculate the deposit based on the size of the storage space
	pledgeTokens := configs.DepositPerTiB * cfg.GetStorageSpaceOnTiB()

	//Registration information on the chain
	txhash, err := chn.Register(
		cfg.GetIncomeAcc(),
		cfg.GetServiceAddr(),
		uint16(cfg.GetServicePortNum()),
		pledgeTokens,
		chain.IasCert{},
		chain.IasSig{},
		chain.QuoteBody{},
		chain.Signature{},
	)
	if err != nil {
		if err.Error() == chain.ERR_Empty {
			log.Println("[err] Please check if the wallet is registered and its balance.")
		} else {
			if txhash != "" {
				msg := configs.HELP_Head + fmt.Sprintf(" %v\n", txhash)
				msg += fmt.Sprintf("%v\n", configs.HELP_register)
				msg += configs.HELP_Tail
				log.Printf("[pending] %v\n", msg)
			} else {
				log.Printf("[err] %v.\n", err)
			}
		}
		return err
	}

	fmt.Printf("\x1b[%dm[ok]\x1b[0m Registration success\n", 42)
	return nil
}

func buildDir(cfg confile.IConfile, chn chain.IChain) (string, string, string, string, string, error) {
	ctrlAcc, err := chn.GetCessAccount()
	if err != nil {
		return "", "", "", "", "", err
	}
	baseDir := filepath.Join(cfg.GetMountedPath(), ctrlAcc, configs.BaseDir)

	_, err = os.Stat(baseDir)
	if err != nil {
		err = os.MkdirAll(baseDir, configs.DirPermission)
		if err != nil {
			return "", "", "", "", "", err
		}
	}

	logDir := filepath.Join(baseDir, configs.LogDir)
	if err := os.MkdirAll(logDir, configs.DirPermission); err != nil {
		return "", "", "", "", "", err
	}

	cacheDir := filepath.Join(baseDir, configs.CacheDir)
	if err := os.MkdirAll(cacheDir, configs.DirPermission); err != nil {
		return "", "", "", "", "", err
	}

	fillerDir := filepath.Join(baseDir, configs.FillerDir)
	if err := os.MkdirAll(fillerDir, configs.DirPermission); err != nil {
		return "", "", "", "", "", err
	}

	fileDir := filepath.Join(baseDir, configs.FileDir)
	if err := os.MkdirAll(fileDir, configs.DirPermission); err != nil {
		return "", "", "", "", "", err
	}

	tmpDir := filepath.Join(baseDir, configs.TmpDir)
	if err := os.MkdirAll(tmpDir, configs.DirPermission); err != nil {
		return "", "", "", "", "", err
	}

	log.Println(baseDir)
	return logDir, cacheDir, fillerDir, fileDir, tmpDir, nil
}

func buildCache(cacheDir string) (db.ICache, error) {
	return db.NewCache(cacheDir, 0, 0, configs.NameSpace)
}

func buildLogs(logDir string) (logger.ILog, error) {
	var logs_info = make(map[string]string)
	for _, v := range configs.LogFiles {
		logs_info[v] = filepath.Join(logDir, v+".log")
	}
	return logger.NewLogs(logs_info)
}

func buildServer(name string, port int, chain chain.IChain, logs logger.ILog, cach db.ICache, filedir, tmpDir string) serve.IServer {
	// NewServer
	s := serve.NewServer(name, "0.0.0.0", port)

	// Configure Routes
	s.AddRouter(serve.Msg_Ping, &serve.PingRouter{})
	s.AddRouter(serve.Msg_Auth, &serve.AuthRouter{})
	s.AddRouter(serve.Msg_File, &serve.FileRouter{Chain: chain, Logs: logs, Cach: cach, FileDir: filedir, TmpDir: tmpDir})

	return s
}
