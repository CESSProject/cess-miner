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
	"strconv"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	sdkgo "github.com/CESSProject/sdk-go"
	"github.com/CESSProject/sdk-go/core/client"
	"github.com/spf13/cobra"
)

// Query miner state
func Command_State_Runfunc(cmd *cobra.Command, args []string) {
	var (
		ok  bool
		err error
		n   = node.New()
	)

	if len(os.Args) < 3 {
		logERR("Please enter the stakes amount")
		os.Exit(1)
	}

	_, err = strconv.ParseUint(os.Args[2], 10, 64)
	if err != nil {
		logERR("Please enter the correct stakes amount")
		os.Exit(1)
	}

	// Build profile instances
	n.Cfg, err = buildConfigFile(cmd, "", 0)
	if err != nil {
		logERR(err.Error())
		os.Exit(1)
	}

	//Build client
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

	//print your own details
	fmt.Printf("PeerId: %v\nState: %v\nFreeSpace: %v \nUsedSpace: %v\nLockedSpace: %v\nStakestakes: %v TCESS\n",
		string(minerInfo.PeerId[:]), string(minerInfo.State), minerInfo.IdleSpace, minerInfo.ServiceSpace, minerInfo.LockSpace, minerInfo.Collaterals)
	os.Exit(0)
}
