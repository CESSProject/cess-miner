package console

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/internal/chain"
	. "github.com/CESSProject/cess-bucket/internal/logger"
	"github.com/CESSProject/cess-bucket/tools"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/spf13/cobra"
)

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
	txhash, err := chain.Increase(api, configs.C.SignatureAcc, chain.ChainTx_Sminer_Increase, tokens)
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
	txhash, err := chain.Withdraw(api, configs.C.SignatureAcc, chain.ChainTx_Sminer_Withdraw)
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
		txhash, err := chain.UpdateIncome(configs.C.SignatureAcc, types.NewAccountID(pubkey))
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
