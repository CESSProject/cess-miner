/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"context"
	"math/big"
	"os"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	cess "github.com/CESSProject/cess-go-sdk"
	"github.com/CESSProject/cess-go-sdk/config"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/p2p-go/out"
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
		out.Err("Please enter the stakes amount")
		os.Exit(1)
	}

	stakes, ok := new(big.Int).SetString(os.Args[2]+pattern.TokenPrecision_CESS, 10)
	if !ok {
		out.Err("Please enter the correct stakes amount")
		os.Exit(1)
	}

	// Build profile instances
	n.Confile, err = buildAuthenticationConfig(cmd)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}

	//Build client
	n.SDK, err = cess.New(
		context.Background(),
		config.CharacterName_Bucket,
		cess.ConnectRpcAddrs(n.GetRpcAddr()),
		cess.Mnemonic(n.GetMnemonic()),
		cess.TransactionTimeout(configs.TimeToWaitEvent),
	)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}

	txhash, err := n.IncreaseStakingAmount(stakes)
	if err != nil {
		if txhash == "" {
			out.Err(err.Error())
			os.Exit(1)
		}
		out.Warn(txhash)
		os.Exit(0)
	}

	out.Ok(txhash)
	os.Exit(0)
}
