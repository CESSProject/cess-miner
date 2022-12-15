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
	"os"
	"strconv"
	"strings"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/spf13/cobra"
)

// Update the miner's access address
func Command_UpdateAddress_Runfunc(cmd *cobra.Command, args []string) {
	if len(os.Args) >= 3 {
		data := strings.Split(os.Args[2], ":")
		if len(data) != 2 {
			log.Printf("\x1b[%dm[err]\x1b[0m You should enter something like 'bucket address ip:port[domain_name]'\n", 41)
			os.Exit(1)
		}
		if !utils.IsIPv4(data[0]) {
			log.Printf("\x1b[%dm[ok]\x1b[0m address error\n", 42)
			os.Exit(1)
		}
		_, err := strconv.Atoi(data[1])
		if err != nil {
			log.Printf("\x1b[%dm[ok]\x1b[0m address error\n", 42)
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

		txhash, err := chn.UpdateAddress(data[0], data[1])
		if err != nil {
			if err.Error() == chain.ERR_Empty {
				log.Println("[err] Please check your wallet balance.")
			} else {
				if txhash != "" {
					msg := configs.HELP_Head + fmt.Sprintf(" %v\n", txhash)
					msg += fmt.Sprintf("%v\n", configs.HELP_UpdateAddress)
					msg += configs.HELP_Tail
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
	log.Printf("\x1b[%dm[err]\x1b[0m You should enter something like 'bucket update_address ip:port'\n", 41)
	os.Exit(1)
}
