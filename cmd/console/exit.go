/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"context"
	"os"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	cess "github.com/CESSProject/cess-go-sdk"
	"github.com/CESSProject/cess-go-sdk/config"
	"github.com/CESSProject/p2p-go/out"
	"github.com/spf13/cobra"
)

const (
	exit_cmd       = "exit"
	exit_cmd_use   = "exit"
	exit_cmd_short = "Unregister the storage miner role"
)

var exitCmd = &cobra.Command{
	Use:                   exit_cmd_use,
	Short:                 exit_cmd_short,
	Run:                   Command_Exit_Runfunc,
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(exitCmd)
}

// Exit
func Command_Exit_Runfunc(cmd *cobra.Command, args []string) {
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

	txhash, err := n.ExitSminer(n.GetSignatureAcc())
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
