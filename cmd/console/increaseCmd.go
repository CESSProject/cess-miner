/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"math/big"
	"os"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	sdkgo "github.com/CESSProject/sdk-go"
	"github.com/CESSProject/sdk-go/core/client"
	"github.com/CESSProject/sdk-go/core/rule"
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
		ok  bool
		err error
		n   = node.New()
	)

	if len(os.Args) < 3 {
		configs.Err("Please enter the stakes amount")
		os.Exit(1)
	}

	stakes, ok := new(big.Int).SetString(os.Args[2]+rule.CESSTokenDecimals, 10)
	if !ok {
		configs.Err("Please enter the correct stakes amount")
		os.Exit(1)
	}

	// Build profile instances
	n.Cfg, err = buildConfigFile(cmd, "", 0)
	if err != nil {
		configs.Err(err.Error())
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
		configs.Err(err.Error())
		os.Exit(1)
	}
	n.Cli, ok = cli.(*client.Cli)
	if !ok {
		configs.Err("Invalid client type")
		os.Exit(1)
	}

	txhash, err := n.Cli.IncreaseStakes(stakes)
	if err != nil {
		if txhash == "" {
			configs.Err(err.Error())
			os.Exit(1)
		}
		configs.Warn(txhash)
		os.Exit(0)
	}

	configs.Ok(txhash)
	os.Exit(0)
}
