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
	setCores()
}

func runOnLinuxSystem() bool {
	return runtime.GOOS == "linux"
}

func runWithRootPrivileges() bool {
	return os.Geteuid() == 0
}

// Allocate 2/3 cores to the program
func setCores() {
	num := runtime.NumCPU()
	num = num * 2 / 3
	if num <= 1 {
		runtime.GOMAXPROCS(1)
	} else {
		runtime.GOMAXPROCS(num)
	}
}
