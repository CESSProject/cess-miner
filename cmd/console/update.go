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
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/cess-miner/configs"
	out "github.com/CESSProject/cess-miner/pkg/fout"
	"github.com/spf13/cobra"
)

const update_cmd = "update"
const update_cmd_use = update_cmd
const update_cmd_short = "Update inforation [earnings | endpoint]"

const update_earnings_cmd = "earnings"
const update_earnings_cmd_use = update_earnings_cmd
const update_earnings_cmd_short = "Update earnings account"

const update_endpoint_cmd = "endpoint"
const update_endpoint_cmd_use = update_endpoint_cmd
const update_endpoint_cmd_short = "Update endpoint"

var updateCmd = &cobra.Command{
	Use:                   update_cmd_use,
	Short:                 update_cmd_short,
	DisableFlagsInUseLine: true,
}

var updateEarningsCmd = &cobra.Command{
	Use:                update_earnings_cmd_use,
	Short:              update_earnings_cmd_short,
	Run:                updearningsCmdFunc,
	DisableSuggestions: true,
}

var updateEndpointCmd = &cobra.Command{
	Use:                update_endpoint_cmd_use,
	Short:              update_endpoint_cmd_short,
	Run:                updendpointCmdFunc,
	DisableSuggestions: true,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.AddCommand(updateEarningsCmd)
	updateCmd.AddCommand(updateEndpointCmd)
}

func updearningsCmdFunc(cmd *cobra.Command, args []string) {
	if len(os.Args) < 3 {
		out.Err("Please enter your earnings account")
		os.Exit(1)
	}

	err := sutils.VerityAddress(os.Args[3], sutils.CessPrefix)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}

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

	blockhash, err := cli.UpdateBeneficiary(os.Args[3])
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

func updendpointCmdFunc(cmd *cobra.Command, args []string) {
	if len(os.Args) < 3 {
		out.Err("Please enter your earnings account")
		os.Exit(1)
	}

	if len(os.Args[3]) <= 0 {
		out.Err("empty endponit")
		os.Exit(1)
	}

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

	blockhash, err := cli.UpdateSminerEndpoint([]byte(os.Args[3]))
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
