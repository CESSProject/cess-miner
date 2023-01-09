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
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/spf13/cobra"
)

// updateCertCmd is used to Update the certificate of the miner's sgx service
//
// Usage:
//
//	bucket update_cert
func updateCertCmd(cmd *cobra.Command, args []string) {
	var (
		err            error
		configFilePath string
		n              = node.New()
	)

	// config file
	configpath1, _ := cmd.Flags().GetString("config")
	configpath2, _ := cmd.Flags().GetString("c")
	if configpath1 != "" {
		configFilePath = configpath1
	} else {
		configFilePath = configpath2
	}

	n.Cfile = confile.NewConfigfile()
	if err := n.Cfile.Parse(configFilePath); err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// chain client
	n.Chn, err = chain.NewChainClient(
		n.Cfile.GetRpcAddr(),
		n.Cfile.GetCtrlPrk(),
		n.Cfile.GetIncomeAcc(),
		configs.TimeOut_WaitBlock,
	)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// Start Callback
	n.StartCallback()

	//Query your own information on the chain
	_, err = n.Chn.GetMinerInfo(n.Chn.GetPublicKey())
	if err != nil {
		if err.Error() == chain.ERR_Empty {
			log.Printf("[err] Not found: %v\n", err)
			os.Exit(1)
		}
		log.Printf("[err] Query error: %v\n", err)
		os.Exit(1)
	}

	var report node.Report
	err = node.GetReportReq(configs.URL_GetReport_Callback, n.Cfile.GetServiceAddr(), n.Cfile.GetSgxPortNum(), configs.URL_GetReport)
	if err != nil {
		log.Println("Please start the sgx service first")
		os.Exit(1)
	}

	timeout := time.NewTimer(configs.TimeOut_WaitReport)
	defer timeout.Stop()
	select {
	case <-timeout.C:
		log.Println("Timed out waiting for sgx report")
		os.Exit(1)
	case report = <-node.Ch_Report:
	}

	if report.Cert == "" || report.Ias_sig == "" || report.Quote == "" || report.Quote_sig == "" {
		log.Println("Invalid sgx report")
		os.Exit(1)
	}

	sig, err := hex.DecodeString(report.Quote_sig)
	if err != nil {
		log.Println("Invalid sgx report quote_sig")
		os.Exit(1)
	}

	//increase deposit
	txhash, err := n.Chn.UpdateCert(
		types.NewBytes([]byte(report.Cert)),
		types.NewBytes([]byte(report.Ias_sig)),
		types.NewBytes([]byte(report.Quote)),
		types.NewBytes(sig))
	if err != nil {
		if err.Error() == chain.ERR_Empty {
			log.Println("[err] Please check if the wallet is registered and its balance.")
		} else {
			if txhash != "" {
				msg := configs.HELP_Head + fmt.Sprintf(" %v\n", txhash)
				msg += fmt.Sprintf("%v\n", configs.HELP_MinerUpdateIasCert)
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
