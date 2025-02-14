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
	out "github.com/CESSProject/cess-miner/pkg/fout"
	"github.com/spf13/cobra"
)

const (
	withdraw_cmd       = "withdraw"
	withdraw_cmd_use   = "withdraw"
	withdraw_cmd_short = "Withdraw staking"
)

var withdrawCmd = &cobra.Command{
	Use:                   withdraw_cmd_use,
	Short:                 withdraw_cmd_short,
	Run:                   withdrawCmdFunc,
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(withdrawCmd)
}

// Withdraw the staking
func withdrawCmdFunc(cmd *cobra.Command, args []string) {
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

	err = cli.InitExtrinsicsNameForMiner()
	if err != nil {
		out.Err("Please verify the RPC version and ensure it has been synchronized to the latest state.")
		os.Exit(1)
	}

	blockhash, err := cli.MinerWithdraw()
	if err != nil {
		if blockhash == "" {
			out.Err(err.Error())
			os.Exit(1)
		}
		out.Warn(blockhash)
		os.Exit(0)
	}

	out.Ok(blockhash)
	os.Exit(0)
}
