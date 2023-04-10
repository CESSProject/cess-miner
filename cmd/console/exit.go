package console

import (
	"os"

	"github.com/spf13/cobra"
)

// Exit mining
func Command_Exit_Runfunc(cmd *cobra.Command, args []string) {
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

	// // Exit the mining function
	// txhash, err := chain.ExitMining(api, configs.C.SignatureAcc, chain.TX_SMINER_EXIT)
	// if txhash != "" {
	// 	chain.ClearFiller(api, configs.C.SignatureAcc)
	// 	fmt.Println("success")
	// 	os.Exit(0)
	// }
	// fmt.Printf("\x1b[%dm[err]\x1b[0m Failed to exit: %v\n", 41, err)
	os.Exit(1)
}
