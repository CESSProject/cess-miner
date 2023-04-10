package console

// // Storage miner registration information on the chain
// func Command_Register_Runfunc(cmd *cobra.Command, args []string) {
// 	//Parse command arguments and  configuration file
// 	parseFlags(cmd)

// 	api, err := chain.NewRpcClient(configs.C.RpcAddr)
// 	if err != nil {
// 		fmt.Printf("\x1b[%dm[err]\x1b[0m Connection error: %v\n", 41, err)
// 		os.Exit(1)
// 	}

// 	//Query your own information on the chain
// 	_, err = chain.GetMinerInfo(api)
// 	if err != nil {
// 		if err.Error() == chain.ERR_Empty {
// 			err = register(api)
// 			if err != nil {
// 				fmt.Printf("\x1b[%dm[err]\x1b[0m Register failed: %v\n", 41, err)
// 				os.Exit(1)
// 			}
// 			os.Exit(0)
// 		}
// 		fmt.Printf("\x1b[%dm[err]\x1b[0m Query error: %v\n", 41, err)
// 		os.Exit(1)
// 	}

// 	fmt.Printf("\x1b[%dm[ok]\x1b[0m You are registered\n", 42)
// 	os.Exit(0)
// }

// func register(api *gsrpc.SubstrateAPI) error {
// 	//Calculate the deposit based on the size of the storage space
// 	pledgeTokens := 2000 * (configs.C.StorageSpace / 1024)
// 	if configs.C.StorageSpace%1024 != 0 {
// 		pledgeTokens += 2000
// 	}

// 	_, err := os.Stat(configs.BaseDir)
// 	if err == nil {
// 		bkpname := configs.BaseDir + "_" + fmt.Sprintf("%v", time.Now().Unix()) + "_bkp"
// 		os.Rename(configs.BaseDir, bkpname)
// 	}

// 	//Registration information on the chain
// 	txhash, err := chain.Register(
// 		api,
// 		configs.C.IncomeAcc,
// 		configs.C.ServiceIP,
// 		uint16(configs.C.ServicePort),
// 		pledgeTokens,
// 	)
// 	if err != nil {
// 		if err.Error() == chain.ERR_Empty {
// 			log.Println("[err] Please check your wallet balance.")
// 		} else {
// 			if txhash != "" {
// 				msg := configs.HELP_common + fmt.Sprintf(" %v\n", txhash)
// 				msg += configs.HELP_register
// 				log.Printf("[pending] %v\n", msg)
// 			} else {
// 				log.Printf("[err] %v.\n", err)
// 			}
// 		}
// 		return err
// 	}

// 	fmt.Printf("\x1b[%dm[ok]\x1b[0m Registration success\n", 42)

// 	//Create the storage data directory
// 	err = os.MkdirAll(configs.BaseDir, os.ModeDir)
// 	if err != nil {
// 		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
// 		return err
// 	}

// 	//Create log directory
// 	configs.LogfileDir = filepath.Join(configs.BaseDir, configs.LogfileDir)
// 	if err = tools.CreatDirIfNotExist(configs.LogfileDir); err != nil {
// 		goto Err
// 	}
// 	//Create space directory
// 	configs.SpaceDir = filepath.Join(configs.BaseDir, configs.SpaceDir)
// 	if err = tools.CreatDirIfNotExist(configs.SpaceDir); err != nil {
// 		goto Err
// 	}
// 	//Create file directory
// 	configs.FilesDir = filepath.Join(configs.BaseDir, configs.FilesDir)
// 	if err = tools.CreatDirIfNotExist(configs.FilesDir); err != nil {
// 		goto Err
// 	}

// 	log.Println(configs.LogfileDir)
// 	log.Println(configs.SpaceDir)
// 	log.Println(configs.FilesDir)

// 	//Initialize the logger
// 	logger.LoggerInit()

// 	//Record registration information to the log
// 	Out.Sugar().Infof("Registration message:")
// 	Out.Sugar().Infof("ChainAddr:%v", configs.C.RpcAddr)
// 	Out.Sugar().Infof("Register transaction hash:%v", txhash)
// 	return nil
// Err:
// 	log.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
// 	return err
// }

// func register_if() (bool, error) {
// 	api, err := chain.GetRpcClient_Safe(configs.C.RpcAddr)
// 	defer chain.Free()
// 	if err != nil {
// 		return false, err
// 	}

// 	// sync block
// 	for {
// 		ok, err := chain.GetSyncStatus(api)
// 		if err != nil {
// 			return false, err
// 		}
// 		if !ok {
// 			break
// 		}
// 		log.Println("In sync block...")
// 		time.Sleep(configs.BlockInterval)
// 	}
// 	log.Println("Complete synchronization of primary network block data")

// 	//Query your own information on the chain
// 	_, err = chain.GetMinerInfo(api)
// 	if err != nil {
// 		if err.Error() == chain.ERR_Empty {
// 			return false, nil
// 		}
// 		return false, err
// 	}
// 	return true, nil
// }

// // Parse command arguments
// func parseFlags(cmd *cobra.Command) {
// 	//Get custom configuration file
// 	configpath1, _ := cmd.Flags().GetString("config")
// 	configpath2, _ := cmd.Flags().GetString("c")
// 	if configpath1 != "" {
// 		configs.ConfFilePath = configpath1
// 	} else {
// 		configs.ConfFilePath = configpath2
// 	}
// 	//Parse the configuration file
// 	parseProfile()
// }

// func parseProfile() {
// 	var (
// 		err          error
// 		confFilePath string
// 	)
// 	if configs.ConfFilePath == "" {
// 		confFilePath = "./conf.toml"
// 	} else {
// 		confFilePath = configs.ConfFilePath
// 	}

// 	f, err := os.Stat(confFilePath)
// 	if err != nil {
// 		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file does not exist\n", 41, confFilePath)
// 		os.Exit(1)
// 	}
// 	if f.IsDir() {
// 		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' is not a file\n", 41, confFilePath)
// 		os.Exit(1)
// 	}

// 	viper.SetConfigFile(confFilePath)
// 	viper.SetConfigType("toml")

// 	err = viper.ReadInConfig()
// 	if err != nil {
// 		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file type error\n", 41, confFilePath)
// 		os.Exit(1)
// 	}
// 	err = viper.Unmarshal(configs.C)
// 	if err != nil {
// 		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file format error\n", 41, confFilePath)
// 		os.Exit(1)
// 	}

// 	if configs.C.MountedPath == "" ||
// 		configs.C.ServiceIP == "" ||
// 		configs.C.IncomeAcc == "" ||
// 		configs.C.SignatureAcc == "" {
// 		fmt.Printf("\x1b[%dm[err]\x1b[0m The configuration file cannot have empty entries\n", 41)
// 		os.Exit(1)
// 	}

// 	if configs.C.ServicePort < 1024 {
// 		fmt.Printf("\x1b[%dm[err]\x1b[0m Prohibit the use of system reserved port: %v\n", 41, configs.C.ServicePort)
// 		os.Exit(1)
// 	}
// 	if configs.C.ServicePort > 65535 {
// 		fmt.Printf("\x1b[%dm[err]\x1b[0m The port number cannot exceed 65535\n", 41)
// 		os.Exit(1)
// 	}

// 	_, err = tools.GetMountPathInfo(configs.C.MountedPath)
// 	if err != nil {
// 		fmt.Printf("\x1b[%dm[err]\x1b[0m '%v' %v\n", 41, configs.C.MountedPath, err)
// 		os.Exit(1)
// 	}

// 	acc, err := chain.GetSelfPublicKey(configs.C.SignatureAcc)
// 	if err != nil {
// 		log.Printf("[err] %v\n", err)
// 		os.Exit(1)
// 	}

// 	addr, err := tools.EncodeToCESSAddr(acc)
// 	if err != nil {
// 		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
// 		os.Exit(1)
// 	}

// 	pattern.SetMinerAcc(acc)
// 	pattern.SetMinerSignAddr(configs.C.IncomeAcc)
// 	configs.BaseDir = filepath.Join(configs.C.MountedPath, addr, configs.BaseDir)
// }
