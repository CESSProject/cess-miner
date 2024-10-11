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

	cess "github.com/CESSProject/cess-go-sdk"
	"github.com/CESSProject/cess-miner/configs"
	out "github.com/CESSProject/cess-miner/pkg/fout"
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
	Run:                   rewardCmdFunc,
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(rewardCmd)
}

func rewardCmdFunc(cmd *cobra.Command, args []string) {
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

	rewardInfo, err := cli.QueryRewardMap(cli.GetSignatureAccPulickey(), -1)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}
	var total string
	var totalStr string
	var claimed string
	var claimedStr string
	var unclaimed string

	if len(rewardInfo.TotalReward.Bytes()) == 0 {
		totalStr = "0"
	} else {
		totalStr = rewardInfo.TotalReward.String()
	}
	if len(rewardInfo.RewardIssued.Bytes()) == 0 {
		claimedStr = "0"
	} else {
		claimedStr = rewardInfo.RewardIssued.String()
	}

	t, ok := new(big.Int).SetString(totalStr, 10)
	if !ok {
		out.Err(err.Error())
		os.Exit(1)
	}
	c, ok := new(big.Int).SetString(claimedStr, 10)
	if !ok {
		out.Err(err.Error())
		os.Exit(1)
	}

	t = t.Sub(t, c)
	u := t.String()

	var sep uint8 = 0
	for i := len(totalStr) - 1; i >= 0; i-- {
		total = fmt.Sprintf("%c%s", totalStr[i], total)
		sep++
		if sep%3 == 0 {
			total = fmt.Sprintf("_%s", total)
		}
	}
	total = strings.TrimPrefix(total, "_")

	sep = 0
	for i := len(claimedStr) - 1; i >= 0; i-- {
		claimed = fmt.Sprintf("%c%s", claimedStr[i], claimed)
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
