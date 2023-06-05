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
	"github.com/CESSProject/sdk-go/config"
	"github.com/CESSProject/sdk-go/core/pattern"
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
		configs.Err("Please enter the stakes amount")
		os.Exit(1)
	}

	stakes, ok := new(big.Int).SetString(os.Args[2]+pattern.TokenPrecision_CESS, 10)
	if !ok {
		configs.Err("Please enter the correct stakes amount")
		os.Exit(1)
	}

	// Build profile instances
	n.Confile, err = buildAuthenticationConfig(cmd)
	if err != nil {
		configs.Err(err.Error())
		os.Exit(1)
	}

	//Build client
	n.SDK, err = sdkgo.New(
		config.CharacterName_Bucket,
		sdkgo.ConnectRpcAddrs(n.GetRpcAddr()),
		sdkgo.Mnemonic(n.GetMnemonic()),
		sdkgo.TransactionTimeout(configs.TimeToWaitEvent),
	)
	if err != nil {
		configs.Err(err.Error())
		os.Exit(1)
	}

	txhash, err := n.IncreaseStakingAmount(stakes)
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
