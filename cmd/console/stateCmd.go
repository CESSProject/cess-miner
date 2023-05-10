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
	"github.com/CESSProject/sdk-go/core/client"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// Query miner state
func Command_State_Runfunc(cmd *cobra.Command, args []string) {
	var (
		ok  bool
		err error
		n   = node.New()
	)

	// Build profile instances
	n.Cfg, err = buildConfigFile(cmd, "", 0)
	if err != nil {
		logERR(err.Error())
		os.Exit(1)
	}

	// Build client
	cli, err := sdkgo.New(
		configs.Name,
		sdkgo.ConnectRpcAddrs(n.Cfg.GetRpcAddr()),
		sdkgo.ListenPort(n.Cfg.GetServicePort()),
		sdkgo.Workspace(n.Cfg.GetWorkspace()),
		sdkgo.Mnemonic(n.Cfg.GetMnemonic()),
		sdkgo.TransactionTimeout(configs.TimeToWaitEvent),
	)
	if err != nil {
		logERR(err.Error())
		os.Exit(1)
	}

	n.Cli, ok = cli.(*client.Cli)
	if !ok {
		logERR("Invalid client type")
		os.Exit(1)
	}

	//Query your own information on the chain
	minerInfo, err := n.Cli.QueryStorageMiner(n.Cfg.GetPublickey())
	if err != nil {
		logERR(err.Error())
		os.Exit(1)
	}

	minerInfo.Collaterals.Div(new(big.Int).SetBytes(minerInfo.Collaterals.Bytes()), big.NewInt(1000000000000))

	beneficiaryAcc, _ := utils.EncodeToCESSAddr(minerInfo.BeneficiaryAcc[:])

	var tableRows = []table.Row{
		{"peer id", string(minerInfo.PeerId[:])},
		{"state", string(minerInfo.State)},
		{"staking amount", fmt.Sprintf("%v TCESS", minerInfo.Collaterals)},
		{"validated space", fmt.Sprintf("%v bytes", minerInfo.IdleSpace)},
		{"used space", fmt.Sprintf("%v bytes", minerInfo.ServiceSpace)},
		{"locked space", fmt.Sprintf("%v bytes", minerInfo.LockSpace)},
		{"staking account", n.Cfg.GetAccount()},
		{"earnings account", beneficiaryAcc},
	}
	tw := table.NewWriter()
	tw.AppendRows(tableRows)
	fmt.Println(tw.Render())
	os.Exit(0)
}
