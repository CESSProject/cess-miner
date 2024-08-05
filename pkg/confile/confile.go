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
const TempleteProfile = `# The rpc endpoint of the chain node
Rpc:
  # testnet
  - "wss://testnet-rpc.cess.network/ws/"
# Bootstrap Nodes
Boot:
  # testnet
  - "_dnsaddr.boot-miner-testnet.cess.network"
# Signature account mnemonic
Mnemonic: ""
# Staking account
# If you fill in the staking account, the staking will be paid by the staking account,
# otherwise the staking will be paid by the signature account.
StakingAcc: ""
# earnings account
EarningsAcc: ""
# Service workspace
Workspace: ""
# P2P communication port
Port: 4001
# Maximum space used, the unit is GiB
UseSpace: 2000
# Number of cpu's used, 0 means use all
UseCpu: 1
# Priority tee list address
TeeList:`

type Confiler interface {
	Parse(fpath string) error
	ReadRpcEndpoints() []string
	ReadBootnodes() []string
	ReadServicePort() int
	ReadWorkspace() string
	ReadMnemonic() string
	ReadStakingAcc() string
	ReadEarningsAcc() string
	ReadUseSpace() uint64
	ReadSignaturePublickey() []byte
	ReadSignatureAccount() string
	ReadUseCpu() uint8
	ReadPriorityTeeList() []string
}

type Confile struct {
	Rpc         []string `name:"Rpc" toml:"Rpc" yaml:"Rpc"`
	Boot        []string `name:"Boot" toml:"Boot" yaml:"Boot"`
	Mnemonic    string   `name:"Mnemonic" toml:"Mnemonic" yaml:"Mnemonic"`
	StakingAcc  string   `name:"StakingAcc" toml:"StakingAcc" yaml:"StakingAcc"`
	EarningsAcc string   `name:"EarningsAcc" toml:"EarningsAcc" yaml:"EarningsAcc"`
	Workspace   string   `name:"Workspace" toml:"Workspace" yaml:"Workspace"`
	Port        int      `name:"Port" toml:"Port" yaml:"Port"`
	UseSpace    uint64   `name:"UseSpace" toml:"UseSpace" yaml:"UseSpace"`
	UseCpu      uint8    `name:"UseCpu" toml:"UseCpu" yaml:"UseCpu"`
	TeeList     []string `name:"TeeList" toml:"TeeList" yaml:"TeeList"`
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
	_, err = signature.KeyringPairFromSecret(c.Mnemonic, 0)
	if err != nil {
		return errors.Errorf("invalid mnemonic: %v", err)
	}

	if len(c.Rpc) == 0 ||
		len(c.Boot) == 0 {
		return errors.New("cannot have empty configurations")
	}

	if c.Port < 1024 {
		return errors.Errorf("prohibit the use of system reserved port: %v", c.Port)
	}

	if c.Port > 65535 {
		return errors.New("the port number cannot exceed 65535")
	}

	if c.StakingAcc != "" {
		err = sutils.VerityAddress(c.StakingAcc, sutils.CessPrefix)
		if err != nil {
			return errors.New("invalid staking account")
		}
	}

	err = sutils.VerityAddress(c.EarningsAcc, sutils.CessPrefix)
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

	if len(c.TeeList) > 0 {
		for i := 0; i < len(c.TeeList); i++ {
			if strings.HasPrefix(c.TeeList[i], "http://") {
				c.TeeList[i] = strings.TrimPrefix(c.TeeList[i], "http://")
				c.TeeList[i] = strings.TrimSuffix(c.TeeList[i], "/")
				if !strings.Contains(c.TeeList[i], ":") {
					c.TeeList[i] = c.TeeList[i] + ":80"
				}
			} else if strings.HasPrefix(c.TeeList[i], "https://") {
				c.TeeList[i] = strings.TrimPrefix(c.TeeList[i], "https://")
				c.TeeList[i] = strings.TrimSuffix(c.TeeList[i], "/")
				if !strings.Contains(c.TeeList[i], ":") {
					c.TeeList[i] = c.TeeList[i] + ":443"
				}
			} else {
				if !strings.Contains(c.TeeList[i], ":") {
					c.TeeList[i] = c.TeeList[i] + ":80"
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
	c.Rpc = rpc
}

func (c *Confile) SetBootNodes(boot []string) {
	c.Boot = boot
}

func (c *Confile) SetUseSpace(useSpace uint64) {
	c.UseSpace = useSpace
}

func (c *Confile) SetServicePort(port int) error {
	if sutils.IsPortInUse(port) {
		return errors.New("This port is in use")
	}

	if port < 1024 {
		return errors.Errorf("Prohibit the use of system reserved port: %v", port)
	}
	if port > 65535 {
		return errors.New("The port number cannot exceed 65535")
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
	c.EarningsAcc = earningsAcc
	return nil
}

func (c *Confile) SetPriorityTeeList(tees []string) {
	c.TeeList = tees
}

/////////////////////////////////////////////

func (c *Confile) ReadRpcEndpoints() []string {
	return c.Rpc
}

func (c *Confile) ReadBootnodes() []string {
	return c.Boot
}

func (c *Confile) ReadServicePort() int {
	return c.Port
}

func (c *Confile) ReadWorkspace() string {
	return c.Workspace
}

func (c *Confile) ReadMnemonic() string {
	return c.Mnemonic
}

func (c *Confile) ReadStakingAcc() string {
	return c.StakingAcc
}

func (c *Confile) ReadEarningsAcc() string {
	return c.EarningsAcc
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
	return c.UseSpace
}

func (c *Confile) ReadUseCpu() uint8 {
	return c.UseCpu
}

func (c *Confile) ReadPriorityTeeList() []string {
	return c.TeeList
}
