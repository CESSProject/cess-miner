/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"os"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	sdkgo "github.com/CESSProject/sdk-go"
	"github.com/CESSProject/sdk-go/core/client"
	"github.com/spf13/cobra"
)

const update_cmd = "update"
const update_cmd_use = "update"
const update_cmd_short = "update inforation"
const update_earnings_cmd = "earnings"
const update_earnings_cmd_use = update_cmd + update_earnings_cmd + " <new earnings account>"
const update_earnings_cmd_short = "Update earnings account"

var updateCmd = &cobra.Command{
	Use:   update_cmd,
	Short: update_cmd_short,
	Run: func(cmd *cobra.Command, args []string) {
		updateEarningsAccount(cmd)
		cmd.Help()
	},
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
}

var updateEarningsCmd = &cobra.Command{
	Use:   update_earnings_cmd_use,
	Short: update_earnings_cmd_short,
	Run: func(cmd *cobra.Command, args []string) {
		updateEarningsAccount(cmd)
	},
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.AddCommand(updateEarningsCmd)
}

// updateIncomeAccount
func updateEarningsAccount(cmd *cobra.Command) {
	var (
		ok  bool
		err error
		n   = node.New()
	)

	if len(os.Args) < 3 {
		configs.Err("Please enter your earnings account")
		os.Exit(1)
	}

	err = utils.VerityAddress(os.Args[3], utils.CESSChainTestPrefix)
	if err != nil {
		configs.Err(err.Error())
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

	txhash, err := n.Cli.UpdateIncomeAccount(os.Args[3])
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
