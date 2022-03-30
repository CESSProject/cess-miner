package initlz

import (
	"cess-bucket/internal/proof"
	"cess-bucket/tools"
	"fmt"
	"os"
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
		os.Exit(1)
	}
	if !tools.RunWithRootPrivileges() {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please execute with root privileges\n", 41)
		os.Exit(1)
	}
	tools.SetAllCores()
}
