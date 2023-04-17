/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"os"
	"path/filepath"

	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/spf13/cobra"
)

var create_cmd = "create"
var create_cmd_config = "config"

var createCmd = &cobra.Command{
	Use:   create_cmd,
	Short: "Create a file",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}
		if args[0] != create_cmd_config {
			cmd.Help()
			return
		}
	},
	DisableFlagsInUseLine: true,
}

var createCmd_config = &cobra.Command{
	Use:   create_cmd_config,
	Short: "config file template",
	Run: func(cmd *cobra.Command, args []string) {
		CreateConfigFile()
		return
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.AddCommand(createCmd_config)
}

// Create a configuration file template
func CreateConfigFile() {
	f, err := os.Create(confile.DefaultProfile)
	if err != nil {
		logERR(err.Error())
		return
	}
	defer f.Close()
	_, err = f.WriteString(confile.TempleteProfile)
	if err != nil {
		logERR(err.Error())
		return
	}
	err = f.Sync()
	if err != nil {
		logERR(err.Error())
		return
	}
	pwd, err := os.Getwd()
	if err != nil {
		logERR(err.Error())
		return
	}
	logOK(filepath.Join(pwd, confile.DefaultProfile))
}
