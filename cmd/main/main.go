package main

import (
	"storage-mining/cmd"
	"storage-mining/initlz"
	"storage-mining/internal/chain"
	"storage-mining/internal/proof"
)

// program entry
func main() {

	cmd.Execute()

	// init
	initlz.SystemInit()

	// start-up
	chain.Chain_Main()
	proof.Proof_Main()

	select {}
	// web service
	//handler.Handler_main()
}
