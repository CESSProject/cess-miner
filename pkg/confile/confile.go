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
	"strings"

	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/cess-miner/configs"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const DefaultProfile = "conf.yaml"
const TempleteProfile = `app:
  # workspace
  workspace: "/"
  # communication port
  port: 15001
  # maximum space used, the unit is GiB
  maxusespace: 2000
  # number of cpus used, 0 means use all
  cores: 0
  # the server API endpoint
  apiendpoint: ""

chain:
  # signature account mnemonic
  mnemonic: "" 
  # staking account
  # if you fill in the staking account, the staking will be paid by the staking account,
  # otherwise the staking will be paid by the signature account.
  stakingacc: ""
  # earnings account
  earningsacc: ""
  # timeout for waiting for transaction packaging, default 12 seconds
  timeout: 12
  # rpc address list
  rpcs:
    - "wss://testnet-rpc.cess.cloud/ws/"
  # priority tee address list
  tees:`

type Confiler interface {
	Parse(fpath string) error
	ReadRpcEndpoints() []string
	ReadServicePort() uint16
	ReadWorkspace() string
	ReadMnemonic() string
	ReadStakingAcc() string
	ReadEarningsAcc() string
	ReadUseSpace() uint64
	ReadSignaturePublickey() []byte
	ReadSignatureAccount() string
	ReadUseCpu() uint32
	ReadPriorityTeeList() []string
	ReadApiEndpoint() string
}

type App struct {
	Workspace   string `name:"workspace" toml:"workspace" yaml:"workspace"`
	Port        uint16 `name:"port" toml:"port" yaml:"port"`
	Maxusespace uint64 `name:"maxusespace" toml:"maxusespace" yaml:"maxusespace"`
	Cores       uint32 `name:"cores" toml:"cores" yaml:"cores"`
	ApiEndpoint string `name:"apiendpoint" toml:"apiendpoint" yaml:"apiendpoint"`
}

type Chain struct {
	Mnemonic    string   `name:"mnemonic" toml:"mnemonic" yaml:"mnemonic"`
	Stakingacc  string   `name:"stakingacc" toml:"stakingacc" yaml:"stakingacc"`
	Earningsacc string   `name:"earningsacc" toml:"earningsacc" yaml:"earningsacc"`
	Timeout     uint16   `name:"timeout" toml:"timeout" yaml:"timeout"`
	Rpcs        []string `name:"rpcs" toml:"rpcs" yaml:"rpcs"`
	Tees        []string `name:"tees" toml:"tees" yaml:"tees"`
}

type Confile struct {
	App   `yaml:"app"`
	Chain `yaml:"chain"`
}

var _ Confiler = (*Confile)(nil)

func NewConfigFile() *Confile {
	return &Confile{}
}

func (c *Confile) Parse(fpath string) error {
	fstat, err := os.Stat(fpath)
	if err != nil {
		return err
	}
	if fstat.IsDir() {
		return errors.Errorf("The '%v' is not a file", fpath)
	}
	viper.SetConfigFile(fpath)
	viper.SetConfigType(path.Ext(fpath)[1:])

	err = viper.ReadInConfig()
	if err != nil {
		return errors.Errorf("[ReadInConfig] %v", err)
	}
	err = viper.Unmarshal(c)
	if err != nil {
		return errors.Errorf("[Unmarshal] %v", err)
	}

	if len(c.ApiEndpoint) <= 0 {
		return errors.New("'apiendpoint' can not be empty")
	}

	_, err = signature.KeyringPairFromSecret(c.Mnemonic, 0)
	if err != nil {
		return errors.Errorf("invalid mnemonic: %v", err)
	}

	if len(c.Rpcs) == 0 {
		return errors.New("cannot have empty configurations")
	}

	if c.Port < 1024 {
		return errors.Errorf("prohibit the use of system reserved port: %v", c.Port)
	}

	if c.Stakingacc != "" {
		err = sutils.VerityAddress(c.Stakingacc, sutils.CessPrefix)
		if err != nil {
			return errors.New("invalid staking account")
		}
	}

	err = sutils.VerityAddress(c.Earningsacc, sutils.CessPrefix)
	if err != nil {
		return errors.New("invalid earnings account")
	}

	fstat, err = os.Stat(c.Workspace)
	if err != nil {
		err = os.MkdirAll(c.Workspace, configs.FileMode)
		if err != nil {
			return err
		}
	} else {
		if !fstat.IsDir() {
			return errors.Errorf("the '%v' is not a directory", c.Workspace)
		}
	}

	if len(c.Tees) > 0 {
		for i := 0; i < len(c.Tees); i++ {
			if strings.HasPrefix(c.Tees[i], "http://") {
				c.Tees[i] = strings.TrimPrefix(c.Tees[i], "http://")
				c.Tees[i] = strings.TrimSuffix(c.Tees[i], "/")
				if !strings.Contains(c.Tees[i], ":") {
					c.Tees[i] = c.Tees[i] + ":80"
				}
			} else if strings.HasPrefix(c.Tees[i], "https://") {
				c.Tees[i] = strings.TrimPrefix(c.Tees[i], "https://")
				c.Tees[i] = strings.TrimSuffix(c.Tees[i], "/")
				if !strings.Contains(c.Tees[i], ":") {
					c.Tees[i] = c.Tees[i] + ":443"
				}
			} else {
				if !strings.Contains(c.Tees[i], ":") {
					c.Tees[i] = c.Tees[i] + ":80"
				}
			}
		}
	}

	// dirFreeSpace, err := utils.GetDirFreeSpace(c.Workspace)
	// if err != nil {
	// 	return errors.Wrapf(err, "[GetDirFreeSpace]")
	// }

	// if dirFreeSpace/1024/1024/1024 < c.UseSpace {
	// 	out.Warn(fmt.Sprintf("The available space is less than %dG", c.UseSpace))
	// }

	return nil
}

func (c *Confile) SetRpcAddr(rpc []string) {
	c.Rpcs = rpc
}

func (c *Confile) SetUseSpace(useSpace uint64) {
	c.Maxusespace = useSpace
}

func (c *Confile) SetCpuCores(cores int) {
	c.Cores = uint32(cores)
}

func (c *Confile) SetServicePort(port uint16) error {
	if port < 1024 {
		return errors.Errorf("Prohibit the use of system reserved port: %v", port)
	}
	c.Port = port
	return nil
}

func (c *Confile) SetWorkspace(workspace string) error {
	fstat, err := os.Stat(workspace)
	if err != nil {
		err = os.MkdirAll(workspace, configs.FileMode)
		if err != nil {
			return err
		}
	} else {
		if !fstat.IsDir() {
			return fmt.Errorf("%s is not a directory", workspace)
		}
	}
	c.Workspace = workspace
	return nil
}

func (c *Confile) SetMnemonic(mnemonic string) error {
	_, err := signature.KeyringPairFromSecret(mnemonic, 0)
	if err != nil {
		return err
	}
	c.Mnemonic = mnemonic
	return nil
}

func (c *Confile) SetEarningsAcc(earningsAcc string) error {
	err := sutils.VerityAddress(earningsAcc, sutils.CessPrefix)
	if err != nil {
		return err
	}
	c.Earningsacc = earningsAcc
	return nil
}

func (c *Confile) SetEndpoint(endpoint string) {
	c.ApiEndpoint = endpoint
}

func (c *Confile) SetPriorityTeeList(tees []string) {
	c.Tees = tees
}

/////////////////////////////////////////////

func (c *Confile) ReadRpcEndpoints() []string {
	return c.Rpcs
}

func (c *Confile) ReadServicePort() uint16 {
	return c.Port
}

func (c *Confile) ReadWorkspace() string {
	return c.Workspace
}

func (c *Confile) ReadMnemonic() string {
	return c.Mnemonic
}

func (c *Confile) ReadStakingAcc() string {
	return c.Stakingacc
}

func (c *Confile) ReadEarningsAcc() string {
	return c.Earningsacc
}

func (c *Confile) ReadSignaturePublickey() []byte {
	key, _ := signature.KeyringPairFromSecret(c.Mnemonic, 0)
	return key.PublicKey
}

func (c *Confile) ReadSignatureAccount() string {
	acc, _ := sutils.EncodePublicKeyAsCessAccount(c.ReadSignaturePublickey())
	return acc
}

func (c *Confile) ReadUseSpace() uint64 {
	return c.Maxusespace
}

func (c *Confile) ReadUseCpu() uint32 {
	return c.Cores
}

func (c *Confile) ReadPriorityTeeList() []string {
	return c.Tees
}

func (c *Confile) ReadApiEndpoint() string {
	return c.ApiEndpoint
}
