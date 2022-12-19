/*
   Copyright 2022 CESS (Cumulus Encrypted Storage System) authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

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
