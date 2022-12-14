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

package console

import (
	"log"
	"os"
	"path/filepath"

	"github.com/CESSProject/cess-bucket/internal/confile"
	"github.com/spf13/cobra"
)

// defaultCmd generates a configuration file template in the current path
//
// Usage:
//
//	bucket default
func defaultCmd(cmd *cobra.Command, args []string) {
	pwd, err := os.Getwd()
	if err != nil {
		log.Printf("[err] %v\n", err)
		os.Exit(1)
	}
	path := filepath.Join(pwd, confile.ConfigurationFileTemplateName)
	_, err = os.Stat(path)
	if err == nil {
		log.Printf("[err] <%v> already exists", path)
		os.Exit(1)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		log.Printf("[err] %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	_, err = f.WriteString(confile.ConfigurationFileTemplete)
	if err != nil {
		log.Printf("[err] %v\n", err)
		os.Exit(1)
	}
	err = f.Sync()
	if err != nil {
		log.Printf("[err] %v\n", err)
		os.Exit(1)
	}
	log.Printf("[ok] %v\n", path)
	os.Exit(0)
}
