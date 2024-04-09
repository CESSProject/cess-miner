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
	"strconv"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	cess "github.com/CESSProject/cess-go-sdk"
	"github.com/CESSProject/cess-go-sdk/config"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/p2p-go/out"
	"github.com/spf13/cobra"
)

const increase_cmd = "increase"
const increase_cmd_use = increase_cmd
const increase_cmd_short = "increase [staking | space]"

const increaseStaking_cmd = "staking"
const increaseStaking_cmd_use = increaseStaking_cmd
const increaseStaking_cmd_short = "increase staking"

const increaseSpace_cmd = "space"
const increaseSpace_cmd_use = increaseSpace_cmd
const increaseSpace_cmd_short = "increase space"

var increaseCmd = &cobra.Command{
	Use:                   increase_cmd_use,
	Short:                 increase_cmd_short,
	DisableFlagsInUseLine: true,
}

var increaseStakingCmd = &cobra.Command{
	Use:   increaseStaking_cmd_use,
	Short: increaseStaking_cmd_short,
	Run: func(cmd *cobra.Command, args []string) {
		increaseStakingCmd_Runfunc(cmd, args)
	},
	DisableFlagsInUseLine: true,
}

var increaseSpaceCmd = &cobra.Command{
	Use:   increaseSpace_cmd_use,
	Short: increaseSpace_cmd_short,
	Run: func(cmd *cobra.Command, args []string) {
		increaseSpaceCmd_Runfunc(cmd, args)
	},
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(increaseCmd)
	increaseCmd.AddCommand(increaseStakingCmd)
	increaseCmd.AddCommand(increaseSpaceCmd)
}

// increase staking
func increaseStakingCmd_Runfunc(cmd *cobra.Command, args []string) {
	var (
		err error
		n   = node.NewEmptyNode()
	)

	if len(os.Args) < 4 {
		out.Err("Please enter the staking amount, the unit is TCESS")
		os.Exit(1)
	}

	stakes, ok := new(big.Int).SetString(os.Args[3]+pattern.TokenPrecision_CESS, 10)
	if !ok {
		out.Err("Please enter the correct staking amount")
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
		cess.Name(config.CharacterName_Bucket),
		cess.ConnectRpcAddrs(n.GetRpcAddr()),
		cess.Mnemonic(n.GetMnemonic()),
		cess.TransactionTimeout(configs.TimeToWaitEvent),
	)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}
	defer n.GetSubstrateAPI().Client.Close()

	txhash, err := n.IncreaseStakingAmount(n.GetSignatureAcc(), stakes)
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

// increase space
func increaseSpaceCmd_Runfunc(cmd *cobra.Command, args []string) {
	var (
		err error
		n   = node.NewEmptyNode()
	)

	if len(os.Args) < 4 {
		out.Err("Please enter the space size to be increased in TiB")
		os.Exit(1)
	}

	space, err := strconv.Atoi(os.Args[3])
	if err != nil {
		out.Err("Please enter the correct space size")
		os.Exit(1)
	}

	n.Confile, err = buildAuthenticationConfig(cmd)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}

	n.SDK, err = cess.New(
		context.Background(),
		cess.Name(config.CharacterName_Bucket),
		cess.ConnectRpcAddrs(n.GetRpcAddr()),
		cess.Mnemonic(n.GetMnemonic()),
		cess.TransactionTimeout(configs.TimeToWaitEvent),
	)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}

	txhash, err := n.IncreaseDeclarationSpace(uint32(space))
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
