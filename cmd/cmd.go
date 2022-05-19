package cmd

import (
	"cess-bucket/configs"
	"cess-bucket/initlz"
	"cess-bucket/internal/chain"
	"cess-bucket/internal/encryption"
	"cess-bucket/internal/logger"
	. "cess-bucket/internal/logger"
	"cess-bucket/internal/proof"
	"cess-bucket/internal/pt"
	"cess-bucket/internal/rpc"
	"cess-bucket/tools"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strconv"

	"github.com/btcsuite/btcutil/base58"
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
		Command_Run(),
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
		Short:                 "Generate configuration file template",
		Run:                   Command_Default_Runfunc,
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
		Short:                 "Start mining",
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

// Storage miner registration information on the chain
func Command_Register_Runfunc(cmd *cobra.Command, args []string) {
	//Parse command arguments and  configuration file
	parseFlags(cmd)

	//Initialize: connect to chain
	chain.Chain_Init()

	//Query your own information on the chain
	mData, code, err := chain.GetMinerItems(configs.C.SignaturePrk)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}

	//Find your information and exit
	if code != configs.Code_404 || mData.Peerid != 0 {
		fmt.Printf("\x1b[%dm[ok]\x1b[0m The account is already registered.\n", 42)
		os.Exit(0)
	}

	//Check if the storage directory and exists
	_, err = os.Stat(configs.BaseDir)
	if err == nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m '%v' directory conflict\n", 41, configs.BaseDir)
		os.Exit(1)
	}

	//register
	register()
}

func register() {
	//Calculate the deposit based on the size of the storage space
	pledgeTokens := 2000 * (configs.C.StorageSpace / (1024 * 1024 * 1024 * 1024))
	if configs.C.StorageSpace%(1024*1024*1024*1024) != 0 {
		pledgeTokens += 2000
	}

	//Get the external IP of the machine
	eip, err := tools.GetExternalIp()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	//Determine whether the external network IP is consistent with the configuration file
	if eip != configs.C.ServiceAddr {
		fmt.Printf("\x1b[%dm[err]\x1b[0mYou can use \"curl ifconfig.co\" to view the external network ip address\n", 41)
		os.Exit(1)
	}

	//Encode IP address in base58
	ipAddr := base58.Encode([]byte(configs.C.ServiceAddr + ":" + fmt.Sprintf("%d", configs.C.ServicePort)))

	//Create the storage data directory
	err = os.MkdirAll(configs.BaseDir, os.ModeDir)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	//Generate RSA key pair
	encryption.GenKeypair()
	publicKeyfile := filepath.Join(configs.BaseDir, configs.PublicKeyfile)
	puk, err := ioutil.ReadFile(publicKeyfile)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	//Registration information on the chain
	txhash, code, err := chain.RegisterBucketToChain(
		configs.C.SignaturePrk,
		configs.C.IncomeAcc,
		ipAddr,
		pledgeTokens,
		puk,
	)
	if err != nil {
		if code != int(configs.Code_600) {
			fmt.Printf("\x1b[%dm[err]\x1b[0m Registration failed, Please try again later. [%v]\n", 41, err)
			os.Exit(1)
		}
	}
	//Query your own information on the chain
	mData, code, err := chain.GetMinerItems(configs.C.SignaturePrk)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please check the network and try again.\n", 41)
		os.Exit(1)
	}
	if code == configs.Code_404 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Registration failed, Please try again later.\n", 41)
		os.Exit(1)
	}

	fmt.Printf("\x1b[%dm[ok]\x1b[0m Registration success\n", 42)

	//Create log directory
	configs.LogfileDir = filepath.Join(configs.BaseDir, configs.LogfileDir)
	if err = tools.CreatDirIfNotExist(configs.LogfileDir); err != nil {
		goto Err
	}
	//Create space directory
	configs.SpaceDir = filepath.Join(configs.BaseDir, configs.SpaceDir)
	if err = tools.CreatDirIfNotExist(configs.SpaceDir); err != nil {
		goto Err
	}
	//Create file directory
	configs.FilesDir = filepath.Join(configs.BaseDir, configs.FilesDir)
	if err = tools.CreatDirIfNotExist(configs.FilesDir); err != nil {
		goto Err
	}

	configs.MinerId_I = uint64(mData.Peerid)
	configs.MinerId_S = fmt.Sprintf("%v", mData.Peerid)

	//Initialize the logger
	logger.LoggerInit()

	//Record registration information to the log
	Out.Sugar().Infof("Registration message:")
	Out.Sugar().Infof("ChainAddr:%v", configs.C.RpcAddr)
	Out.Sugar().Infof("StorageSpace:%v", configs.C.StorageSpace)
	Out.Sugar().Infof("MountedPath:%v", configs.C.MountedPath)
	Out.Sugar().Infof("ServiceAddr:%v", ipAddr)
	Out.Sugar().Infof("RevenueAcc:%v", configs.C.IncomeAcc)
	Out.Sugar().Infof("SignaturePrk:%v", configs.C.SignaturePrk)
	Out.Sugar().Infof("Register transaction hash:%v", txhash)
	os.Exit(0)
Err:
	fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
	os.Exit(1)
}

// Query your own details on-chain
func Command_State_Runfunc(cmd *cobra.Command, args []string) {
	//Parse command arguments and  configuration file
	parseFlags(cmd)

	//Initialize: connect to chain
	chain.Chain_Init()

	//Query your own information on the chain
	mData, code, err := chain.GetMinerItems(configs.C.SignaturePrk)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}

	//Exit program without registration
	if code == configs.Code_404 || mData.Peerid == 0 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Unregistered\n", 41)
		os.Exit(0)
	}

	//Query your own details on-chain
	minerInfo, err := chain.GetMinerDetailInfo(
		configs.C.SignaturePrk,
		chain.State_Sminer,
		chain.Sminer_MinerItems,
		chain.Sminer_MinerDetails,
	)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		Err.Sugar().Errorf("%v", err)
		os.Exit(1)
	}
	tokens := minerInfo.MinerInfo1.Collaterals.Div(minerInfo.MinerInfo1.Collaterals.Int, big.NewInt(1000000000000))
	addr := base58.Decode(string(minerInfo.MinerInfo1.ServiceAddr))

	//print your own details
	fmt.Printf("MinerId: C%v\nState: %v\nStorageSpace: %vMB\nUsedSpace: %vMB\nPledgeTokens: %vCESS\nServiceAddr: %v\n",
		minerInfo.MinerInfo1.Peerid, string(minerInfo.MinerInfo1.State), minerInfo.MinerInfo2.Power, minerInfo.MinerInfo2.Space, tokens, string(addr))
	os.Exit(0)
}

// Start mining
func Command_Run_Runfunc(cmd *cobra.Command, args []string) {
	//Parse command arguments and  configuration file
	parseFlags(cmd)

	//Initialize: connect to chain
	chain.Chain_Init()

	//Query your own information on the chain
	mData, code, err := chain.GetMinerItems(configs.C.SignaturePrk)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please check the network and try again. [%v]\n", 41, err)
		os.Exit(1)
	}

	//Exit program without registration
	if code == configs.Code_404 || mData.Peerid == 0 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Unregistered\n", 41)
		os.Exit(1)
	}

	//Determine whether the data directory exists, and exit if it does not exist
	f, err := os.Stat(configs.BaseDir)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m '%v' not found\n", 41, configs.BaseDir)
		os.Exit(1)
	}
	if !f.IsDir() {
		fmt.Printf("\x1b[%dm[err]\x1b[0m '%v' is not a directory\n", 41, configs.BaseDir)
		os.Exit(1)
	}

	//update data directory
	configs.LogfileDir = filepath.Join(configs.BaseDir, configs.LogfileDir)
	configs.SpaceDir = filepath.Join(configs.BaseDir, configs.SpaceDir)
	configs.FilesDir = filepath.Join(configs.BaseDir, configs.FilesDir)
	configs.MinerId_S = fmt.Sprintf("%v", mData.Peerid)
	configs.MinerId_I = uint64(mData.Peerid)

	//global initialization
	initlz.SystemInit()

	//Initialize the logger
	logger.LoggerInit()

	//Check if key file exists
	encryption.Check_Keypair()

	// start-up
	go proof.Proof_Main()
	rpc.Rpc_Main()
}

// Exit mining
func Command_Exit_Runfunc(cmd *cobra.Command, args []string) {
	//Parse command arguments and  configuration file
	parseFlags(cmd)

	//Initialize: connect to chain
	chain.Chain_Init()

	//Query your own information on the chain
	mData, code, err := chain.GetMinerItems(configs.C.SignaturePrk)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}
	if code == configs.Code_404 || mData.Peerid == 0 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Unregistered\n", 41)
		os.Exit(0)
	}

	// Exit the mining function
	ok, err := chain.ExitMining(configs.C.SignaturePrk, chain.ChainTx_Sminer_ExitMining)
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

//Increase deposit
func Command_Increase_Runfunc(cmd *cobra.Command, args []string) {
	//Too few command line arguments
	if len(os.Args) < 3 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please enter the increased deposit amount.\n", 41)
		os.Exit(1)
	}

	//Convert the deposit amount to an integer
	_, err := strconv.ParseUint(os.Args[2], 10, 64)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please enter the correct deposit amount (positive integer).\n", 41)
		os.Exit(1)
	}

	//Parse command arguments and  configuration file
	parseFlags(cmd)

	//Initialize: connect to chain
	chain.Chain_Init()

	//Query your own information on the chain
	mData, code, err := chain.GetMinerItems(configs.C.SignaturePrk)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}
	if code == configs.Code_404 || mData.Peerid == 0 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Unregistered\n", 41)
		os.Exit(0)
	}

	//Convert the deposit amount into TCESS units
	tokens, ok := new(big.Int).SetString(os.Args[2]+configs.TokenAccuracy, 10)
	if !ok {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please enter the correct deposit amount (positive integer).\n", 41)
		os.Exit(1)
	}

	//increase deposit
	ok, err = chain.Increase(configs.C.SignaturePrk, chain.ChainTx_Sminer_Increase, tokens)
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

// Withdraw the deposit
func Command_Withdraw_Runfunc(cmd *cobra.Command, args []string) {
	//Parse command arguments and  configuration file
	parseFlags(cmd)

	//Initialize: connect to chain
	chain.Chain_Init()

	//Query your own information on the chain
	mData, code, err := chain.GetMinerItems(configs.C.SignaturePrk)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}
	if code == configs.Code_404 || mData.Peerid == 0 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Unregistered\n", 41)
		os.Exit(0)
	}

	// Withdraw deposit function
	ok, err := chain.Withdraw(configs.C.SignaturePrk, chain.ChainTx_Sminer_Withdraw)
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

// Parse command arguments
func parseFlags(cmd *cobra.Command) {
	//Get custom configuration file
	configpath1, _ := cmd.Flags().GetString("config")
	configpath2, _ := cmd.Flags().GetString("c")
	if configpath1 != "" {
		configs.ConfFilePath = configpath1
	} else {
		configs.ConfFilePath = configpath2
	}
	//Parse the configuration file
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
	err = viper.Unmarshal(configs.C)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file format error\n", 41, confFilePath)
		os.Exit(1)
	}

	if configs.C.MountedPath == "" ||
		configs.C.ServiceAddr == "" ||
		configs.C.IncomeAcc == "" ||
		configs.C.SignaturePrk == "" {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The configuration file cannot have empty entries.\n", 41)
		os.Exit(1)
	}

	if configs.C.ServicePort < 1024 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Prohibit the use of system reserved port: %v.\n", 41, configs.C.ServicePort)
		os.Exit(1)
	}

	if configs.C.ServicePort > 65535 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Invalid port: %v.\n", 41, configs.C.ServicePort)
		os.Exit(1)
	}

	if configs.C.StorageSpace < 1000 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m You need to configure at least 1000GB of storage space.\n", 41)
		os.Exit(1)
	}

	_, err = pt.GetMountPathInfo(configs.C.MountedPath)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
	_, err = os.Stat(configs.C.MountedPath)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	hashs, err := tools.CalcHash([]byte(configs.C.SignaturePrk))
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
	configs.BaseDir = filepath.Join(configs.C.MountedPath, tools.GetStringWithoutNumbers(hashs), configs.BaseDir)
}
