package console

import (
	"fmt"
	"log"
	"os"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/confile"

	"github.com/CESSProject/cess-bucket/pkg/utils"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/spf13/cobra"
)

// Update the miner's access address
func Command_UpdateIncome_Runfunc(cmd *cobra.Command, args []string) {
	if len(os.Args) >= 3 {
		pubkey, err := utils.DecodePublicKeyOfCessAccount(os.Args[2])
		if err != nil {
			log.Printf("\x1b[%dm[ok]\x1b[0m account error\n", 42)
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

		txhash, err := chn.UpdateIncome(types.NewAccountID(pubkey))
		if err != nil {
			if err.Error() == chain.ERR_Empty {
				log.Println("[err] Please check if the wallet is registered and its balance.")
			} else {
				if txhash != "" {
					msg := configs.HELP_Head + fmt.Sprintf(" %v\n", txhash)
					msg += fmt.Sprintf("%v\n", configs.HELP_UpdataBeneficiary)
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

	log.Printf("\x1b[%dm[err]\x1b[0m You should enter something like 'bucket update_income <account>'\n", 41)
	os.Exit(1)
}
