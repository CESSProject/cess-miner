package initlz

import (
	"fmt"
	"os"
	"runtime"
)

// system init
func SystemInit() {
	if !runOnLinuxSystem() {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please execute on Linux system\n", 41)
		os.Exit(1)
	}
	setAllCores()
}

func runOnLinuxSystem() bool {
	return runtime.GOOS == "linux"
}

func runWithRootPrivileges() bool {
	return os.Geteuid() == 0
}

func setAllCores() {
	runtime.GOMAXPROCS(runtime.NumCPU() - 1)
}
