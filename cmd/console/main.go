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
		Command_Version(),
		Command_Register(),
		Command_State(),
		Command_Run(),
		Command_Exit(),
		Command_Increase(),
		Command_Withdraw(),
		Command_UpdateAddress(),
		Command_UpdateIncome(),
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

func Command_Version() *cobra.Command {
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

func Command_Register() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "register",
		Short:                 "Register mining miner information to the chain",
		Run:                   Command_Register_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_State() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "state",
		Short:                 "Query mining miner information",
		Run:                   Command_State_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_Run() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "run",
		Short:                 "Register and start mining",
		Run:                   Command_Run_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_Exit() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "exit",
		Short:                 "Exit the mining platform",
		Run:                   Command_Exit_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_Increase() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "increase <number of tokens>",
		Short:                 "Increase the deposit of mining miner",
		Run:                   Command_Increase_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_Withdraw() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "withdraw",
		Short:                 "Redemption deposit",
		Run:                   Command_Withdraw_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_UpdateAddress() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "update_address",
		Short:                 "Update the miner's access address",
		Example:               "bucket update_address ip:port",
		Run:                   Command_UpdateAddress_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_UpdateIncome() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "update_income",
		Short:                 "Update the miner's income account",
		Run:                   Command_UpdateIncome_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}
