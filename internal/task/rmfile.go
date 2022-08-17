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
	Out.Info(">>>>> Start task_RemoveInvalidFiles <<<<<")
	for {
		invalidFiles, err := chain.GetInvalidFiles()
		if err != nil {
			if err.Error() != chain.ERR_Empty {
				Out.Sugar().Infof("%v", err)
			}
			time.Sleep(time.Minute * time.Duration(tools.RandomInRange(5, 10)))
			continue
		}

		if len(invalidFiles) == 0 {
			time.Sleep(time.Minute * time.Duration(tools.RandomInRange(5, 10)))
			continue
		}

		Out.Sugar().Infof("--> Prepare to remove invalid files [%v]", len(invalidFiles))
		for x := 0; x < len(invalidFiles); x++ {
			Out.Sugar().Infof("   %v: %s", x, string(invalidFiles[x]))
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
				Out.Sugar().Infof("[%v] Cleared %v", string(invalidFiles[i]), txhash)
			} else {
				Out.Sugar().Infof("[err] [%v] Clear: %v", string(invalidFiles[i]), err)
			}
			os.Remove(filefullpath)
			os.Remove(filetagfullpath)
		}
	}
}
