/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"fmt"
	"math/big"
	"os"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	sdkgo "github.com/CESSProject/sdk-go"
	"github.com/btcsuite/btcutil/base58"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// Query miner state
func Command_State_Runfunc(cmd *cobra.Command, args []string) {
	var (
		err error
		n   = node.New()
	)

	// Build profile instances
	n.Confile, err = buildAuthenticationConfig(cmd)
	if err != nil {
		configs.Err(err.Error())
		os.Exit(1)
	}

	// Build client
	n.SDK, err = sdkgo.New(
		configs.Name,
		sdkgo.ConnectRpcAddrs(n.GetRpcAddr()),
		sdkgo.Mnemonic(n.GetMnemonic()),
		sdkgo.TransactionTimeout(configs.TimeToWaitEvent),
	)
	if err != nil {
		configs.Err(err.Error())
		os.Exit(1)
	}

	//Query your own information on the chain
	minerInfo, err := n.QueryStorageMiner(n.GetStakingPublickey())
	if err != nil {
		configs.Err(err.Error())
		os.Exit(1)
	}

	minerInfo.Collaterals.Div(new(big.Int).SetBytes(minerInfo.Collaterals.Bytes()), big.NewInt(1000000000000))

	beneficiaryAcc, _ := utils.EncodeToCESSAddr(minerInfo.BeneficiaryAcc[:])

	var tableRows = []table.Row{
		{"peer id", base58.Encode([]byte(string(minerInfo.PeerId[:])))},
		{"state", string(minerInfo.State)},
		{"staking amount", fmt.Sprintf("%v TCESS", minerInfo.Collaterals)},
		{"validated space", fmt.Sprintf("%v bytes", minerInfo.IdleSpace)},
		{"used space", fmt.Sprintf("%v bytes", minerInfo.ServiceSpace)},
		{"locked space", fmt.Sprintf("%v bytes", minerInfo.LockSpace)},
		{"staking account", n.GetStakingAcc()},
		{"earnings account", beneficiaryAcc},
	}
	tw := table.NewWriter()
	tw.AppendRows(tableRows)
	fmt.Println(tw.Render())
	os.Exit(0)
}
