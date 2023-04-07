package console

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/initlz"
	"github.com/CESSProject/cess-bucket/internal/chain"
	"github.com/CESSProject/cess-bucket/internal/logger"
	. "github.com/CESSProject/cess-bucket/internal/logger"
	"github.com/CESSProject/cess-bucket/internal/node"
	"github.com/CESSProject/cess-bucket/internal/pattern"
	"github.com/CESSProject/cess-bucket/tools"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
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
		Command_UpdateAddress(),
		Command_UpdateIncome(),
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
		Short:                 "Generate configuration file template in current directory",
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

	api, err := chain.NewRpcClient(configs.C.RpcAddr)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Connection error: %v\n", 41, err)
		os.Exit(1)
	}

	//Query your own information on the chain
	_, err = chain.GetMinerInfo(api)
	if err != nil {
		if err.Error() == chain.ERR_Empty {
			err = register(api)
			if err != nil {
				fmt.Printf("\x1b[%dm[err]\x1b[0m Register failed: %v\n", 41, err)
				os.Exit(1)
			}
			os.Exit(0)
		}
		fmt.Printf("\x1b[%dm[err]\x1b[0m Query error: %v\n", 41, err)
		os.Exit(1)
	}

	fmt.Printf("\x1b[%dm[ok]\x1b[0m You are registered\n", 42)
	os.Exit(0)
}

func register(api *gsrpc.SubstrateAPI) error {
	//Calculate the deposit based on the size of the storage space
	pledgeTokens := 2000 * (configs.C.StorageSpace / 1024)
	if configs.C.StorageSpace%1024 != 0 {
		pledgeTokens += 2000
	}

	_, err := os.Stat(configs.BaseDir)
	if err == nil {
		bkpname := configs.BaseDir + "_" + fmt.Sprintf("%v", time.Now().Unix()) + "_bkp"
		os.Rename(configs.BaseDir, bkpname)
	}

	//Registration information on the chain
	txhash, err := chain.Register(
		api,
		configs.C.IncomeAcc,
		configs.C.ServiceIP,
		uint16(configs.C.ServicePort),
		pledgeTokens,
	)
	if err != nil {
		if err.Error() == chain.ERR_Empty {
			log.Println("[err] Please check your wallet balance.")
		} else {
			if txhash != "" {
				msg := configs.HELP_common + fmt.Sprintf(" %v\n", txhash)
				msg += configs.HELP_register
				log.Printf("[pending] %v\n", msg)
			} else {
				log.Printf("[err] %v.\n", err)
			}
		}
		return err
	}

	fmt.Printf("\x1b[%dm[ok]\x1b[0m Registration success\n", 42)

	//Create the storage data directory
	err = os.MkdirAll(configs.BaseDir, os.ModeDir)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		return err
	}

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

	log.Println(configs.LogfileDir)
	log.Println(configs.SpaceDir)
	log.Println(configs.FilesDir)

	//Initialize the logger
	logger.LoggerInit()

	//Record registration information to the log
	Out.Sugar().Infof("Registration message:")
	Out.Sugar().Infof("ChainAddr:%v", configs.C.RpcAddr)
	Out.Sugar().Infof("Register transaction hash:%v", txhash)
	return nil
Err:
	log.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
	return err
}

// Query your own details on-chain
func Command_State_Runfunc(cmd *cobra.Command, args []string) {
	//Parse command arguments and  configuration file
	parseFlags(cmd)

	api, err := chain.NewRpcClient(configs.C.RpcAddr)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Connection error: %v\n", 41, err)
		os.Exit(1)
	}
	//Query your own information on the chain
	mData, err := chain.GetMinerInfo(api)
	if err != nil {
		if err.Error() == chain.ERR_Empty {
			log.Printf("[err] Not found: %v\n", err)
			os.Exit(1)
		}
		log.Printf("[err] Query error: %v\n", err)
		os.Exit(1)
	}
	mData.Collaterals.Div(new(big.Int).SetBytes(mData.Collaterals.Bytes()), big.NewInt(1000000000000))
	addr := fmt.Sprintf("%d.%d.%d.%d:%d", mData.Ip.Value[0], mData.Ip.Value[1], mData.Ip.Value[2], mData.Ip.Value[3], mData.Ip.Port)
	var power, space float32
	var power_unit, space_unit string
	count := 0
	for mData.Power.BitLen() > int(16) {
		mData.Power.Div(new(big.Int).SetBytes(mData.Power.Bytes()), big.NewInt(1024))
		count++
	}
	if mData.Power.Int64() > 1024 {
		power = float32(mData.Power.Int64()) / float32(1024)
		count++
	} else {
		power = float32(mData.Power.Int64())
	}
	switch count {
	case 0:
		power_unit = "Byte"
	case 1:
		power_unit = "KiB"
	case 2:
		power_unit = "MiB"
	case 3:
		power_unit = "GiB"
	case 4:
		power_unit = "TiB"
	case 5:
		power_unit = "PiB"
	case 6:
		power_unit = "EiB"
	case 7:
		power_unit = "ZiB"
	case 8:
		power_unit = "YiB"
	case 9:
		power_unit = "NiB"
	case 10:
		power_unit = "DiB"
	default:
		power_unit = fmt.Sprintf("DiB(%v)", count-10)
	}
	count = 0
	for mData.Space.BitLen() > int(16) {
		mData.Space.Div(new(big.Int).SetBytes(mData.Space.Bytes()), big.NewInt(1024))
		count++
	}
	if mData.Space.Int64() > 1024 {
		space = float32(mData.Space.Int64()) / float32(1024)
		count++
	} else {
		space = float32(mData.Space.Int64())
	}

	switch count {
	case 0:
		space_unit = "Byte"
	case 1:
		space_unit = "KiB"
	case 2:
		space_unit = "MiB"
	case 3:
		space_unit = "GiB"
	case 4:
		space_unit = "TiB"
	case 5:
		space_unit = "PiB"
	case 6:
		space_unit = "EiB"
	case 7:
		space_unit = "ZiB"
	case 8:
		space_unit = "YiB"
	case 9:
		space_unit = "NiB"
	case 10:
		space_unit = "DiB"
	default:
		power_unit = fmt.Sprintf("DiB(%v)", count-10)
	}

	//print your own details
	fmt.Printf("MinerId: C%v\nState: %v\nStorageSpace: %.2f %v\nUsedSpace: %.2f %v\nPledgeTokens: %v TCESS\nServiceAddr: %v\n",
		mData.PeerId, string(mData.State), power, power_unit, space, space_unit, mData.Collaterals, string(addr))
	os.Exit(0)
}

// Start mining
func Command_Run_Runfunc(cmd *cobra.Command, args []string) {
	//Parse command arguments and  configuration file
	parseFlags(cmd)

	//global initialization
	initlz.SystemInit()

	flag, err := register_if()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Err: %v\n", 41, err)
		os.Exit(1)
	}

	if !flag {
		err = register(nil)
		if err != nil {
			log.Printf("[err] Registration failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		//Create log directory
		configs.LogfileDir = filepath.Join(configs.BaseDir, configs.LogfileDir)
		if err = tools.CreatDirIfNotExist(configs.LogfileDir); err != nil {
			fmt.Printf("\x1b[%dm[err]\x1b[0m Err: %v\n", 41, err)
			os.Exit(1)
		}
		//Create space directory
		configs.SpaceDir = filepath.Join(configs.BaseDir, configs.SpaceDir)
		if err = tools.CreatDirIfNotExist(configs.SpaceDir); err != nil {
			fmt.Printf("\x1b[%dm[err]\x1b[0m Err: %v\n", 41, err)
			os.Exit(1)
		}
		//Create file directory
		configs.FilesDir = filepath.Join(configs.BaseDir, configs.FilesDir)
		if err = tools.CreatDirIfNotExist(configs.FilesDir); err != nil {
			fmt.Printf("\x1b[%dm[err]\x1b[0m Err: %v\n", 41, err)
			os.Exit(1)
		}
		log.Println(configs.LogfileDir)
		log.Println(configs.SpaceDir)
		log.Println(configs.FilesDir)
		//Initialize the logger
		logger.LoggerInit()
	}

	// start-up
	n := node.New()
	n.Run()
}

// Exit mining
func Command_Exit_Runfunc(cmd *cobra.Command, args []string) {
	//Parse command arguments and  configuration file
	parseFlags(cmd)

	api, err := chain.NewRpcClient(configs.C.RpcAddr)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Connection error: %v\n", 41, err)
		os.Exit(1)
	}

	//Query your own information on the chain
	_, err = chain.GetMinerInfo(api)
	if err != nil {
		if err.Error() == chain.ERR_Empty {
			log.Printf("[err] Unregistered miner\n")
			os.Exit(1)
		}
		log.Printf("[err] Query error: %v\n", err)
		os.Exit(1)
	}

	// Exit the mining function
	txhash, err := chain.ExitMining(api, configs.C.SignatureAcc, chain.TX_SMINER_EXIT)
	if txhash != "" {
		chain.ClearFiller(api, configs.C.SignatureAcc)
		fmt.Println("success")
		os.Exit(0)
	}
	fmt.Printf("\x1b[%dm[err]\x1b[0m Failed to exit: %v\n", 41, err)
	os.Exit(1)
}

// Increase deposit
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

	api, err := chain.NewRpcClient(configs.C.RpcAddr)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Connection error: %v\n", 41, err)
		os.Exit(1)
	}

	//Query your own information on the chain
	_, err = chain.GetMinerInfo(api)
	if err != nil {
		if err.Error() == chain.ERR_Empty {
			log.Printf("[err] Unregistered miner\n")
			os.Exit(1)
		}
		log.Printf("[err] Query error: %v\n", err)
		os.Exit(1)
	}

	//Convert the deposit amount into TCESS units
	tokens, ok := new(big.Int).SetString(os.Args[2]+configs.TokenAccuracy, 10)
	if !ok {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please enter the correct deposit amount (positive integer).\n", 41)
		os.Exit(1)
	}

	//increase deposit
	txhash, err := chain.Increase(api, configs.C.SignatureAcc, chain.TX_SMINER_PLEDGETOKEN, tokens)
	if txhash == "" {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Failed to increase: %v\n", 41, err)
		os.Exit(1)
	}
	fmt.Println("success")
	os.Exit(0)
}

// Withdraw the deposit
func Command_Withdraw_Runfunc(cmd *cobra.Command, args []string) {
	//Parse command arguments and  configuration file
	parseFlags(cmd)

	api, err := chain.NewRpcClient(configs.C.RpcAddr)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Connection error: %v\n", 41, err)
		os.Exit(1)
	}

	//Query your own information on the chain
	_, err = chain.GetMinerInfo(api)
	if err != nil {
		if err.Error() == chain.ERR_Empty {
			log.Printf("[err] Unregistered miner\n")
			os.Exit(1)
		}
		log.Printf("[err] Query error: %v\n", err)
		os.Exit(1)
	}

	//Query the block height when the miner exits
	number, err := chain.GetBlockHeightExited(api)
	if err != nil {
		if err.Error() == chain.ERR_Empty {
			fmt.Printf("\x1b[%dm[err]\x1b[0m No exit, can't execute withdraw.\n", 41)
			os.Exit(1)
		}
		fmt.Printf("\x1b[%dm[err]\x1b[0m Failed to query exit block: %v\n", 41, err)
		os.Exit(1)
	}

	//Get the current block height
	lastnumber, err := chain.GetBlockHeight(api)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Failed to query the latest block: %v\n", 41, err)
		os.Exit(1)
	}

	if lastnumber < number {
		fmt.Printf("\x1b[%dm[err]\x1b[0m unexpected error\n", 41)
		os.Exit(1)
	}

	//Determine whether the cooling period is over
	if (lastnumber - number) < configs.ExitColling {
		wait := configs.ExitColling + number - lastnumber
		fmt.Printf("\x1b[%dm[err]\x1b[0m You are in a cooldown period, time remaining: %v blocks.\n", 41, wait)
		os.Exit(1)
	}

	// Withdraw deposit function
	txhash, err := chain.Withdraw(api, configs.C.SignatureAcc, chain.TX_SMINER_WITHDRAW)
	if txhash != "" {
		fmt.Println("success")
		os.Exit(0)
	}
	fmt.Printf("\x1b[%dm[err]\x1b[0m withdraw failed: %v\n", 41, err)
	os.Exit(1)
}

// Update the miner's access address
func Command_UpdateAddress_Runfunc(cmd *cobra.Command, args []string) {
	if len(os.Args) >= 3 {
		data := strings.Split(os.Args[2], ":")
		if len(data) != 2 {
			log.Printf("\x1b[%dm[err]\x1b[0m You should enter something like 'bucket address ip:port[domain_name]'\n", 41)
			os.Exit(1)
		}
		if !tools.IsIPv4(data[0]) {
			log.Printf("\x1b[%dm[ok]\x1b[0m address error\n", 42)
			os.Exit(1)
		}
		_, err := strconv.Atoi(data[1])
		if err != nil {
			log.Printf("\x1b[%dm[ok]\x1b[0m address error\n", 42)
			os.Exit(1)
		}

		//Parse command arguments and  configuration file
		parseFlags(cmd)

		txhash, err := chain.UpdateAddress(configs.C.SignatureAcc, data[0], data[1])
		if err != nil {
			if err.Error() == chain.ERR_Empty {
				log.Println("[err] Please check your wallet balance.")
			} else {
				if txhash != "" {
					msg := configs.HELP_common + fmt.Sprintf(" %v\n", txhash)
					msg += configs.HELP_UpdateAddress
					log.Printf("[pending] %v\n", msg)
				} else {
					log.Printf("[err] %v.\n", err)
				}
			}
			os.Exit(1)
		}
		log.Printf("\x1b[%dm[ok]\x1b[0m success\n", 42)
		os.Exit(0)
	}
	log.Printf("\x1b[%dm[err]\x1b[0m You should enter something like 'bucket address ip:port[domain_name]'\n", 41)
	os.Exit(1)
}

// Update the miner's access address
func Command_UpdateIncome_Runfunc(cmd *cobra.Command, args []string) {
	if len(os.Args) >= 3 {
		pubkey, err := tools.DecodeToCessPub(os.Args[2])
		if err != nil {
			log.Printf("\x1b[%dm[ok]\x1b[0m account error\n", 42)
			os.Exit(1)
		}
		//Parse command arguments and  configuration file
		parseFlags(cmd)
		acc, _ := types.NewAccountID(pubkey)
		txhash, err := chain.UpdateIncome(configs.C.SignatureAcc, *acc)
		if err != nil {
			if err.Error() == chain.ERR_Empty {
				log.Println("[err] Please check your wallet balance.")
			} else {
				if txhash != "" {
					msg := configs.HELP_common + fmt.Sprintf(" %v\n", txhash)
					msg += configs.HELP_UpdataBeneficiary
					log.Printf("[pending] %v\n", msg)
				} else {
					log.Printf("[err] %v.\n", err)
				}
			}
			os.Exit(1)
		}
		log.Printf("\x1b[%dm[ok]\x1b[0m success\n", 42)
		os.Exit(0)
	}
	log.Printf("\x1b[%dm[err]\x1b[0m You should enter something like 'bucket update_income account'\n", 41)
	os.Exit(1)
}

func register_if() (bool, error) {
	api, err := chain.GetRpcClient_Safe(configs.C.RpcAddr)
	defer chain.Free()
	if err != nil {
		return false, err
	}

	// sync block
	for {
		ok, err := chain.GetSyncStatus(api)
		if err != nil {
			return false, err
		}
		if !ok {
			break
		}
		log.Println("In sync block...")
		time.Sleep(configs.BlockInterval)
	}
	log.Println("Complete synchronization of primary network block data")

	//Query your own information on the chain
	_, err = chain.GetMinerInfo(api)
	if err != nil {
		if err.Error() == chain.ERR_Empty {
			return false, nil
		}
		return false, err
	}
	return true, nil
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
		configs.C.ServiceIP == "" ||
		configs.C.IncomeAcc == "" ||
		configs.C.SignatureAcc == "" {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The configuration file cannot have empty entries\n", 41)
		os.Exit(1)
	}

	if configs.C.ServicePort < 1024 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Prohibit the use of system reserved port: %v\n", 41, configs.C.ServicePort)
		os.Exit(1)
	}
	if configs.C.ServicePort > 65535 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The port number cannot exceed 65535\n", 41)
		os.Exit(1)
	}

	_, err = tools.GetMountPathInfo(configs.C.MountedPath)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m '%v' %v\n", 41, configs.C.MountedPath, err)
		os.Exit(1)
	}

	acc, err := chain.GetSelfPublicKey(configs.C.SignatureAcc)
	if err != nil {
		log.Printf("[err] %v\n", err)
		os.Exit(1)
	}

	addr, err := tools.EncodeToCESSAddr(acc)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	pattern.SetMinerAcc(acc)
	pattern.SetMinerSignAddr(configs.C.IncomeAcc)
	configs.BaseDir = filepath.Join(configs.C.MountedPath, addr, configs.BaseDir)
}
