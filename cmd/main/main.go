package main

import (
	"storage-mining/initlz"
	"storage-mining/internal/chain"
	"storage-mining/internal/handler"
	"storage-mining/internal/proof"
)

// program entry
func main() {
	// init
	initlz.SystemInit()

	// start-up
	chain.Chain_Main()
	proof.Proof_Main()

	// web service
	handler.Handler_main()
}
