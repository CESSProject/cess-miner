/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"
	"storage-mining/configs"
	"storage-mining/tools"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	Name        = "mining-cli"
	Description = "Mining program of CESS platform"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   Name,
	Short: Description,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configs.ConfFilePath, "config", "c", "", "Custom profile")
	rootCmd.AddCommand(
		Command_Default(),
		Command_Version(),
		Command_Register(),
		Command_State(),
		Command_Mining(),
		Command_Exit(),
		Command_Increase(),
		Command_Withdraw(),
		Command_Obtain(),
	)
}

func Command_Version() *cobra.Command {
	cc := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run:   Command_Version_Runfunc,
	}
	return cc
}

func Command_Default() *cobra.Command {
	cc := &cobra.Command{
		Use:   "default",
		Short: "Generate profile template",
		Run:   Command_Default_Runfunc,
	}
	return cc
}

func Command_Register() *cobra.Command {
	cc := &cobra.Command{
		Use:   "register",
		Short: "Register miner information to cess chain",
		Run:   Command_Register_Runfunc,
	}
	return cc
}

func Command_State() *cobra.Command {
	cc := &cobra.Command{
		Use:   "state",
		Short: "List miners' own information",
		Run:   Command_State_Runfunc,
	}
	return cc
}

func Command_Mining() *cobra.Command {
	cc := &cobra.Command{
		Use:   "mining",
		Short: "Start mining at CESS mining platform",
		Run:   Command_Mining_Runfunc,
	}
	return cc
}

func Command_Exit() *cobra.Command {
	cc := &cobra.Command{
		Use:   "exit",
		Short: "Exit CESS mining platform",
		Run:   Command_Exit_Runfunc,
	}
	return cc
}

func Command_Increase() *cobra.Command {
	cc := &cobra.Command{
		Use:   "increase",
		Short: "Increase the deposit of miners",
		Run:   Command_Increase_Runfunc,
	}
	return cc
}

func Command_Withdraw() *cobra.Command {
	cc := &cobra.Command{
		Use:   "withdraw",
		Short: "Redemption deposit",
		Run:   Command_Withdraw_Runfunc,
	}
	return cc
}

func Command_Obtain() *cobra.Command {
	cc := &cobra.Command{
		Use:   "obtain",
		Short: "Get cess test coin",
		Run:   Command_Obtain_Runfunc,
	}
	return cc
}

func Command_Version_Runfunc(cmd *cobra.Command, args []string) {
	fmt.Println(configs.Version)
}

func Command_Default_Runfunc(cmd *cobra.Command, args []string) {
	tools.WriteStringtoFile(configs.ConfigFile_Templete, configs.DefaultConfigurationFileName)
}

func Command_Register_Runfunc(cmd *cobra.Command, args []string) {
	//TODO
	refreshProfile(cmd)
}

func Command_State_Runfunc(cmd *cobra.Command, args []string) {
	//TODO
	refreshProfile(cmd)
}

func Command_Mining_Runfunc(cmd *cobra.Command, args []string) {
	//TODO
	refreshProfile(cmd)
}

func Command_Exit_Runfunc(cmd *cobra.Command, args []string) {
	//TODO
	refreshProfile(cmd)
}
func Command_Increase_Runfunc(cmd *cobra.Command, args []string) {
	//TODO
	refreshProfile(cmd)
}

func Command_Withdraw_Runfunc(cmd *cobra.Command, args []string) {
	//TODO
	refreshProfile(cmd)
}
func Command_Obtain_Runfunc(cmd *cobra.Command, args []string) {
	//TODO
	refreshProfile(cmd)
}

//
func refreshProfile(cmd *cobra.Command) {
	configpath1, _ := cmd.Flags().GetString("config")
	configpath2, _ := cmd.Flags().GetString("c")
	if configpath1 != "" {
		configs.ConfFilePath = configpath1
	} else {
		configs.ConfFilePath = configpath2
	}
	parseProfile()
}

func parseProfile() {
	var (
		err          error
		confFilePath string
	)
	if configs.ConfFilePath == "" {
		confFilePath = "./conf.toml"
	} else {
		confFilePath = configs.ConfFilePath
	}

	f, err := os.Stat(confFilePath)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file does not exist\n", 41, confFilePath)
		os.Exit(configs.Exit_ConfFileNotExist)
	}
	if f.IsDir() {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' is not a file\n", 41, confFilePath)
		os.Exit(configs.Exit_ConfFileNotExist)
	}

	viper.SetConfigFile(confFilePath)
	viper.SetConfigType("toml")

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file type error\n", 41, confFilePath)
		os.Exit(configs.Exit_ConfFileTypeError)
	}
	err = viper.Unmarshal(configs.Confile)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file format error\n", 41, confFilePath)
		os.Exit(configs.Exit_ConfFileFormatError)
	}
	fmt.Println(configs.Confile)
}
