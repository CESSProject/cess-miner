package console

import (
	"os"

	"github.com/spf13/cobra"
)

// Withdraw the deposit
func Command_Withdraw_Runfunc(cmd *cobra.Command, args []string) {
	//Parse command arguments and  configuration file
	// parseFlags(cmd)

	// api, err := chain.NewRpcClient(configs.C.RpcAddr)
	// if err != nil {
	// 	fmt.Printf("\x1b[%dm[err]\x1b[0m Connection error: %v\n", 41, err)
	// 	os.Exit(1)
	// }

	// //Query your own information on the chain
	// _, err = chain.GetMinerInfo(api)
	// if err != nil {
	// 	if err.Error() == chain.ERR_Empty {
	// 		log.Printf("[err] Unregistered miner\n")
	// 		os.Exit(1)
	// 	}
	// 	log.Printf("[err] Query error: %v\n", err)
	// 	os.Exit(1)
	// }

	// //Query the block height when the miner exits
	// number, err := chain.GetBlockHeightExited(api)
	// if err != nil {
	// 	if err.Error() == chain.ERR_Empty {
	// 		fmt.Printf("\x1b[%dm[err]\x1b[0m No exit, can't execute withdraw.\n", 41)
	// 		os.Exit(1)
	// 	}
	// 	fmt.Printf("\x1b[%dm[err]\x1b[0m Failed to query exit block: %v\n", 41, err)
	// 	os.Exit(1)
	// }

	// //Get the current block height
	// lastnumber, err := chain.GetBlockHeight(api)
	// if err != nil {
	// 	fmt.Printf("\x1b[%dm[err]\x1b[0m Failed to query the latest block: %v\n", 41, err)
	// 	os.Exit(1)
	// }

	// if lastnumber < number {
	// 	fmt.Printf("\x1b[%dm[err]\x1b[0m unexpected error\n", 41)
	// 	os.Exit(1)
	// }

	// //Determine whether the cooling period is over
	// if (lastnumber - number) < configs.ExitColling {
	// 	wait := configs.ExitColling + number - lastnumber
	// 	fmt.Printf("\x1b[%dm[err]\x1b[0m You are in a cooldown period, time remaining: %v blocks.\n", 41, wait)
	// 	os.Exit(1)
	// }

	// // Withdraw deposit function
	// txhash, err := chain.Withdraw(api, configs.C.SignatureAcc, chain.TX_SMINER_WITHDRAW)
	// if txhash != "" {
	// 	fmt.Println("success")
	// 	os.Exit(0)
	// }
	// fmt.Printf("\x1b[%dm[err]\x1b[0m withdraw failed: %v\n", 41, err)
	os.Exit(1)
}
