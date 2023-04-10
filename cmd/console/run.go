package console

import (
	"log"
	"os"
	"path/filepath"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	"github.com/CESSProject/cess-bucket/pkg/cache"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	sdkgo "github.com/CESSProject/sdk-go"
	"github.com/spf13/cobra"
)

// runCmd is used to start the service
//
// Usage:
//
//	bucket run
func runCmd(cmd *cobra.Command, args []string) {
	var (
		err      error
		logDir   string
		cacheDir string
		n        = node.New()
	)

	// Build profile instances
	n.Cfg, err = buildConfigFile(cmd, "", 0)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	//Build client
	n.Cli, err = sdkgo.New(
		configs.Name,
		sdkgo.ConnectRpcAddrs(n.Cfg.GetRpcAddr()),
		sdkgo.ListenPort(n.Cfg.GetServicePort()),
		sdkgo.Workspace(n.Cfg.GetWorkspace()),
		sdkgo.Mnemonic(n.Cfg.GetMnemonic()),
	)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	_, err = n.Cli.Register(configs.Name, "", 0)
	if err != nil {
		log.Println("Register err: ", err)
		os.Exit(1)
	}

	// Build data directory
	logDir, cacheDir, n.SpaceDir, n.FileDir, n.TmpDir, err = buildDir(n.Cli.Workspace())
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// Build cache instance
	n.Cach, err = buildCache(cacheDir)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	//Build log instance
	n.Log, err = buildLogs(logDir)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// run
	n.Run()
}

func buildConfigFile(cmd *cobra.Command, ip4 string, port int) (confile.Confile, error) {
	var conFilePath string
	configpath1, _ := cmd.Flags().GetString("config")
	configpath2, _ := cmd.Flags().GetString("c")
	if configpath1 != "" {
		conFilePath = configpath1
	} else if configpath2 != "" {
		conFilePath = configpath2
	} else {
		conFilePath = configs.DefaultConfigFile
	}

	cfg := confile.NewConfigfile()
	err := cfg.Parse(conFilePath, ip4, port)
	if err == nil {
		return cfg, err
	}

	rpc, err := cmd.Flags().GetString("rpc")
	if err != nil {
		return cfg, err
	}
	workspace, err := cmd.Flags().GetString("ws")
	if err != nil {
		return cfg, err
	}
	ip, err := cmd.Flags().GetString("ip")
	if err != nil {
		return cfg, err
	}
	port, err = cmd.Flags().GetInt("port")
	if err != nil {
		port, err = cmd.Flags().GetInt("p")
		if err != nil {
			return cfg, err
		}
	}
	cfg.SetRpcAddr([]string{rpc})
	err = cfg.SetWorkspace(workspace)
	if err != nil {
		return cfg, err
	}
	err = cfg.SetServiceAddr(ip)
	if err != nil {
		return cfg, err
	}
	err = cfg.SetServicePort(port)
	if err != nil {
		return cfg, err
	}
	mnemonic, err := utils.PasswdWithMask("Please enter your mnemonic and press Enter to end:", "", "")
	if err != nil {
		return cfg, err
	}
	err = cfg.SetMnemonic(mnemonic)
	if err != nil {
		return cfg, err
	}
	return cfg, nil
}

// func register(chn chain.IChain, cfg confile.IConfile) error {
// 	//Calculate the deposit based on the size of the storage space
// 	pledgeTokens := configs.DepositPerTiB * cfg.GetStorageSpaceOnTiB()
// 	// Get report
// 	var report node.Report
// 	err := node.GetReportReq(configs.URL_GetReport_Callback, cfg.GetServiceAddr(), cfg.GetSgxPortNum(), configs.URL_GetReport)
// 	if err != nil {
// 		return errors.New("Please start the sgx service first")
// 	}

// 	timeout := time.NewTimer(configs.TimeOut_WaitReport)
// 	defer timeout.Stop()
// 	select {
// 	case <-timeout.C:
// 		return errors.New("Timed out waiting for sgx report")
// 	case report = <-node.Ch_Report:
// 	}

// 	if report.Cert == "" || report.Ias_sig == "" || report.Quote == "" || report.Quote_sig == "" {
// 		return errors.New("Invalid sgx report")
// 	}

// 	sig, err := hex.DecodeString(report.Quote_sig)
// 	if err != nil {
// 		return errors.New("Invalid sgx report quote_sig")
// 	}

// 	//Registration information on the chain
// 	txhash, err := chn.Register(
// 		cfg.GetIncomeAcc(),
// 		cfg.GetServiceAddr(),
// 		uint16(cfg.GetServicePortNum()),
// 		pledgeTokens,
// 		types.NewBytes([]byte(report.Cert)),
// 		types.NewBytes([]byte(report.Ias_sig)),
// 		types.NewBytes([]byte(report.Quote)),
// 		types.NewBytes(sig),
// 	)
// 	if err != nil {
// 		if err.Error() == chain.ERR_Empty {
// 			log.Println("[err] Please check if the wallet is registered and its balance.")
// 		} else {
// 			if txhash != "" {
// 				msg := configs.HELP_Head + fmt.Sprintf(" %v\n", txhash)
// 				msg += fmt.Sprintf("%v\n", configs.HELP_register)
// 				msg += configs.HELP_Tail
// 				log.Printf("[pending] %v\n", msg)
// 			} else {
// 				log.Printf("[err] %v.\n", err)
// 			}
// 		}
// 		return err
// 	}

// 	ctrlAcc, err := chn.GetCessAccount()
// 	if err != nil {
// 		return err
// 	}
// 	baseDir := filepath.Join(cfg.GetMountedPath(), ctrlAcc, configs.BaseDir)

// 	fstat, err := os.Stat(baseDir)
// 	if err == nil {
// 		if fstat.IsDir() {
// 			os.RemoveAll(baseDir)
// 		}
// 	}

// 	log.Println("Registration success")
// 	return nil
// }

func buildDir(workspace string) (string, string, string, string, string, error) {
	logDir := filepath.Join(workspace, configs.LogDir)
	if err := os.MkdirAll(logDir, configs.DirPermission); err != nil {
		return "", "", "", "", "", err
	}

	cacheDir := filepath.Join(workspace, configs.DbDir)
	if err := os.MkdirAll(cacheDir, configs.DirPermission); err != nil {
		return "", "", "", "", "", err
	}

	spaceDir := filepath.Join(workspace, configs.SpaceDir)
	if err := os.MkdirAll(spaceDir, configs.DirPermission); err != nil {
		return "", "", "", "", "", err
	}

	fileDir := filepath.Join(workspace, configs.FileDir)
	if err := os.MkdirAll(fileDir, configs.DirPermission); err != nil {
		return "", "", "", "", "", err
	}

	tmpDir := filepath.Join(workspace, configs.TmpDir)
	if err := os.MkdirAll(tmpDir, configs.DirPermission); err != nil {
		return "", "", "", "", "", err
	}

	log.Println(workspace)
	return logDir, cacheDir, spaceDir, fileDir, tmpDir, nil
}

func buildCache(cacheDir string) (cache.Cache, error) {
	return cache.NewCache(cacheDir, 0, 0, configs.NameSpace)
}

func buildLogs(logDir string) (logger.Logger, error) {
	var logs_info = make(map[string]string)
	for _, v := range configs.LogFiles {
		logs_info[v] = filepath.Join(logDir, v+".log")
	}
	return logger.NewLogs(logs_info)
}
