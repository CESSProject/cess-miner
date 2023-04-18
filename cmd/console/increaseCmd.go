/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"os"
	"strconv"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	sdkgo "github.com/CESSProject/sdk-go"
	"github.com/spf13/cobra"
)

const increase_cmd = "increase"

var increaseCmd = &cobra.Command{
	Use:                   increase_cmd + " <stakes amount>",
	Short:                 "Increase the stakes of storage miner",
	Run:                   Command_Increase_Runfunc,
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(increaseCmd)
}

// Increase stakes
func Command_Increase_Runfunc(cmd *cobra.Command, args []string) {
	var (
		err error
		n   = node.New()
	)

	if len(os.Args) < 3 {
		logERR("Please enter the stakes amount")
		os.Exit(1)
	}

	_, err = strconv.ParseUint(os.Args[2], 10, 64)
	if err != nil {
		logERR("Please enter the correct stakes amount")
		os.Exit(1)
	}

	// Build profile instances
	n.Cfg, err = buildConfigFile(cmd, "", 0)
	if err != nil {
		logERR(err.Error())
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
		logERR(err.Error())
		os.Exit(1)
	}
	txhash, err := n.Cli.IncreaseStakes(os.Args[2])
	if err != nil {
		if txhash == "" {
			logERR(err.Error())
			os.Exit(1)
		}
		logWARN(txhash)
		os.Exit(0)
	}

	logOK(txhash)
	os.Exit(0)
}
