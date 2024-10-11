/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"fmt"
	"os"

	"github.com/CESSProject/cess-miner/configs"
	"github.com/spf13/cobra"
)

const (
	version_cmd       = "version"
	version_cmd_use   = "version"
	version_cmd_short = "Show version"
)

var versionCmd = &cobra.Command{
	Use:                   version_cmd_use,
	Short:                 version_cmd_short,
	Run:                   versionCmdFunc,
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func versionCmdFunc(cmd *cobra.Command, args []string) {
	fmt.Println(configs.Name + " " + configs.Version)
	os.Exit(0)
}
