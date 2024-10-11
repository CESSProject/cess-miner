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
	"strconv"
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
	start_cmd       = "start"
	start_cmd_use   = "start"
	start_cmd_short = "Running without a configuration file"
)

var startCmd = &cobra.Command{
	Use:                   start_cmd_use,
	Short:                 start_cmd_short,
	Run:                   startCmdFunc,
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func startCmdFunc(cmd *cobra.Command, args []string) {
	node.NewNodeWithConfig(buildConfigs(cmd)).InitNode().Start()
}

func buildConfigs(cmd *cobra.Command) confile.Confiler {
	cfg, err := buildConfigItems(cmd)
	if err != nil {
		out.Err(fmt.Sprintf("build config items err: %v", err))
		os.Exit(1)
	}
	cfg.SetCpuCores(configs.SysInit(cfg.ReadUseCpu()))
	return cfg
}

func buildConfigItems(cmd *cobra.Command) (*confile.Confile, error) {
	var (
		istips      bool
		lines       string
		rpc         []string
		cfg         = confile.NewConfigFile()
		inputReader = bufio.NewReader(os.Stdin)
	)
	rpc, err := cmd.Flags().GetStringSlice("rpcs")
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

	out.Ok(fmt.Sprintf("%v", cfg.ReadRpcEndpoints()))

	workspace, err := cmd.Flags().GetString("workspace")
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
				time.Sleep(time.Second)
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
				time.Sleep(time.Second)
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

	var listenPort uint16
	listenPort, err = cmd.Flags().GetUint16("port")
	if err != nil {
		return cfg, err
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
				time.Sleep(time.Second)
				continue
			}
			lines = strings.ReplaceAll(lines, "\n", "")
			if lines == "" {
				listenPort = configs.DefaultServicePort
			} else {
				n, err := strconv.Atoi(lines)
				if err != nil {
					out.Err("Please enter a number between 1024~65535:")
					continue
				}
				listenPort = uint16(n)
				if listenPort < 1024 {
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
		return cfg, err
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
				time.Sleep(time.Second)
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
						priorityTeeListValues = append(priorityTeeListValues, temp)
					}
				}
			}
			cfg.SetPriorityTeeList(priorityTeeListValues)
			break
		}
	} else {
		cfg.SetPriorityTeeList(priorityTeeList)
	}

	out.Ok(fmt.Sprintf("%v", cfg.ReadPriorityTeeList()))

	var endpoint string
	endpoint, err = cmd.Flags().GetString("endpoint")
	if err != nil {
		return cfg, err
	}
	istips = false
	if endpoint == "" {
		for {
			if !istips {
				out.Input("Enter the endpoint, if you have already registered and don't want to update, press Enter to skip:")
				istips = true
			}
			lines, err = inputReader.ReadString('\n')
			if err != nil {
				out.Err(err.Error())
				time.Sleep(time.Second)
				continue
			}
			endpoint = strings.ReplaceAll(lines, "\n", "")
			cfg.SetEndpoint(endpoint)
			break
		}
	} else {
		cfg.SetEndpoint(endpoint)
	}

	out.Ok(fmt.Sprintf("%v", cfg.ReadApiEndpoint()))

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
