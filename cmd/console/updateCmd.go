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
	cess "github.com/CESSProject/cess-go-sdk"
	"github.com/CESSProject/cess-go-sdk/config"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/spf13/cobra"
)

const update_cmd = "update"
const update_cmd_use = "update"
const update_cmd_short = "update inforation"
const update_earnings_cmd = "earnings"
const update_earnings_cmd_use = update_earnings_cmd
const update_earnings_cmd_short = "update earnings account"

var updateCmd = &cobra.Command{
	Use:   update_cmd,
	Short: update_cmd_short,
	Run: func(cmd *cobra.Command, args []string) {
		updateEarningsAccount(cmd)
		cmd.Help()
	},
	DisableFlagsInUseLine: true,
}

var updateEarningsCmd = &cobra.Command{
	Use:   update_earnings_cmd_use,
	Short: update_earnings_cmd_short,
	Run: func(cmd *cobra.Command, args []string) {
		updateEarningsAccount(cmd)
	},
	DisableSuggestions: true,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.AddCommand(updateEarningsCmd)
}

// updateIncomeAccount
func updateEarningsAccount(cmd *cobra.Command) {
	var (
		err error
		n   = node.New()
	)

	if len(os.Args) < 3 {
		configs.Err("Please enter your earnings account")
		os.Exit(1)
	}

	err = sutils.VerityAddress(os.Args[3], sutils.CessPrefix)
	if err != nil {
		configs.Err(err.Error())
		os.Exit(1)
	}

	// Build profile instances
	n.Confile, err = buildAuthenticationConfig(cmd)
	if err != nil {
		configs.Err(err.Error())
		os.Exit(1)
	}

	//Build client
	n.SDK, err = cess.New(
		config.CharacterName_Bucket,
		cess.ConnectRpcAddrs(n.GetRpcAddr()),
		cess.Mnemonic(n.GetMnemonic()),
		cess.TransactionTimeout(configs.TimeToWaitEvent),
	)
	if err != nil {
		configs.Err(err.Error())
		os.Exit(1)
	}

	txhash, err := n.UpdateEarningsAccount(os.Args[3])
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
