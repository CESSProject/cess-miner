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

package node

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/utils"
)

func (n *Node) task_common(ch chan bool) {
	defer func() {
		err := recover()
		if err != nil {
			n.Logs.Pnc(utils.RecoverError(err))
		}
		ch <- true
	}()
	n.Logs.Out("info", fmt.Errorf(">>>>> Start task_common <<<<<"))

	timer_ClearMem := time.NewTimer(configs.ClearMemInterval)
	defer timer_ClearMem.Stop()
	timer_ReplaceFile := time.NewTimer(configs.ReplaceFileInterval)
	defer timer_ReplaceFile.Stop()
	timer_GC := time.NewTimer(time.Minute)
	defer timer_GC.Stop()

	for {
		select {
		case <-timer_ClearMem.C:
			utils.ClearMemBuf()
			_, err := n.Chn.GetMinerInfo(n.Chn.GetPublicKey())
			if err != nil {
				if err.Error() == chain.ERR_Empty {
					os.Exit(1)
				}
			}
		case <-timer_ReplaceFile.C:
			invalidFiles, err := n.Chn.GetInvalidFiles()
			if err != nil {
				if err.Error() != chain.ERR_Empty {
					n.Logs.Repl("err", err)
				}
			}

			if len(invalidFiles) == 0 {
				continue
			}

			n.Logs.Repl("info", fmt.Errorf("Prepare to remove invalid files [%v]", len(invalidFiles)))
			for x := 0; x < len(invalidFiles); x++ {
				n.Logs.Repl("info", fmt.Errorf("   %v: %s", x, string(invalidFiles[x][:])))
			}

			for i := 0; i < len(invalidFiles); i++ {
				fileid := string(invalidFiles[i][:])
				filefullpath := ""
				filetagfullpath := ""
				if fileid[:4] != "cess" {
					filefullpath = filepath.Join(n.FillerDir, fileid)
					filetagfullpath = filepath.Join(n.FillerDir, fileid+".tag")
				} else {
					filefullpath = filepath.Join(n.FileDir, fileid)
					filetagfullpath = filepath.Join(n.FileDir, fileid+".tag")
				}
				txhash, err := n.Chn.ClearInvalidFiles(invalidFiles[i])
				if txhash != "" {
					n.Logs.Repl("info", fmt.Errorf("[%v] Cleared %v", string(invalidFiles[i][:]), txhash))
				} else {
					n.Logs.Repl("err", fmt.Errorf("[%v] Cleared err: %v", string(invalidFiles[i][:]), err))
				}
				os.Remove(filefullpath)
				os.Remove(filetagfullpath)
			}
		case <-timer_GC.C:
			runtime.GC()
		}
	}
}
