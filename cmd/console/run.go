/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/node"
	"github.com/CESSProject/cess-miner/pkg/confile"
	out "github.com/CESSProject/cess-miner/pkg/fout"
	"github.com/howeyc/gopass"
	"github.com/spf13/cobra"
)

const (
	run_cmd       = "run"
	run_cmd_use   = "run"
	run_cmd_short = "Running through a configuration file"
)

var runCmd = &cobra.Command{
	Use:                   run_cmd_use,
	Short:                 run_cmd_short,
	Run:                   runCmdFunc,
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

// runCmd run the service with the configuration file
func runCmdFunc(cmd *cobra.Command, args []string) {
	node.NewNodeWithConfig(InitConfigFile(cmd)).InitNode().Start()
}

func InitConfigFile(cmd *cobra.Command) confile.Confiler {
	// parse configuration file
	config_file, err := parseArgs_config(cmd)
	if err != nil {
		out.Err(fmt.Sprintf("parseArgs_config err: %v", err))
		os.Exit(1)
	}
	cfg, err := parseConfigFile(config_file)
	if err != nil {
		out.Err(fmt.Sprintf("parse config file err: %v", err))
		os.Exit(1)
	}
	cfg.SetCpuCores(configs.SysInit(cfg.ReadUseCpu()))
	return cfg
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
	rpc, err = cmd.Flags().GetStringSlice("rpcs")
	if err != nil {
		return cfg, err
	}
	var rpcValus = make([]string, 0)
	if len(rpc) == 0 {
		for {
			if !istips {
				out.Input(fmt.Sprintf("Enter the rpc address of the chain, multiple addresses are separated by spaces, press Enter to skip\nto use [%s] as default rpc address:", configs.DefaultRpcAddr))
				istips = true
			}
			lines, err = inputReader.ReadString('\n')
			if err != nil {
				out.Err(err.Error())
				time.Sleep(time.Second)
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
				rpcValus = []string{configs.DefaultRpcAddr}
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
