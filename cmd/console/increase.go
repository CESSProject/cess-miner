/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strconv"

	cess "github.com/CESSProject/cess-go-sdk"
	"github.com/CESSProject/cess-go-sdk/chain"
	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/pkg/confile"
	out "github.com/CESSProject/cess-miner/pkg/fout"
	"github.com/spf13/cobra"
)

const increase_cmd = "increase"
const increase_cmd_use = increase_cmd
const increase_cmd_short = "Increase [staking | space]"

const increaseStaking_cmd = "staking"
const increaseStaking_cmd_use = increaseStaking_cmd
const increaseStaking_cmd_short = "Increase staking, the unit is TCESS"

const increaseSpace_cmd = "space"
const increaseSpace_cmd_use = increaseSpace_cmd
const increaseSpace_cmd_short = "Increase space, the unit is TiB"

var increaseCmd = &cobra.Command{
	Use:                   increase_cmd_use,
	Short:                 increase_cmd_short,
	DisableFlagsInUseLine: true,
}

var increaseStakingCmd = &cobra.Command{
	Use:   increaseStaking_cmd_use,
	Short: increaseStaking_cmd_short,
	Run: func(cmd *cobra.Command, args []string) {
		incstakingCmdFunc(cmd, args)
	},
	DisableFlagsInUseLine: true,
}

var increaseSpaceCmd = &cobra.Command{
	Use:   increaseSpace_cmd_use,
	Short: increaseSpace_cmd_short,
	Run: func(cmd *cobra.Command, args []string) {
		incspaceCmdFunc(cmd, args)
	},
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(increaseCmd)
	increaseCmd.AddCommand(increaseStakingCmd)
	increaseCmd.AddCommand(increaseSpaceCmd)
}

// increase staking
func incstakingCmdFunc(cmd *cobra.Command, args []string) {
	if len(os.Args) < 4 {
		out.Err("Please enter the staking amount, the unit is TCESS")
		os.Exit(1)
	}

	_, err := strconv.ParseUint(os.Args[3], 10, 64)
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
		out.Err("The rpc address does not match the software version, please check the rpc address.")
		os.Exit(1)
	}
	addStaking := os.Args[3] + chain.TokenPrecision_CESS
	txhash, err := cli.IncreaseCollateral(cli.GetSignatureAccPulickey(), addStaking)
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
func incspaceCmdFunc(cmd *cobra.Command, args []string) {
	if len(os.Args) < 4 {
		out.Err("Please enter the space size to be increased in TiB")
		os.Exit(1)
	}

	space, err := strconv.Atoi(os.Args[3])
	if err != nil {
		out.Err("Please enter the correct space size")
		os.Exit(1)
	}

	cfg := confile.NewConfigFile()
	config_file, err := parseArgs_config(cmd)
	if err != nil {
		cfg, err = buildConfigItems(cmd)
		if err != nil {
			out.Err(fmt.Sprintf("build config items err: %v", err))
			os.Exit(1)
		}
	} else {
		cfg, err = parseConfigFile(config_file)
		if err != nil {
			out.Err(fmt.Sprintf("parse config file err: %v", err))
			os.Exit(1)
		}
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
		out.Err("The rpc address does not match the software version, please check the rpc address.")
		os.Exit(1)
	}

	minerInfo, err := cli.QueryMinerItems(cli.GetSignatureAccPulickey(), -1)
	if err != nil {
		if err.Error() != chain.ERR_Empty {
			out.Err(err.Error())
			os.Exit(1)
		}
		out.Err("You are not a storage miner: " + cli.GetSignatureAcc())
		os.Exit(1)
	}

	accInfo, err := cli.QueryAccountInfoByAccountID(minerInfo.StakingAccount[:], -1)
	if err != nil {
		if err.Error() != chain.ERR_Empty {
			out.Err(err.Error())
			os.Exit(1)
		}
		out.Err("staking account does not exist, possible: 1.balance is empty 2.rpc address error")
		os.Exit(1)
	}

	token := space * chain.StakingStakePerTiB
	token_cess, _ := new(big.Int).SetString(fmt.Sprintf("%d%s", token, chain.TokenPrecision_CESS), 10)
	if accInfo.Data.Free.CmpAbs(token_cess) < 0 {
		out.Err(fmt.Sprintf("staking account balance less than %d %s", token, cli.GetTokenSymbol()))
		os.Exit(1)
	}

	txhash, err := cli.IncreaseDeclarationSpace(uint32(space))
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
