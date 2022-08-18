package task

import (
	"cess-bucket/configs"
	"cess-bucket/internal/chain"
	. "cess-bucket/internal/logger"
	"cess-bucket/tools"
	"os"
	"path/filepath"
	"time"
)

//The task_RemoveInvalidFiles task automatically checks its own failed files and clears them.
//Delete from the local disk first, and then notify the chain to delete.
//It keeps running as a subtask.
func task_RemoveInvalidFiles(ch chan bool) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
		ch <- true
	}()
	Del.Info(">>>>> Start task_RemoveInvalidFiles <<<<<")
	for {
		invalidFiles, err := chain.GetInvalidFiles()
		if err != nil {
			if err.Error() != chain.ERR_Empty {
				Del.Sugar().Errorf("%v", err)
				invalidFiles, _ = chain.GetInvalidFiles()
			}
		}

		if len(invalidFiles) == 0 {
			time.Sleep(time.Minute * 10)
			continue
		}

		Del.Sugar().Infof("--> Prepare to remove invalid files [%v]", len(invalidFiles))
		for x := 0; x < len(invalidFiles); x++ {
			Del.Sugar().Infof("   %v: %s", x, string(invalidFiles[x]))
		}

		for i := 0; i < len(invalidFiles); i++ {
			fileid := string(invalidFiles[i])
			filefullpath := ""
			filetagfullpath := ""
			if fileid[:4] != "cess" {
				filefullpath = filepath.Join(configs.SpaceDir, fileid)
				filetagfullpath = filepath.Join(configs.SpaceDir, fileid+".tag")
			} else {
				filefullpath = filepath.Join(configs.FilesDir, fileid)
				filetagfullpath = filepath.Join(configs.FilesDir, fileid+".tag")
			}
			txhash, err := chain.ClearInvalidFiles(invalidFiles[i])
			if txhash != "" {
				Del.Sugar().Infof("[%v] Cleared %v", string(invalidFiles[i]), txhash)
			} else {
				Del.Sugar().Errorf("[err] [%v] Clear: %v", string(invalidFiles[i]), err)
			}
			os.Remove(filefullpath)
			os.Remove(filetagfullpath)
		}
		time.Sleep(time.Minute * 10)
	}
}
