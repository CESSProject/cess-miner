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
	"log"
	"math/big"
	"os"
	"strconv"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/spf13/cobra"
)

// increaseCmd is used to increase the deposit of mining miner
//
// Usage:
//
//	bucket increase <amount>
func increaseCmd(cmd *cobra.Command, args []string) {
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

	// config file
	var configFilePath string
	configpath1, _ := cmd.Flags().GetString("config")
	configpath2, _ := cmd.Flags().GetString("c")
	if configpath1 != "" {
		configFilePath = configpath1
	} else {
		configFilePath = configpath2
	}

	confile := confile.NewConfigfile()
	if err := confile.Parse(configFilePath); err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// chain client
	chn, err := chain.NewChainClient(
		confile.GetRpcAddr(),
		confile.GetCtrlPrk(),
		confile.GetIncomeAcc(),
		configs.TimeOut_WaitBlock,
	)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	//Query your own information on the chain
	_, err = chn.GetMinerInfo(chn.GetPublicKey())
	if err != nil {
		if err.Error() == chain.ERR_Empty {
			log.Printf("[err] Not found: %v\n", err)
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
	txhash, err := chn.Increase(tokens)
	if err != nil {
		if err.Error() == chain.ERR_Empty {
			log.Println("[err] Please check if the wallet is registered and its balance.")
		} else {
			if txhash != "" {
				msg := configs.HELP_Head + fmt.Sprintf(" %v\n", txhash)
				msg += fmt.Sprintf("%v\n", configs.HELP_MinerIncrease)
				msg += configs.HELP_Tail
				log.Printf("[pending] %v\n", msg)
			} else {
				log.Printf("[err] %v.\n", err)
			}
		}
		os.Exit(1)
	}

	fmt.Printf("\x1b[%dm[ok]\x1b[0m success\n", 42)
	os.Exit(0)
}
