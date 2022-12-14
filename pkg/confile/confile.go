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

package confile

import (
	"fmt"
	"os"
	"path"

	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	DefaultConfigurationFile      = "./conf.toml"
	ConfigurationFileTemplateName = "conf.toml"
	ConfigurationFileTemplete     = `# The rpc address of the chain node
RpcAddr      = ""
# Path to the mounted disk where the data is saved
MountedPath  = ""
# Total space used to store files, the unit is GB
StorageSpace = 0
# The IP of the machine running the mining service
ServiceIP    = ""
# Port number monitored by the mining service
ServicePort  = 0
# The address of income account
IncomeAcc    = ""
# phrase of the signature account
SignatureAcc = ""
# If 'ServiceIP' is not public IP, You can set up a domain name
DomainName   = ""`
)

type Confiler interface {
	Parse(path string) error
	GetRpcAddr() string
	GetServiceAddr() string
	GetServicePort() string
	GetServicePortNum() int
	GetDataDir() string
	GetCtrlPrk() string
	GetIncomeAcc() string
}

type confile struct {
	RpcAddr      string `toml:"RpcAddr"`
	MountedPath  string `toml:"MountedPath"`
	StorageSpace uint64 `toml:"StorageSpace"`
	ServiceIP    string `toml:"ServiceIP"`
	ServicePort  uint32 `toml:"ServicePort"`
	IncomeAcc    string `toml:"IncomeAcc"`
	SignatureAcc string `toml:"SignatureAcc"`
	DomainName   string `toml:"DomainName"`
}

func NewConfigfile() Confiler {
	return &confile{}
}

func (c *confile) Parse(fpath string) error {
	var confilePath = fpath
	if confilePath == "" {
		confilePath = DefaultConfigurationFile
	}
	fstat, err := os.Stat(confilePath)
	if err != nil {
		return errors.Errorf("Parse: %v", err)
	}
	if fstat.IsDir() {
		return errors.Errorf("The '%v' is not a file", confilePath)
	}

	viper.SetConfigFile(confilePath)
	viper.SetConfigType(path.Ext(confilePath)[1:])

	err = viper.ReadInConfig()
	if err != nil {
		return errors.Errorf("ReadInConfig: %v", err)
	}
	err = viper.Unmarshal(c)
	if err != nil {
		return errors.Errorf("Unmarshal: %v", err)
	}

	_, err = signature.KeyringPairFromSecret(c.SignatureAcc, 0)
	if err != nil {
		return errors.Errorf("SignatureAcc: %v", err)
	}

	_, err = utils.DecodePublicKeyOfCessAccount(c.IncomeAcc)
	if err != nil {
		return errors.Errorf("Decode SignatureAcc: %v", err)
	}

	if c.MountedPath == "" ||
		c.RpcAddr == "" ||
		c.ServiceIP == "" {
		return errors.New("The configuration file cannot have empty entries")
	}

	if !utils.IsIPv4(c.ServiceIP) {
		return errors.New("Please enter the ipv4 address")
	}

	extIp, err := utils.GetExternalIp()
	if err == nil {
		if extIp != c.ServiceIP {
			msg := fmt.Sprintf("It is detected that your public IP address is: %s, Please check whether your configuration is correct.", extIp)
			return errors.New(msg)
		}
	}

	if c.ServicePort < 1024 {
		return errors.Errorf("Prohibit the use of system reserved port: %v", c.ServicePort)
	}
	if c.ServicePort > 65535 {
		return errors.New("The port number cannot exceed 65535")
	}

	_, err = utils.GetMountPathInfo(c.MountedPath)
	if err != nil {
		return fmt.Errorf("%v not mounted", c.MountedPath)
	}
	return nil
}

func (c *confile) GetRpcAddr() string {
	return c.RpcAddr
}

func (c *confile) GetServiceAddr() string {
	return c.ServiceIP
}

func (c *confile) GetServicePort() string {
	return fmt.Sprintf("%v", c.ServicePort)
}

func (c *confile) GetServicePortNum() int {
	return int(c.ServicePort)
}

func (c *confile) GetDataDir() string {
	return c.MountedPath
}

func (c *confile) GetCtrlPrk() string {
	return c.SignatureAcc
}

func (c *confile) GetIncomeAcc() string {
	return c.IncomeAcc
}
