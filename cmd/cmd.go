package cmd

import (
	"cess-bucket/configs"
	"cess-bucket/initlz"
	"cess-bucket/internal/chain"
	. "cess-bucket/internal/logger"
	"cess-bucket/internal/proof"
	"cess-bucket/internal/rpc"
	"cess-bucket/tools"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	Name        = "cess-bucket"
	Description = "A mining program provided by cess platform for storage miners."
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
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
}

// init
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
		Use:                   "version",
		Short:                 "Print version information",
		Run:                   Command_Version_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_Default() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "default",
		Short:                 "Generate profile template",
		Run:                   Command_Default_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_Register() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "register",
		Short:                 "Register miner information to cess chain",
		Run:                   Command_Register_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_State() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "state",
		Short:                 "List miners' own information",
		Run:                   Command_State_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_Mining() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "mining",
		Short:                 "Start mining at CESS mining platform",
		Run:                   Command_Mining_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_Exit() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "exit",
		Short:                 "Exit CESS mining platform",
		Run:                   Command_Exit_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_Increase() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "increase <number of tokens>",
		Short:                 "Increase the deposit of miners",
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

func Command_Obtain() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "obtain <pubkey> <faucet address>",
		Short:                 "Get cess test coin",
		Run:                   Command_Obtain_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

// Print version number and exit
func Command_Version_Runfunc(cmd *cobra.Command, args []string) {
	fmt.Println(configs.Version)
	os.Exit(0)
}

// Generate configuration file template
func Command_Default_Runfunc(cmd *cobra.Command, args []string) {
	tools.WriteStringtoFile(configs.ConfigFile_Templete, "config_template.toml")
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
	path := filepath.Join(pwd, "config_template.toml")
	fmt.Println("[ok] ", path)
	os.Exit(0)
}

// Miner registration
func Command_Register_Runfunc(cmd *cobra.Command, args []string) {
	refreshProfile(cmd)
	peerid, err := queryMinerId(false)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
	if peerid > 0 {
		fmt.Printf("\x1b[%dm[ok]\x1b[0m Already registered [C%v]\n", 42, peerid)
		os.Exit(0)
	} else {
		if configs.Confile.MinerData.MountedPath == "" ||
			configs.Confile.MinerData.ServiceAddr == "" ||
			configs.Confile.MinerData.ServicePort == 0 ||
			configs.Confile.MinerData.StorageSpace == 0 ||
			configs.Confile.MinerData.RevenuePuK == "" ||
			configs.Confile.MinerData.TransactionPrK == "" {
			fmt.Printf("\x1b[%dm[err]\x1b[0m The configuration file cannot have empty entries.\n", 41)
			os.Exit(1)
		}
		register()
	}
}

// Check your status
func Command_State_Runfunc(cmd *cobra.Command, args []string) {
	refreshProfile(cmd)
	peerid, err := queryMinerId(false)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		Err.Sugar().Errorf("%v", err)
		os.Exit(1)
	}
	if peerid == 0 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Unregistered\n", 42)
		os.Exit(0)
	} else {
		minerInfo, err := chain.GetMinerDetailInfo(
			configs.Confile.MinerData.TransactionPrK,
			configs.ChainModule_Sminer,
			configs.ChainModule_Sminer_MinerItems,
			configs.ChainModule_Sminer_MinerDetails,
		)
		if err != nil {
			fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
			Err.Sugar().Errorf("%v", err)
			os.Exit(-1)
		}
		tokens := minerInfo.MinerInfo1.Collaterals.Div(minerInfo.MinerInfo1.Collaterals.Int, big.NewInt(1000000000000))
		addr := tools.Base58Decoding(string(minerInfo.MinerInfo1.ServiceAddr))
		fmt.Printf("MinerId:C%v\nState:%v\nStorageSpace:%vMB\nUsedSpace:%vMB\nPledgeTokens:%vCESS\nServiceAddr:%v\n",
			minerInfo.MinerInfo1.Peerid, string(minerInfo.MinerInfo1.State), minerInfo.MinerInfo2.Power, minerInfo.MinerInfo2.Space, tokens, addr)
	}
	os.Exit(0)
}

// Start mining
func Command_Mining_Runfunc(cmd *cobra.Command, args []string) {
	refreshProfile(cmd)
	peerid, err := queryMinerId(false)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		Err.Sugar().Errorf("%v", err)
		os.Exit(1)
	}
	if peerid == 0 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Unregistered\n", 41)
		os.Exit(0)
	} else {
		// init
		initlz.SystemInit()
		proof.Proof_Init()
		// start-up
		proof.Proof_Main()
		rpc.Rpc_Main()
	}
}

// Exit mining
func Command_Exit_Runfunc(cmd *cobra.Command, args []string) {
	refreshProfile(cmd)
	peerid, err := queryMinerId(false)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		Err.Sugar().Errorf("%v", err)
		os.Exit(-1)
	}
	if peerid == 0 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Unregistered\n", 42)
		os.Exit(0)
	} else {
		exitmining()
	}
}

//Increase deposit
func Command_Increase_Runfunc(cmd *cobra.Command, args []string) {
	if len(os.Args) < 3 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please enter the increased deposit amount.\n", 41)
		os.Exit(1)
	}
	_, err := strconv.ParseUint(os.Args[2], 10, 64)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please enter the correct deposit amount (positive integer).\n", 41)
		os.Exit(1)
	}
	refreshProfile(cmd)
	peerid, err := queryMinerId(false)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		Err.Sugar().Errorf("%v", err)
		os.Exit(-1)
	}
	if peerid == 0 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Unregistered\n", 42)
		os.Exit(0)
	} else {
		increase()
	}
}

// Withdraw the deposit
func Command_Withdraw_Runfunc(cmd *cobra.Command, args []string) {
	refreshProfile(cmd)
	peerid, err := queryMinerId(false)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		Err.Sugar().Errorf("%v", err)
		os.Exit(-1)
	}
	if peerid == 0 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Unregistered\n", 42)
		os.Exit(0)
	} else {
		withdraw()
	}
}

// obtain tCESS
func Command_Obtain_Runfunc(cmd *cobra.Command, args []string) {
	if len(os.Args) < 4 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please enter wallet address public key and faucet address.\n", 41)
		os.Exit(1)
	}
	err := chain.ObtainFromFaucet(os.Args[3], os.Args[2])
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err.Error())
		os.Exit(1)
	} else {
		fmt.Println("success")
		os.Exit(0)
	}
}

// Parse the configuration file
func refreshProfile(cmd *cobra.Command) {
	configpath1, _ := cmd.Flags().GetString("config")
	configpath2, _ := cmd.Flags().GetString("c")
	if configpath1 != "" {
		configs.ConfFilePath = configpath1
	} else {
		configs.ConfFilePath = configpath2
	}
	parseProfile()
	chain.Chain_Init()
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
		os.Exit(1)
	}
	if f.IsDir() {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' is not a file\n", 41, confFilePath)
		os.Exit(1)
	}

	viper.SetConfigFile(confFilePath)
	viper.SetConfigType("toml")

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file type error\n", 41, confFilePath)
		os.Exit(1)
	}
	err = viper.Unmarshal(configs.Confile)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file format error\n", 41, confFilePath)
		os.Exit(1)
	}

	_, err = os.Stat(configs.Confile.MinerData.MountedPath)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(-1)
	}
}

// Query miner id information
// Return miner id
func queryMinerId(flag bool) (uint64, error) {
	mData, err := chain.GetMinerInfo1(
		configs.Confile.MinerData.TransactionPrK,
		configs.ChainModule_Sminer,
		configs.ChainModule_Sminer_MinerItems,
	)
	if err != nil {
		return 0, err
	}
	if mData.Peerid == 0 {
		return 0, nil
	}

	if configs.MinerId_I == 0 {
		configs.MinerDataPath += fmt.Sprintf("%d", mData.Peerid)
		configs.MinerId_I = uint64(mData.Peerid)
		configs.MinerId_S = fmt.Sprintf("C%v", mData.Peerid)
		path := filepath.Join(configs.Confile.MinerData.MountedPath, configs.MinerDataPath)
		configs.MinerDataPath = path

		_, err = os.Stat(path)
		if err == nil {
			if flag {
				os.RemoveAll(path)
			}
		}
		err = os.MkdirAll(path, os.ModeDir)
		if err != nil {
			fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
			os.Exit(1)
		}

		saddr := tools.Base58Decoding(string(mData.ServiceAddr))
		tmp := strings.Split(saddr, ":")
		if len(tmp) == 2 {
			configs.MinerServiceAddr = tmp[0]
			configs.MinerServicePort, _ = strconv.Atoi(tmp[1])
		} else {
			configs.MinerServiceAddr = tmp[0]
			configs.MinerServicePort = 80
		}
		LoggerInit()
	}
	return uint64(mData.Peerid), nil
}

// Miner registration function
func register() {
	var pledgeTokens uint64
	pledgeTokens = 2000 * (configs.Confile.MinerData.StorageSpace / (1024 * 1024 * 1024 * 1024))
	if configs.Confile.MinerData.StorageSpace%(1024*1024*1024*1024) != 0 {
		pledgeTokens += 2000
	}

	res := tools.Base58Encoding(configs.Confile.MinerData.ServiceAddr + ":" + fmt.Sprintf("%d", configs.Confile.MinerData.ServicePort))

	ok, err := chain.RegisterToChain(
		configs.Confile.MinerData.TransactionPrK,
		configs.Confile.MinerData.RevenuePuK,
		res,
		configs.ChainTx_Sminer_Register,
		pledgeTokens,
	)
	if !ok || err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Registration failed, Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}

	id, err := queryMinerId(true)
	if err == nil && id > 0 {
		_, err = os.Stat(configs.MinerDataPath)

		Out.Sugar().Infof("Your peerId is [C%v]", id)
		fmt.Printf("\x1b[%dm[ok]\x1b[0m registration success, your id is C%v\n", 42, id)
	} else {
		fmt.Println("Network timed out, please try again!")
	}
	os.Exit(0)
}

// Increase deposit function
func increase() {
	tokens, ok := new(big.Int).SetString(os.Args[2]+configs.TokenAccuracy, 10)
	if !ok {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please enter the correct deposit amount (positive integer).\n", 41)
		os.Exit(1)
	}

	ok, err := chain.Increase(configs.Confile.MinerData.TransactionPrK, configs.ChainTx_Sminer_Increase, tokens)
	if err != nil {
		Out.Sugar().Infof("Increase failed......,err:%v", err)
		Err.Sugar().Errorf("%v", err)
		fmt.Printf("\x1b[%dm[err]\x1b[0m Increase failed, Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}
	if !ok {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Increase failed, Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}
	fmt.Println("success")
	os.Exit(0)
}

// Exit the mining function
func exitmining() {
	ok, err := chain.ExitMining(configs.Confile.MinerData.TransactionPrK, configs.ChainTx_Sminer_ExitMining)
	if err != nil {
		Out.Sugar().Infof("Exit failed......,err:%v", err)
		Err.Sugar().Errorf("%v", err)
		fmt.Printf("\x1b[%dm[err]\x1b[0m Exit failed, Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}
	if !ok {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Exit failed, Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}
	fmt.Println("success")
	os.Exit(0)
}

// Withdraw deposit function
func withdraw() {
	ok, err := chain.Withdraw(configs.Confile.MinerData.TransactionPrK, configs.ChainTx_Sminer_Withdraw)
	if err != nil {
		Out.Sugar().Infof("withdraw failed......,err:%v", err)
		Err.Sugar().Errorf("%v", err)
		fmt.Printf("\x1b[%dm[err]\x1b[0m withdraw failed, Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}
	if !ok {
		fmt.Printf("\x1b[%dm[err]\x1b[0m withdraw failed, Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}
	fmt.Println("success")
	os.Exit(0)
}
