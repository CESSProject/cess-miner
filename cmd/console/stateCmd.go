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

package console

import (
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/confile"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/spf13/cobra"
)

// stateCmd query your own details on-chain
//
// Usage:
//
//	bucket state
func stateCmd(cmd *cobra.Command, args []string) {
	// config file
	var configFilePath string
	configpath1, _ := cmd.Flags().GetString("config")
	configpath2, _ := cmd.Flags().GetString("c")
	if configpath1 != "" {
		configFilePath = configpath1
	} else {
		configFilePath = configpath2
	}

	confile := confile.NewConfigfile()
	if err := confile.Parse(configFilePath); err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// chain client
	chn, err := chain.NewChainClient(
		confile.GetRpcAddr(),
		confile.GetCtrlPrk(),
		confile.GetIncomeAcc(),
		configs.TimeOut_WaitBlock,
	)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	//Query your own information on the chain
	mData, err := chn.GetMinerInfo(chn.GetPublicKey())
	if err != nil {
		if err.Error() == chain.ERR_Empty {
			log.Printf("[err] Not found: %v\n", err)
			os.Exit(1)
		}
		log.Printf("[err] Query error: %v\n", err)
		os.Exit(1)
	}
	mData.Collaterals.Div(new(big.Int).SetBytes(mData.Collaterals.Bytes()), big.NewInt(1000000000000))
	addr := fmt.Sprintf("%d.%d.%d.%d:%d", mData.Ip.Value[0], mData.Ip.Value[1], mData.Ip.Value[2], mData.Ip.Value[3], mData.Ip.Port)
	var power, space float32
	var power_unit, space_unit string
	count := 0
	for mData.Idle_space.BitLen() > int(16) {
		mData.Idle_space.Div(new(big.Int).SetBytes(mData.Idle_space.Bytes()), big.NewInt(configs.SIZE_1KiB))
		count++
	}
	if mData.Idle_space.Int64() > configs.SIZE_1KiB {
		power = float32(mData.Idle_space.Int64()) / float32(configs.SIZE_1KiB)
		count++
	} else {
		power = float32(mData.Idle_space.Int64())
	}
	switch count {
	case 0:
		power_unit = "Byte"
	case 1:
		power_unit = "KiB"
	case 2:
		power_unit = "MiB"
	case 3:
		power_unit = "GiB"
	case 4:
		power_unit = "TiB"
	case 5:
		power_unit = "PiB"
	case 6:
		power_unit = "EiB"
	case 7:
		power_unit = "ZiB"
	case 8:
		power_unit = "YiB"
	case 9:
		power_unit = "NiB"
	case 10:
		power_unit = "DiB"
	default:
		power_unit = fmt.Sprintf("DiB(%v)", count-10)
	}
	count = 0
	for mData.Service_space.BitLen() > int(16) {
		mData.Service_space.Div(new(big.Int).SetBytes(mData.Service_space.Bytes()), big.NewInt(configs.SIZE_1KiB))
		count++
	}
	if mData.Service_space.Int64() > configs.SIZE_1KiB {
		space = float32(mData.Service_space.Int64()) / float32(configs.SIZE_1KiB)
		count++
	} else {
		space = float32(mData.Service_space.Int64())
	}

	switch count {
	case 0:
		space_unit = "Byte"
	case 1:
		space_unit = "KiB"
	case 2:
		space_unit = "MiB"
	case 3:
		space_unit = "GiB"
	case 4:
		space_unit = "TiB"
	case 5:
		space_unit = "PiB"
	case 6:
		space_unit = "EiB"
	case 7:
		space_unit = "ZiB"
	case 8:
		space_unit = "YiB"
	case 9:
		space_unit = "NiB"
	case 10:
		space_unit = "DiB"
	default:
		power_unit = fmt.Sprintf("DiB(%v)", count-10)
	}

	acc, _ := utils.EncodePublicKeyAsCessAccount(chn.GetPublicKey())
	//print your own details
	fmt.Printf("Miner Account: %v\nIncome Account: %v\nState: %v\nIdle Space: %.2f %v\nService Space: %.2f %v\nPledgeTokens: %v TCESS\nService Address: %v\n",
		acc, chn.GetIncomeAccount(), string(mData.State), power, power_unit, space, space_unit, mData.Collaterals, string(addr))
	os.Exit(0)
}
