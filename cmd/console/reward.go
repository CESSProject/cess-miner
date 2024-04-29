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
	"strings"

	"github.com/CESSProject/cess-bucket/configs"
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
	cfg, err := buildAuthenticationConfig(cmd)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}

	cli, err := cess.New(
		context.Background(),
		cess.Name(config.CharacterName_Bucket),
		cess.ConnectRpcAddrs(cfg.ReadRpcEndpoints()),
		cess.Mnemonic(cfg.ReadMnemonic()),
		cess.TransactionTimeout(configs.TimeToWaitEvent),
	)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}
	defer cli.Close()

	rewardInfo, err := cli.QueryRewards(cli.GetSignatureAccPulickey())
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}
	var total string
	var claimed string
	var unclaimed string

	if len(rewardInfo.Total) == 0 {
		rewardInfo.Total = "0"
	}
	if len(rewardInfo.Claimed) == 0 {
		rewardInfo.Claimed = "0"
	}

	t, ok := new(big.Int).SetString(rewardInfo.Total, 10)
	if !ok {
		out.Err(err.Error())
		os.Exit(1)
	}
	c, ok := new(big.Int).SetString(rewardInfo.Claimed, 10)
	if !ok {
		out.Err(err.Error())
		os.Exit(1)
	}

	t = t.Sub(t, c)
	u := t.String()

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

	sep = 0
	for i := len(u) - 1; i >= 0; i-- {
		unclaimed = fmt.Sprintf("%c%s", u[i], unclaimed)
		sep++
		if sep%3 == 0 {
			unclaimed = fmt.Sprintf("_%s", unclaimed)
		}
	}
	unclaimed = strings.TrimPrefix(unclaimed, "_")

	var tableRows = []table.Row{
		{"total reward", total},
		{"claimed reward", claimed},
		{"unclaimed reward", unclaimed},
	}
	tw := table.NewWriter()
	tw.AppendRows(tableRows)
	fmt.Println(tw.Render())
	os.Exit(0)
}
