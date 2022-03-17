package cmd

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"storage-mining/configs"
	"storage-mining/initlz"
	"storage-mining/internal/chain"
	"storage-mining/internal/logger"
	"storage-mining/internal/proof"
	"storage-mining/rpc"
	"storage-mining/tools"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	Name        = "cess-bucket"
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
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println("[err] ", err)
	}
	path := filepath.Join(pwd, configs.DefaultConfigurationFileName)
	fmt.Println("[ok] ", path)
	os.Exit(0)
}

func Command_Register_Runfunc(cmd *cobra.Command, args []string) {
	refreshProfile(cmd)
	peerid, err := queryMinerId()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(-1)
	}
	if peerid > 0 {
		fmt.Printf("\x1b[%dm[ok]\x1b[0m Already registered [C%v]\n", 42, peerid)
		logger.InfoLogger.Sugar().Infof("Already registered [C%v]", peerid)
		os.Exit(0)
	} else {
		if configs.Confile.MinerData.MountedPath == "" ||
			configs.Confile.MinerData.ServiceAddr == "" ||
			configs.Confile.MinerData.ServicePort == 0 ||
			configs.Confile.MinerData.StorageSpace == 0 ||
			configs.Confile.MinerData.RevenuePuK == "" ||
			configs.Confile.MinerData.TransactionPrK == "" {
			fmt.Printf("\x1b[%dm[err]\x1b[0m The configuration file cannot have empty entries.\n", 41)
			os.Exit(-1)
		}
		register()
	}
}

func Command_State_Runfunc(cmd *cobra.Command, args []string) {
	refreshProfile(cmd)
	peerid, err := queryMinerId()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		logger.ErrLogger.Sugar().Errorf("%v", err)
		os.Exit(-1)
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
			logger.ErrLogger.Sugar().Errorf("%v", err)
			os.Exit(-1)
		}
		tokens := minerInfo.Collaterals1.Div(minerInfo.Collaterals1.Int, big.NewInt(1000000000000))
		addr := tools.Base58Decoding(string(minerInfo.ServiceAddr))
		fmt.Printf("MinerId:C%v\nState:%v\nStorageSpace:%vGB\nUsedSpace:%vGB\nPledgeTokens:%vCESS\nServiceAddr:%v\n",
			minerInfo.Peerid, string(minerInfo.State), minerInfo.Power, minerInfo.Space, tokens, addr)
	}
	os.Exit(0)
}

func Command_Mining_Runfunc(cmd *cobra.Command, args []string) {
	refreshProfile(cmd)
	peerid, err := queryMinerId()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		logger.ErrLogger.Sugar().Errorf("%v", err)
		os.Exit(-1)
	}
	if peerid == 0 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Unregistered\n", 42)
		os.Exit(0)
	} else {
		// init
		initlz.SystemInit()

		// start-up
		//chain.Chain_Main()
		proof.Proof_Main()

		// web service
		rpc.Rpc_Main()
	}
}

func Command_Exit_Runfunc(cmd *cobra.Command, args []string) {
	refreshProfile(cmd)
	peerid, err := queryMinerId()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		logger.ErrLogger.Sugar().Errorf("%v", err)
		os.Exit(-1)
	}
	if peerid == 0 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Unregistered\n", 42)
		os.Exit(0)
	} else {
		exitmining()
	}
}

func Command_Increase_Runfunc(cmd *cobra.Command, args []string) {
	refreshProfile(cmd)
	peerid, err := queryMinerId()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		logger.ErrLogger.Sugar().Errorf("%v", err)
		os.Exit(-1)
	}
	if peerid == 0 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Unregistered\n", 42)
		os.Exit(0)
	} else {
		increase()
	}
}

func Command_Withdraw_Runfunc(cmd *cobra.Command, args []string) {
	refreshProfile(cmd)
	peerid, err := queryMinerId()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		logger.ErrLogger.Sugar().Errorf("%v", err)
		os.Exit(-1)
	}
	if peerid == 0 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Unregistered\n", 42)
		os.Exit(0)
	} else {
		withdraw()
	}
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
	if err != nil {
		return 0, err
	}

	configs.MinerDataPath += fmt.Sprintf("%d", mData.Peerid)
	configs.MinerId_I = uint64(mData.Peerid)
	configs.MinerId_S = fmt.Sprintf("C%v", mData.Peerid)
	path := filepath.Join(configs.Confile.MinerData.MountedPath, configs.MinerDataPath)
	configs.MinerDataPath = path

	_, err = os.Stat(path)
	if err != nil {
		err = os.MkdirAll(path, os.ModeDir)
		if err != nil {
			fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
			os.Exit(configs.Exit_CreateFolder)
		}
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
	logger.LoggerInit()
	return uint64(mData.Peerid), nil
}

//
func register() {
	var pledgeTokens uint64
	pledgeTokens = 2000 * (configs.Confile.MinerData.StorageSpace / (1024 * 1024 * 1024 * 1024))
	if configs.Confile.MinerData.StorageSpace%(1024*1024*1024*1024) != 0 {
		pledgeTokens += 2000
	}

	res := tools.Base58Encoding(configs.Confile.MinerData.ServiceAddr + ":" + fmt.Sprintf("%d", configs.Confile.MinerData.ServicePort))

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

//
func increase() {
	if len(os.Args) < 3 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please enter the increased deposit amount.\n", 41)
		os.Exit(-1)
	}
	_, err := strconv.ParseUint(os.Args[2], 10, 64)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please enter the correct deposit amount (positive integer).\n", 41)
		os.Exit(-1)
	}

	tokens, ok := new(big.Int).SetString(os.Args[2]+configs.TokenAccuracy, 10)
	if !ok {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please enter the correct deposit amount (positive integer).\n", 41)
		os.Exit(-1)
	}

	ok, err = chain.Increase(configs.Confile.MinerData.TransactionPrK, configs.ChainTx_Sminer_Increase, tokens)
	if err != nil {
		logger.InfoLogger.Sugar().Infof("Increase failed......,err:%v", err)
		logger.ErrLogger.Sugar().Errorf("%v", err)
		fmt.Printf("\x1b[%dm[err]\x1b[0m Increase failed, Please try again later. [%v]\n", 41, err)
		os.Exit(-1)
	}
	if !ok {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Increase failed, Please try again later. [%v]\n", 41, err)
		os.Exit(-1)
	}
	fmt.Println("success")
	os.Exit(0)
}

//
func exitmining() {
	ok, err := chain.ExitMining(configs.Confile.MinerData.TransactionPrK, configs.ChainTx_Sminer_ExitMining)
	if err != nil {
		logger.InfoLogger.Sugar().Infof("Exit failed......,err:%v", err)
		logger.ErrLogger.Sugar().Errorf("%v", err)
		fmt.Printf("\x1b[%dm[err]\x1b[0m Exit failed, Please try again later. [%v]\n", 41, err)
		os.Exit(-1)
	}
	if !ok {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Exit failed, Please try again later. [%v]\n", 41, err)
		os.Exit(-1)
	}
	fmt.Println("success")
	os.Exit(0)
}

//
func withdraw() {
	ok, err := chain.Withdraw(configs.Confile.MinerData.TransactionPrK, configs.ChainTx_Sminer_Withdraw)
	if err != nil {
		logger.InfoLogger.Sugar().Infof("withdraw failed......,err:%v", err)
		logger.ErrLogger.Sugar().Errorf("%v", err)
		fmt.Printf("\x1b[%dm[err]\x1b[0m withdraw failed, Please try again later. [%v]\n", 41, err)
		os.Exit(-1)
	}
	if !ok {
		fmt.Printf("\x1b[%dm[err]\x1b[0m withdraw failed, Please try again later. [%v]\n", 41, err)
		os.Exit(-1)
	}
	fmt.Println("success")
	os.Exit(0)
}
