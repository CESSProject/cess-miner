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
	out "github.com/CESSProject/cess-miner/pkg/fout"
	"github.com/spf13/cobra"
)

const (
	config_cmd       = "config"
	config_cmd_short = "Generate configuration file"
)

var configCmd = &cobra.Command{
	Use:                   config_cmd,
	Short:                 config_cmd_short,
	Run:                   configCmdFunc,
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(configCmd)
}

// configCmdFunc generate a configuration file template
func configCmdFunc(cmd *cobra.Command, args []string) {
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
