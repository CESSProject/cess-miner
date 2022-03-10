package initlz

import (
	"fmt"
	"os"
	"storage-mining/configs"
	"storage-mining/internal/proof"
	"storage-mining/tools"
)

func SystemInit() {
	sysInit()
	//logger.LoggerInit()
	//chain.Chain_Init()
	proof.Proof_Init()
}

func sysInit() {
	if !tools.RunOnLinuxSystem() {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please execute on Linux system\n", 41)
		os.Exit(configs.Exit_RunningSystemError)
	}
	if !tools.RunWithRootPrivileges() {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please execute with root privileges\n", 41)
		os.Exit(configs.Exit_ExecutionPermissionError)
	}
	tools.SetAllCores()
}
