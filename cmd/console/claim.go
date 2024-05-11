/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"context"
	"os"

	cess "github.com/CESSProject/cess-go-sdk"
	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/p2p-go/out"
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
	cfg, err := buildAuthenticationConfig(cmd)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}

	cli, err := cess.New(
		context.Background(),
		cess.Name(configs.Name),
		cess.ConnectRpcAddrs(cfg.ReadRpcEndpoints()),
		cess.Mnemonic(cfg.ReadMnemonic()),
		cess.TransactionTimeout(configs.TimeToWaitEvent),
	)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}
	defer cli.Close()

	txhash, err := cli.ReceiveReward()
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
