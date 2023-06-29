/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"fmt"
	"os"

	"github.com/CESSProject/cess-bucket/configs"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   configs.Name,
	Short: configs.Description,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	err := rootCmd.Execute()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
}

// init
func init() {
	configs.SysInit()
	rootCmd.AddCommand(
		Command_Version(),
		Command_State(),
		Command_Run(),
		Command_Withdraw(),
	)
	rootCmd.PersistentFlags().StringP("config", "c", "", "custom configuration file")
	rootCmd.PersistentFlags().StringSliceP("rpc", "", nil, "rpc endpoint list")
	rootCmd.PersistentFlags().StringP("ws", "", "", "workspace")
	rootCmd.PersistentFlags().StringP("earnings", "", "", "earnings account")
	rootCmd.PersistentFlags().IntP("port", "", 0, "listening port")
	rootCmd.PersistentFlags().Uint64P("space", "", 0, "maximum space used (GiB)")
	rootCmd.PersistentFlags().StringSliceP("boot", "", nil, "bootstap node list")
}

func Command_Version() *cobra.Command {
	cc := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(configs.Name + " " + configs.Version)
			os.Exit(0)
		},
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_State() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "stat",
		Short:                 "Query storage miner information",
		Run:                   Command_State_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_Run() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "run",
		Short:                 "Automatically register and run",
		Run:                   runCmd,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_Withdraw() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "withdraw",
		Short:                 "withdraw staking",
		Run:                   Command_Withdraw_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}
