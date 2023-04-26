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
	"path/filepath"
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
	var power, space float32
	var power_unit, space_unit string
	count := 0
	for minerInfo.Power.BitLen() > int(16) {
		minerInfo.Power.Div(new(big.Int).SetBytes(minerInfo.Power.Bytes()), big.NewInt(1024))
		count++
	}
	if minerInfo.Power.Int64() > 1024 {
		power = float32(minerInfo.Power.Int64()) / float32(1024)
		count++
	} else {
		power = float32(minerInfo.Power.Int64())
	}
	switch count {
	case 0:
		power_unit = "Byte"
	case 1:
		power_unit = "KiB"
	case 2:
		power_unit = "MiB"
	case 3:
		power_unit = "GiB"
	case 4:
		power_unit = "TiB"
	case 5:
		power_unit = "PiB"
	case 6:
		power_unit = "EiB"
	case 7:
		power_unit = "ZiB"
	case 8:
		power_unit = "YiB"
	case 9:
		power_unit = "NiB"
	case 10:
		power_unit = "DiB"
	default:
		power_unit = fmt.Sprintf("DiB(%v)", count-10)
	}
	count = 0
	for minerInfo.Space.BitLen() > int(16) {
		minerInfo.Space.Div(new(big.Int).SetBytes(minerInfo.Space.Bytes()), big.NewInt(1024))
		count++
	}
	if minerInfo.Space.Int64() > 1024 {
		space = float32(minerInfo.Space.Int64()) / float32(1024)
		count++
	} else {
		space = float32(minerInfo.Space.Int64())
	}

	switch count {
	case 0:
		space_unit = "Byte"
	case 1:
		space_unit = "KiB"
	case 2:
		space_unit = "MiB"
	case 3:
		space_unit = "GiB"
	case 4:
		space_unit = "TiB"
	case 5:
		space_unit = "PiB"
	case 6:
		space_unit = "EiB"
	case 7:
		space_unit = "ZiB"
	case 8:
		space_unit = "YiB"
	case 9:
		space_unit = "NiB"
	case 10:
		space_unit = "DiB"
	default:
		power_unit = fmt.Sprintf("DiB(%v)", count-10)
	}

	addr := filepath.Base(string(minerInfo.Ip))
	//print your own details
	fmt.Printf("MinerId: C%v\nState: %v\nStorageSpace: %.2f %v\nUsedSpace: %.2f %v\nStakestakes: %v TCESS\nServiceAddr: %v\n",
		minerInfo.PeerId, string(minerInfo.State), power, power_unit, space, space_unit, minerInfo.Collaterals, string(addr))
	os.Exit(0)
}
