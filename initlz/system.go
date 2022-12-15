package initlz

import (
	"log"
	"os"
	"runtime"
)

// system init
func init() {
	// Determine if the operating system is linux
	if runtime.GOOS != "linux" {
		log.Println("[err] Please run on linux system.")
		os.Exit(1)
	}
	// Allocate 2/3 cores to the program
	num := runtime.NumCPU()
	num = num * 2 / 3
	if num <= 1 {
		runtime.GOMAXPROCS(1)
	} else {
		runtime.GOMAXPROCS(num)
	}
}
