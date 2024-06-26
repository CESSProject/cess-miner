/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"os"
	"path/filepath"

	"github.com/CESSProject/cess-miner/pkg/confile"
	"github.com/CESSProject/p2p-go/out"
	"github.com/spf13/cobra"
)

const init_cmd = "config"

var initCmd = &cobra.Command{
	Use:   init_cmd,
	Short: "Generate configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		CreateConfigFile()
		return
	},
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

// Create a configuration file template
func CreateConfigFile() {
	f, err := os.Create(confile.DefaultProfile)
	if err != nil {
		out.Err(err.Error())
		return
	}
	defer f.Close()
	_, err = f.WriteString(confile.TempleteProfile)
	if err != nil {
		out.Err(err.Error())
		return
	}
	err = f.Sync()
	if err != nil {
		out.Err(err.Error())
		return
	}
	pwd, err := os.Getwd()
	if err != nil {
		out.Err(err.Error())
		return
	}
	out.Ok(filepath.Join(pwd, confile.DefaultProfile))
}
