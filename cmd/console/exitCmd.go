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
	exit_cmd       = "exit"
	exit_cmd_use   = "exit"
	exit_cmd_short = "Unregister the storage miner role"
)

var exitCmd = &cobra.Command{
	Use:                   exit_cmd_use,
	Short:                 exit_cmd_short,
	Run:                   Command_Exit_Runfunc,
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(exitCmd)
}

// Exit
func Command_Exit_Runfunc(cmd *cobra.Command, args []string) {
	var (
		ok  bool
		err error
		n   = node.New()
	)

	// Build profile instances
	n.Cfg, err = buildAuthenticationConfig(cmd)
	if err != nil {
		configs.Err(err.Error())
		os.Exit(1)
	}

	//Build client
	cli, err := sdkgo.New(
		configs.Name,
		sdkgo.ConnectRpcAddrs(n.Cfg.GetRpcAddr()),
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
	txhash, err := n.Cli.Exit(configs.Name)
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
