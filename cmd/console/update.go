package console

// func Command_UpdateAddress() *cobra.Command {
// 	cc := &cobra.Command{
// 		Use:                   "update_address",
// 		Short:                 "Update the miner's access address",
// 		Example:               "bucket update_address ip:port",
// 		Run:                   Command_UpdateAddress_Runfunc,
// 		DisableFlagsInUseLine: true,
// 	}
// 	return cc
// }

// func Command_UpdateIncome() *cobra.Command {
// 	cc := &cobra.Command{
// 		Use:                   "update_income",
// 		Short:                 "Update the miner's income account",
// 		Run:                   Command_UpdateIncome_Runfunc,
// 		DisableFlagsInUseLine: true,
// 	}
// 	return cc
// }

// Update the miner's access address
// func Command_UpdateAddress_Runfunc(cmd *cobra.Command, args []string) {
// 	if len(os.Args) >= 3 {
// 		data := strings.Split(os.Args[2], ":")
// 		if len(data) != 2 {
// 			log.Printf("\x1b[%dm[err]\x1b[0m You should enter something like 'bucket address ip:port[domain_name]'\n", 41)
// 			os.Exit(1)
// 		}
// 		if !tools.IsIPv4(data[0]) {
// 			log.Printf("\x1b[%dm[ok]\x1b[0m address error\n", 42)
// 			os.Exit(1)
// 		}
// 		_, err := strconv.Atoi(data[1])
// 		if err != nil {
// 			log.Printf("\x1b[%dm[ok]\x1b[0m address error\n", 42)
// 			os.Exit(1)
// 		}

// 		//Parse command arguments and  configuration file
// 		parseFlags(cmd)

// 		txhash, err := chain.UpdateAddress(configs.C.SignatureAcc, data[0], data[1])
// 		if err != nil {
// 			if err.Error() == chain.ERR_Empty {
// 				log.Println("[err] Please check your wallet balance.")
// 			} else {
// 				if txhash != "" {
// 					msg := configs.HELP_common + fmt.Sprintf(" %v\n", txhash)
// 					msg += configs.HELP_UpdateAddress
// 					log.Printf("[pending] %v\n", msg)
// 				} else {
// 					log.Printf("[err] %v.\n", err)
// 				}
// 			}
// 			os.Exit(1)
// 		}
// 		log.Printf("\x1b[%dm[ok]\x1b[0m success\n", 42)
// 		os.Exit(0)
// 	}
// 	log.Printf("\x1b[%dm[err]\x1b[0m You should enter something like 'bucket address ip:port[domain_name]'\n", 41)
// 	os.Exit(1)
// }

// // Update the miner's access address
// func Command_UpdateIncome_Runfunc(cmd *cobra.Command, args []string) {
// 	if len(os.Args) >= 3 {
// 		pubkey, err := tools.DecodeToCessPub(os.Args[2])
// 		if err != nil {
// 			log.Printf("\x1b[%dm[ok]\x1b[0m account error\n", 42)
// 			os.Exit(1)
// 		}
// 		//Parse command arguments and  configuration file
// 		parseFlags(cmd)
// 		acc, _ := types.NewAccountID(pubkey)
// 		txhash, err := chain.UpdateIncome(configs.C.SignatureAcc, *acc)
// 		if err != nil {
// 			if err.Error() == chain.ERR_Empty {
// 				log.Println("[err] Please check your wallet balance.")
// 			} else {
// 				if txhash != "" {
// 					msg := configs.HELP_common + fmt.Sprintf(" %v\n", txhash)
// 					msg += configs.HELP_UpdataBeneficiary
// 					log.Printf("[pending] %v\n", msg)
// 				} else {
// 					log.Printf("[err] %v.\n", err)
// 				}
// 			}
// 			os.Exit(1)
// 		}
// 		log.Printf("\x1b[%dm[ok]\x1b[0m success\n", 42)
// 		os.Exit(0)
// 	}
// 	log.Printf("\x1b[%dm[err]\x1b[0m You should enter something like 'bucket update_income account'\n", 41)
// 	os.Exit(1)
// }
