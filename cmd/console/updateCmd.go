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
	"github.com/spf13/cobra"
)

const update_cmd = "update"

var updateCmd = &cobra.Command{
	Use:                   update_cmd,
	Short:                 "update income account",
	Run:                   Command_UpdateIncome_Runfunc,
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

// Increase stakes
func Command_UpdateIncome_Runfunc(cmd *cobra.Command, args []string) {
	var (
		err error
		n   = node.New()
	)

	if len(os.Args) < 3 {
		logERR("Please enter your income account")
		os.Exit(1)
	}

	err = utils.VerityAddress(os.Args[2], utils.CESSChainTestPrefix)
	if err != nil {
		logERR(err.Error())
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
	txhash, err := n.Cli.UpdateIncomeAccount(os.Args[2])
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
