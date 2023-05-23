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
	sdkgo "github.com/CESSProject/sdk-go"
	"github.com/CESSProject/sdk-go/core/client"
	"github.com/spf13/cobra"
)

const (
	claim_cmd       = "claim"
	claim_cmd_use   = "claim"
	claim_cmd_short = "claim reward"
)

var claimCmd = &cobra.Command{
	Use:                   claim_cmd_use,
	Short:                 claim_cmd_short,
	Run:                   Command_Claim_Runfunc,
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(claimCmd)
}

// Exit
func Command_Claim_Runfunc(cmd *cobra.Command, args []string) {
	var (
		ok  bool
		err error
		n   = node.New()
	)

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
	txhash, err := n.Cli.Chain.ClaimRewards()
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
