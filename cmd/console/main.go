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
	rootCmd.AddCommand(
		Command_Version(),
		Command_State(),
		Command_Run(),
		Command_Exit(),
		Command_Increase(),
		Command_Withdraw(),
	)
	rootCmd.PersistentFlags().StringP("config", "c", "conf.yaml", "Custom profile")
	rootCmd.PersistentFlags().StringP("rpc", "", "wss://testnet-rpc0.cess.cloud/ws/", "rpc endpoint")
	rootCmd.PersistentFlags().StringP("ws", "", "/", "workspace")
	rootCmd.PersistentFlags().StringP("ip", "", "0.0.0.0", "listening ip address")
	rootCmd.PersistentFlags().IntP("port", "p", 15000, "listening port")
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
		Use:                   "state",
		Short:                 "Query storage miner information",
		Run:                   Command_State_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_Run() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "run",
		Short:                 "Register and start mining",
		Run:                   runCmd,
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
