/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	cess "github.com/CESSProject/cess-go-sdk"
	"github.com/CESSProject/cess-go-sdk/config"
	"github.com/CESSProject/p2p-go/out"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

const (
	reward_cmd       = "reward"
	reward_cmd_use   = "reward"
	reward_cmd_short = "Query reward information"
)

var rewardCmd = &cobra.Command{
	Use:                   reward_cmd_use,
	Short:                 reward_cmd_short,
	Run:                   Command_Reward_Runfunc,
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(rewardCmd)
}

// Exit
func Command_Reward_Runfunc(cmd *cobra.Command, args []string) {
	var (
		err error
		n   = node.NewEmptyNode()
	)

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

	rewardInfo, err := n.QueryRewards(n.GetSignatureAccPulickey())
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}
	var total string
	var claimed string
	var sep uint8 = 0
	for i := len(rewardInfo.Total) - 1; i >= 0; i-- {
		total = fmt.Sprintf("%c%s", rewardInfo.Total[i], total)
		sep++
		if sep%3 == 0 {
			total = fmt.Sprintf("_%s", total)
		}
	}
	total = strings.TrimPrefix(total, "_")

	sep = 0
	for i := len(rewardInfo.Claimed) - 1; i >= 0; i-- {
		claimed = fmt.Sprintf("%c%s", rewardInfo.Claimed[i], claimed)
		sep++
		if sep%3 == 0 {
			claimed = fmt.Sprintf("_%s", claimed)
		}
	}
	claimed = strings.TrimPrefix(claimed, "_")

	var tableRows = []table.Row{
		{"total reward", total},
		{"claimed reward", claimed},
	}
	tw := table.NewWriter()
	tw.AppendRows(tableRows)
	fmt.Println(tw.Render())
	os.Exit(0)
}
