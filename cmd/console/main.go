/*
   Copyright 2022 CESS (Cumulus Encrypted Storage System) authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
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

// init
func init() {
	rootCmd.AddCommand(
		defaultCommand(),
		versionCommand(),
		stateCommand(),
		runCommand(),
		exitCommand(),
		increaseCommand(),
		withdrawCommand(),
		updateAddrCommand(),
		updateIncomeCommand(),
		rewardCommand(),
	)
	rootCmd.PersistentFlags().StringP("config", "c", "", "Specify the configuration file")
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

func versionCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(configs.Version)
			os.Exit(0)
		},
		DisableFlagsInUseLine: true,
	}
	return cc
}

func defaultCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "default",
		Short:                 "Generate configuration file template",
		Run:                   defaultCmd,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func stateCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "state",
		Short:                 "Query mining miner information",
		Run:                   stateCmd,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func runCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "run",
		Short:                 "Register and start mining",
		Run:                   runCmd,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func exitCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "exit",
		Short:                 "Exit the mining platform",
		Run:                   exitCmd,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func increaseCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "increase <number of tokens>",
		Short:                 "Increase the deposit of mining miner",
		Run:                   increaseCmd,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func withdrawCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "withdraw",
		Short:                 "Redemption deposit",
		Run:                   withdrawCmd,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func updateAddrCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "update_address",
		Short:                 "Update the miner's access address",
		Example:               "bucket update_address ip:port",
		Run:                   updateAddrCmd,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func updateIncomeCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "update_income",
		Short:                 "Update the miner's income account",
		Run:                   updateIncomeCmd,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func rewardCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "reward",
		Short:                 "Miners receive their own rewards",
		Run:                   rewardCmd,
		DisableFlagsInUseLine: true,
	}
	return cc
}
