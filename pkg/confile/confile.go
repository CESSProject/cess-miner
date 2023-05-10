/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package confile

import (
	"fmt"
	"os"
	"path"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const DefaultProfile = "conf.yaml"
const TempleteProfile = `# The rpc endpoint of the chain node
Rpc:
  - "ws://127.0.0.1:9948/"
  - "wss://testnet-rpc0.cess.cloud/ws/"
  - "wss://testnet-rpc1.cess.cloud/ws/"
# Signature account mnemonic
Mnemonic: "xxx xxx ... xxx"
# Income account
IncomeAcc: cXxxx...xxx
# Service workspace
Workspace: /
# Service running address
Address: "127.0.0.1"
# Service listening port
Port: 15000
# Maximum space used, the unit is GIB
UseSpace: 2000`

type Confile interface {
	Parse(fpath string, ip string, port int) error
	GetRpcAddr() []string
	GetServiceAddr() string
	GetServicePort() int
	GetWorkspace() string
	GetMnemonic() string
	GetIncomeAcc() string
	GetUseSpace() uint64
	GetPublickey() []byte
	GetAccount() string
}

type confile struct {
	Rpc       []string `name:"Rpc" toml:"Rpc" yaml:"Rpc"`
	Mnemonic  string   `name:"Mnemonic" toml:"Mnemonic" yaml:"Mnemonic"`
	IncomeAcc string   `toml:"IncomeAcc" toml:"IncomeAcc" yaml:"IncomeAcc"`
	Workspace string   `name:"Workspace" toml:"Workspace" yaml:"Workspace"`
	Address   string   `name:"Address" toml:"Address" yaml:"Address"`
	Port      int      `name:"Port" toml:"Port" yaml:"Port"`
	UseSpace  uint64   `toml:"UseSpace" toml:"UseSpace" yaml:"UseSpace"`
}

func NewConfigfile() *confile {
	return &confile{}
}

func (c *confile) Parse(fpath string, ip string, port int) error {
	fstat, err := os.Stat(fpath)
	if err != nil {
		return errors.Errorf("Parse: %v", err)
	}
	if fstat.IsDir() {
		return errors.Errorf("The '%v' is not a file", fpath)
	}

	viper.SetConfigFile(fpath)
	viper.SetConfigType(path.Ext(fpath)[1:])

	err = viper.ReadInConfig()
	if err != nil {
		return errors.Errorf("ReadInConfig: %v", err)
	}
	err = viper.Unmarshal(c)
	if err != nil {
		return errors.Errorf("Unmarshal: %v", err)
	}

	_, err = signature.KeyringPairFromSecret(c.Mnemonic, 0)
	if err != nil {
		return errors.Errorf("Secret: %v", err)
	}

	if ip != "" {
		c.Address = ip
	}
	if len(c.Rpc) == 0 ||
		c.Address == "" {
		return errors.New("The configuration file cannot have empty entries")
	}

	if port != 0 {
		c.Port = port
	}
	if c.Port < 1024 {
		return errors.Errorf("Prohibit the use of system reserved port: %v", c.Port)
	}
	if c.Port > 65535 {
		return errors.New("The port number cannot exceed 65535")
	}

	utils.VerityAddress(c.IncomeAcc, utils.CESSChainTestPrefix)

	fstat, err = os.Stat(c.Workspace)
	if err != nil {
		err = os.MkdirAll(c.Workspace, configs.DirMode)
		if err != nil {
			return err
		}
	}

	if !fstat.IsDir() {
		return errors.Errorf("The '%v' is not a directory", c.Workspace)
	}

	return nil
}

func (c *confile) SetRpcAddr(rpc []string) {
	c.Rpc = rpc
}

func (c *confile) SetServiceAddr(address string) error {
	c.Address = address
	return nil
}

func (c *confile) SetUseSpace(useSpace uint64) {
	c.UseSpace = useSpace
}

func (c *confile) SetServicePort(port int) error {
	if port < 1024 {
		return errors.Errorf("Prohibit the use of system reserved port: %v", port)
	}
	if port > 65535 {
		return errors.New("The port number cannot exceed 65535")
	}
	c.Port = port
	return nil
}

func (c *confile) SetWorkspace(workspace string) error {
	fstat, err := os.Stat(workspace)
	if err != nil {
		err = os.MkdirAll(workspace, configs.DirMode)
		if err != nil {
			return err
		}
	}
	if !fstat.IsDir() {
		return fmt.Errorf("%s is not a directory", workspace)
	}
	c.Workspace = workspace
	return nil
}

func (c *confile) SetIncomeAcc(incomde string) {
	c.IncomeAcc = incomde
}

func (c *confile) SetMnemonic(mnemonic string) error {
	_, err := signature.KeyringPairFromSecret(mnemonic, 0)
	if err != nil {
		return err
	}
	c.Mnemonic = mnemonic
	return nil
}

func (c *confile) GetRpcAddr() []string {
	return c.Rpc
}

func (c *confile) GetServiceAddr() string {
	return c.Address
}

func (c *confile) GetServicePort() int {
	return c.Port
}

func (c *confile) GetWorkspace() string {
	return c.Workspace
}

func (c *confile) GetMnemonic() string {
	return c.Mnemonic
}

func (c *confile) GetIncomeAcc() string {
	return c.IncomeAcc
}

func (c *confile) GetPublickey() []byte {
	key, _ := signature.KeyringPairFromSecret(c.GetMnemonic(), 0)
	return key.PublicKey
}

func (c *confile) GetAccount() string {
	acc, _ := utils.EncodeToCESSAddr(c.GetPublickey())
	return acc
}

func (c *confile) GetUseSpace() uint64 {
	return c.UseSpace
}
