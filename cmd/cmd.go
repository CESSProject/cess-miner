package cmd

import (
	"fmt"
	"os"
	"storage-mining/configs"
	"storage-mining/internal/chain"
	"storage-mining/internal/logger"
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
	os.Exit(0)
}

func Command_Default_Runfunc(cmd *cobra.Command, args []string) {
	tools.WriteStringtoFile(configs.ConfigFile_Templete, configs.DefaultConfigurationFileName)
	os.Exit(0)
}

func Command_Register_Runfunc(cmd *cobra.Command, args []string) {
	refreshProfile(cmd)
	peerid, err := queryMinerId()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		logger.ErrLogger.Sugar().Errorf("%v", err)
		os.Exit(-1)
	}
	if peerid > 0 {
		fmt.Printf("\x1b[%dm[ok]\x1b[0m Already registered [C%v]\n", 42, peerid)
		logger.InfoLogger.Sugar().Infof("Already registered [C%v]", peerid)
		os.Exit(0)
	} else {
		register()
	}
}

func Command_State_Runfunc(cmd *cobra.Command, args []string) {
	refreshProfile(cmd)
	minerInfo, err := chain.GetMinerDetailInfo(
		configs.Confile.MinerData.TransactionPrK,
		configs.ChainModule_Sminer,
		configs.ChainModule_Sminer_MinerItems,
		configs.ChainModule_Sminer_MinerDetails,
	)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		logger.ErrLogger.Sugar().Errorf("%v", err)
		os.Exit(-1)
	}
	fmt.Printf("MinerId:C%v\nState:%v\nStorageSpace:%vGB\nUsedSpace:%vGB\nPledgeTokens:%vCESS\nAccountAddr:%v\n",
		minerInfo.Peerid, string(minerInfo.State), minerInfo.Power, minerInfo.Space, minerInfo.Collaterals1, minerInfo.Address)
	os.Exit(0)
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

//
func queryMinerId() (uint64, error) {
	mData, err := chain.GetMinerInfo1(
		configs.Confile.MinerData.TransactionPrK,
		configs.ChainModule_Sminer,
		configs.ChainModule_Sminer_MinerItems,
	)
	return uint64(mData.Peerid), err
}

//
func register() {
	var pledgeTokens uint64
	pledgeTokens = 2000 * (configs.Confile.MinerData.StorageSpace / (1024 * 1024 * 1024 * 1024))
	if configs.Confile.MinerData.StorageSpace%(1024*1024*1024*1024) != 0 {
		pledgeTokens += 2000
	}

	res := tools.Base58Encoding(configs.Confile.MinerData.ServiceAddr + fmt.Sprintf("%d", configs.Confile.MinerData.ServicePort))

	logger.InfoLogger.Sugar().Infof("Start registration......\n    CessAddr:%v\n    PledgeTokens:%v\n    ServiceAddr:%v\n    TransactionPrK:%v\n    RevenuePuK :%v",
		configs.Confile.CessChain.ChainAddr, pledgeTokens, configs.Confile.MinerData.ServiceAddr, configs.Confile.MinerData.TransactionPrK, configs.Confile.MinerData.RevenuePuK)

	ok, err := chain.RegisterToChain(
		configs.Confile.MinerData.TransactionPrK,
		configs.Confile.MinerData.RevenuePuK,
		res,
		configs.ChainTx_Sminer_Register,
		pledgeTokens,
	)
	if !ok || err != nil {
		logger.InfoLogger.Sugar().Infof("Registration failed......,err:%v", err)
		logger.ErrLogger.Sugar().Errorf("%v", err)
		fmt.Printf("\x1b[%dm[err]\x1b[0m Registration failed, Please try again later. [%v]\n", 41, err)
		os.Exit(configs.Exit_RegisterToChain)
	}

	id, err := queryMinerId()
	if err == nil {
		logger.InfoLogger.Sugar().Infof("Your peerId is [C%v]", id)
		fmt.Printf("\x1b[%dm[ok]\x1b[0m registration success, your id is C%v\n", 42, id)
		os.Exit(0)
	}
	fmt.Println("success")
	os.Exit(0)
}
