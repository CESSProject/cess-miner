package console

import (
	"os"

	"github.com/spf13/cobra"
)

// Increase deposit
func Command_Increase_Runfunc(cmd *cobra.Command, args []string) {
	//Too few command line arguments
	// if len(os.Args) < 3 {
	// 	fmt.Printf("\x1b[%dm[err]\x1b[0m Please enter the increased deposit amount.\n", 41)
	// 	os.Exit(1)
	// }

	// //Convert the deposit amount to an integer
	// _, err := strconv.ParseUint(os.Args[2], 10, 64)
	// if err != nil {
	// 	fmt.Printf("\x1b[%dm[err]\x1b[0m Please enter the correct deposit amount (positive integer).\n", 41)
	// 	os.Exit(1)
	// }

	// //Parse command arguments and  configuration file
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

	// //Convert the deposit amount into TCESS units
	// tokens, ok := new(big.Int).SetString(os.Args[2]+configs.TokenAccuracy, 10)
	// if !ok {
	// 	fmt.Printf("\x1b[%dm[err]\x1b[0m Please enter the correct deposit amount (positive integer).\n", 41)
	// 	os.Exit(1)
	// }

	// //increase deposit
	// txhash, err := chain.Increase(api, configs.C.SignatureAcc, chain.TX_SMINER_PLEDGETOKEN, tokens)
	// if txhash == "" {
	// 	fmt.Printf("\x1b[%dm[err]\x1b[0m Failed to increase: %v\n", 41, err)
	// 	os.Exit(1)
	// }
	// fmt.Println("success")
	os.Exit(0)
}
