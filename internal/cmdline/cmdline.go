package cmdline

import (
	"flag"
	"fmt"
	"os"
	"storage-mining/configs"
	"storage-mining/tools"

	"github.com/spf13/viper"
)

// command line parameters
func CmdlineInit() {
	var (
		err          error
		helpInfo     bool
		showVersion  bool
		confFilePath string
	)
	flag.BoolVar(&helpInfo, "h", false, "Print Help (this message) and exit")
	flag.BoolVar(&showVersion, "v", false, "Print version information and exit")
	flag.StringVar(&confFilePath, "c", "", "Run the program directly to generate\n"+
		"Specify the `configuration file` to ensure that the program runs correctly")
	//flag.BoolVar(&configs.MinerEvent_Exit, "e", false, "Exit the cess mining network")
	//flag.BoolVar(&configs.MinerEvent_RenewalTokens, "t", false, "Automatically register and renewal tokens")
	flag.Usage = usage
	flag.Parse()
	if helpInfo {
		flag.Usage()
		os.Exit(configs.Exit_Normal)
	}
	if showVersion {
		fmt.Println(configs.Version)
		os.Exit(configs.Exit_Normal)
	}
	if confFilePath == "" {
		tools.WriteStringtoFile(configs.ConfigFile_Templete, configs.DefaultConfigurationFileName)
		fmt.Printf("\x1b[%dm[note]\x1b[0m Generate default configuration file,use '-h' to view the help information.\n", 43)
		os.Exit(configs.Exit_Normal)
	}
	_, err = os.Stat(confFilePath)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file does not exist\n", 41, confFilePath)
		os.Exit(configs.Exit_ConfFileNotExist)
	}

	viper.SetConfigFile(confFilePath)
	viper.SetConfigType("toml")
	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file type error\n", 41, confFilePath)
		os.Exit(configs.Exit_ConfFileTypeError)
	}
	err = viper.Unmarshal(configs.Confile)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file format error\n", 41, confFilePath)
		os.Exit(configs.Exit_ConfFileFormatError)
	}
}

func usage() {
	str := `CESS-Storage-Mining

Usage:
    `
	str += fmt.Sprintf("%v", os.Args[0])
	str += ` [arguments] [file]

Arguments:
`
	fmt.Fprintf(os.Stdout, str)
	flag.PrintDefaults()
}
