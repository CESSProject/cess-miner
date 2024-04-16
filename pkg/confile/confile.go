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

	"github.com/CESSProject/cess-bucket/configs"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const DefaultProfile = "conf.yaml"
const TempleteProfile = `# The rpc endpoint of the chain node
Rpc:
  - "wss://testnet-rpc0.cess.cloud/ws/"
  - "wss://testnet-rpc1.cess.cloud/ws/"
  - "wss://testnet-rpc2.cess.cloud/ws/"
# Bootstrap Nodes
Boot:
  - "_dnsaddr.boot-bucket-testnet.cess.cloud"
# Signature account mnemonic
Mnemonic: "xxx xxx ... xxx"
# Staking account
# If you fill in the staking account, the staking will be paid by the staking account,
# otherwise the staking will be paid by the signature account.
StakingAcc: "cXxxx...xxx"
# earnings account
EarningsAcc: cXxxx...xxx
# Service workspace
Workspace: /
# P2P communication port
Port: 4001
# Maximum space used, the unit is GiB
UseSpace: 2000
# Number of cpu's used, 0 means use all
UseCpu: 0
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
	rpc         []string `name:"Rpc" toml:"Rpc" yaml:"Rpc"`
	boot        []string `name:"Boot" toml:"Boot" yaml:"Boot"`
	mnemonic    string   `name:"Mnemonic" toml:"Mnemonic" yaml:"Mnemonic"`
	stakingAcc  string   `name:"StakingAcc" toml:"StakingAcc" yaml:"StakingAcc"`
	earningsAcc string   `name:"EarningsAcc" toml:"EarningsAcc" yaml:"EarningsAcc"`
	workspace   string   `name:"Workspace" toml:"Workspace" yaml:"Workspace"`
	port        int      `name:"Port" toml:"Port" yaml:"Port"`
	useSpace    uint64   `name:"UseSpace" toml:"UseSpace" yaml:"UseSpace"`
	useCpu      uint8    `name:"UseCpu" toml:"UseCpu" yaml:"UseCpu"`
	teeList     []string `name:"TeeList" toml:"TeeList" yaml:"TeeList"`
}

var _ Confiler = (*Confile)(nil)

func NewEmptyConfigfile() *Confile {
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

	_, err = signature.KeyringPairFromSecret(c.mnemonic, 0)
	if err != nil {
		return errors.Errorf("invalid mnemonic: %v", err)
	}

	if len(c.rpc) == 0 ||
		len(c.boot) == 0 {
		return errors.New("cannot have empty configurations")
	}

	if c.port < 1024 {
		return errors.Errorf("prohibit the use of system reserved port: %v", c.port)
	}

	if c.port > 65535 {
		return errors.New("the port number cannot exceed 65535")
	}

	if c.stakingAcc != "" {
		err = sutils.VerityAddress(c.stakingAcc, sutils.CessPrefix)
		if err != nil {
			return errors.New("invalid staking account")
		}
	}

	err = sutils.VerityAddress(c.earningsAcc, sutils.CessPrefix)
	if err != nil {
		return errors.New("invalid earnings account")
	}

	fstat, err = os.Stat(c.workspace)
	if err != nil {
		err = os.MkdirAll(c.workspace, configs.FileMode)
		if err != nil {
			return err
		}
	} else {
		if !fstat.IsDir() {
			return errors.Errorf("the '%v' is not a directory", c.workspace)
		}
	}

	if len(c.teeList) > 0 {
		for i := 0; i < len(c.teeList); i++ {
			if strings.HasPrefix(c.teeList[i], "http://") {
				c.teeList[i] = strings.TrimPrefix(c.teeList[i], "http://")
				c.teeList[i] = strings.TrimSuffix(c.teeList[i], "/")
				if !strings.Contains(c.teeList[i], ":") {
					c.teeList[i] = c.teeList[i] + ":80"
				}
			} else if strings.HasPrefix(c.teeList[i], "https://") {
				c.teeList[i] = strings.TrimPrefix(c.teeList[i], "https://")
				c.teeList[i] = strings.TrimSuffix(c.teeList[i], "/")
				if !strings.Contains(c.teeList[i], ":") {
					c.teeList[i] = c.teeList[i] + ":443"
				}
			} else {
				if !strings.Contains(c.teeList[i], ":") {
					c.teeList[i] = c.teeList[i] + ":80"
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
	c.rpc = rpc
}

func (c *Confile) SetBootNodes(boot []string) {
	c.boot = boot
}

func (c *Confile) SetUseSpace(useSpace uint64) {
	c.useSpace = useSpace
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
	c.port = port
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
	c.workspace = workspace
	return nil
}

func (c *Confile) SetMnemonic(mnemonic string) error {
	_, err := signature.KeyringPairFromSecret(mnemonic, 0)
	if err != nil {
		return err
	}
	c.mnemonic = mnemonic
	return nil
}

func (c *Confile) SetEarningsAcc(earningsAcc string) error {
	err := sutils.VerityAddress(earningsAcc, sutils.CessPrefix)
	if err != nil {
		return err
	}
	c.earningsAcc = earningsAcc
	return nil
}

func (c *Confile) SetPriorityTeeList(tees []string) {
	c.teeList = tees
}

/////////////////////////////////////////////

func (c *Confile) ReadRpcEndpoints() []string {
	return c.rpc
}

func (c *Confile) ReadBootnodes() []string {
	return c.boot
}

func (c *Confile) ReadServicePort() int {
	return c.port
}

func (c *Confile) ReadWorkspace() string {
	return c.workspace
}

func (c *Confile) ReadMnemonic() string {
	return c.mnemonic
}

func (c *Confile) ReadStakingAcc() string {
	return c.stakingAcc
}

func (c *Confile) ReadEarningsAcc() string {
	return c.earningsAcc
}

func (c *Confile) ReadSignaturePublickey() []byte {
	key, _ := signature.KeyringPairFromSecret(c.mnemonic, 0)
	return key.PublicKey
}

func (c *Confile) ReadSignatureAccount() string {
	acc, _ := sutils.EncodePublicKeyAsCessAccount(c.ReadSignaturePublickey())
	return acc
}

func (c *Confile) ReadUseSpace() uint64 {
	return c.useSpace
}

func (c *Confile) ReadUseCpu() uint8 {
	return c.useCpu
}

func (c *Confile) ReadPriorityTeeList() []string {
	return c.teeList
}
