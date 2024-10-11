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

	cess "github.com/CESSProject/cess-go-sdk"
	"github.com/CESSProject/cess-go-sdk/chain"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/cess-miner/configs"
	out "github.com/CESSProject/cess-miner/pkg/fout"
	"github.com/btcsuite/btcutil/base58"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

const (
	stat_cmd       = "stat"
	stat_cmd_use   = "stat"
	stat_cmd_short = "Query storage miner information"
)

var statCmd = &cobra.Command{
	Use:                   stat_cmd_use,
	Short:                 stat_cmd_short,
	Run:                   statCmdFunc,
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(statCmd)
}

func statCmdFunc(cmd *cobra.Command, args []string) {
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

	// query your own information on the chain
	minerInfo, err := cli.QueryMinerItems(cli.GetSignatureAccPulickey(), -1)
	if err != nil {
		if err.Error() != chain.ERR_Empty {
			out.Err(chain.ERR_RPC_CONNECTION.Error())
		} else {
			out.Err("you are not registered as a storage miner, possible cause: 1.insufficient balance in signature account 2.wrong rpc address")
		}
		os.Exit(1)
	}

	minerInfo.Collaterals.Div(new(big.Int).SetBytes(minerInfo.Collaterals.Bytes()), big.NewInt(configs.TokenTCESS))
	minerInfo.Debt.Div(new(big.Int).SetBytes(minerInfo.Debt.Bytes()), big.NewInt(configs.TokenTCESS))

	beneficiaryAcc, _ := sutils.EncodePublicKeyAsCessAccount(minerInfo.BeneficiaryAccount[:])

	startBlock, err := cli.QueryStakingStartBlock(cli.GetSignatureAccPulickey(), -1)
	if err != nil {
		if err.Error() != chain.ERR_Empty {
			out.Err(chain.ERR_RPC_CONNECTION.Error())
			os.Exit(1)
		} else {
			out.Err("your staking starting block is not found")
		}
	}

	var stakingAcc = cfg.ReadStakingAcc()
	if stakingAcc == "" {
		stakingAcc = cli.GetSignatureAcc()
	}

	var tableRows = []table.Row{
		{"name", "storage miner"},
		{"peer id", base58.Encode([]byte(string(minerInfo.PeerId[:])))},
		{"state", string(minerInfo.State)},
		{"staking amount", fmt.Sprintf("%v %s", minerInfo.Collaterals, cli.GetTokenSymbol())},
		{"staking start", startBlock},
		{"debt amount", fmt.Sprintf("%v %s", minerInfo.Debt, cli.GetTokenSymbol())},
		{"declaration space", unitConversion(minerInfo.DeclarationSpace)},
		{"validated space", unitConversion(minerInfo.IdleSpace)},
		{"used space", unitConversion(minerInfo.ServiceSpace)},
		{"locked space", unitConversion(minerInfo.LockSpace)},
		{"signature account", cli.GetSignatureAcc()},
		{"staking account", stakingAcc},
		{"earnings account", beneficiaryAcc},
	}
	tw := table.NewWriter()
	tw.AppendRows(tableRows)
	fmt.Println(tw.Render())
	os.Exit(0)
}

func unitConversion(value types.U128) string {
	var result string
	if value.IsUint64() {
		v := value.Uint64()
		if v >= (chain.SIZE_1GiB * 1024 * 1024 * 1024) {
			result = fmt.Sprintf("%.2f EiB", float64(float64(v)/float64(chain.SIZE_1GiB*1024*1024*1024)))
			return result
		}
		if v >= (chain.SIZE_1GiB * 1024 * 1024) {
			result = fmt.Sprintf("%.2f PiB", float64(float64(v)/float64(chain.SIZE_1GiB*1024*1024)))
			return result
		}
		if v >= (chain.SIZE_1GiB * 1024) {
			result = fmt.Sprintf("%.2f TiB", float64(float64(v)/float64(chain.SIZE_1GiB*1024)))
			return result
		}
		if v >= (chain.SIZE_1GiB) {
			result = fmt.Sprintf("%.2f GiB", float64(float64(v)/float64(chain.SIZE_1GiB)))
			return result
		}
		if v >= (chain.SIZE_1MiB) {
			result = fmt.Sprintf("%.2f MiB", float64(float64(v)/float64(chain.SIZE_1MiB)))
			return result
		}
		if v >= (chain.SIZE_1KiB) {
			result = fmt.Sprintf("%.2f KiB", float64(float64(v)/float64(chain.SIZE_1KiB)))
			return result
		}
		result = fmt.Sprintf("%v Bytes", v)
		return result
	}
	v := new(big.Int).SetBytes(value.Bytes())
	v.Quo(v, new(big.Int).SetUint64((chain.SIZE_1GiB * 1024 * 1024 * 1024)))
	result = fmt.Sprintf("%v EiB", v)
	return result
}
