/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	cess "github.com/CESSProject/cess-go-sdk"
	"github.com/CESSProject/cess-go-sdk/config"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/CESSProject/p2p-go/out"
	"github.com/btcsuite/btcutil/base58"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// Query miner state
func Command_State_Runfunc(cmd *cobra.Command, args []string) {
	var (
		err error
		n   = node.New()
	)

	// Build profile instances
	n.Confile, err = buildAuthenticationConfig(cmd)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}

	// Build client
	n.SDK, err = cess.New(
		context.Background(),
		config.CharacterName_Bucket,
		cess.ConnectRpcAddrs(n.GetRpcAddr()),
		cess.Mnemonic(n.GetMnemonic()),
		cess.TransactionTimeout(configs.TimeToWaitEvent),
	)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}

	//Query your own information on the chain
	minerInfo, err := n.QueryStorageMiner(n.GetStakingPublickey())
	if err != nil {
		if err.Error() != pattern.ERR_Empty {
			out.Err(pattern.ERR_RPC_CONNECTION.Error())
		} else {
			out.Err("You are not a storage node")
		}
		os.Exit(1)
	}

	minerInfo.Collaterals.Div(new(big.Int).SetBytes(minerInfo.Collaterals.Bytes()), big.NewInt(1000000000000))

	beneficiaryAcc, _ := sutils.EncodePublicKeyAsCessAccount(minerInfo.BeneficiaryAcc[:])

	name := n.GetSdkName()
	if strings.Contains(name, "bucket") {
		name = "storage miner"
	}

	var tableRows = []table.Row{
		{"name", name},
		{"peer id", base58.Encode([]byte(string(minerInfo.PeerId[:])))},
		{"state", string(minerInfo.State)},
		{"staking amount", fmt.Sprintf("%v %s", minerInfo.Collaterals, n.GetTokenSymbol())},
		{"validated space", fmt.Sprintf("%s", unitConversion(minerInfo.IdleSpace))},
		{"used space", fmt.Sprintf("%s", unitConversion(minerInfo.ServiceSpace))},
		{"locked space", fmt.Sprintf("%s", unitConversion(minerInfo.LockSpace))},
		{"staking account", n.GetStakingAcc()},
		{"earnings account", beneficiaryAcc},
	}
	tw := table.NewWriter()
	tw.AppendRows(tableRows)
	fmt.Println(tw.Render())
	os.Exit(0)
}

func unitConversion(value types.U128) string {
	var result string
	if value.IsUint64() {
		v := value.Uint64()
		if v >= (pattern.SIZE_1GiB * 1024 * 1024 * 1024) {
			result = fmt.Sprintf("%.2f EiB", float64(float64(v)/float64(pattern.SIZE_1GiB*1024*1024*1024)))
			return result
		}
		if v >= (pattern.SIZE_1GiB * 1024 * 1024) {
			result = fmt.Sprintf("%.2f PiB", float64(float64(v)/float64(pattern.SIZE_1GiB*1024*1024)))
			return result
		}
		if v >= (pattern.SIZE_1GiB * 1024) {
			result = fmt.Sprintf("%.2f TiB", float64(float64(v)/float64(pattern.SIZE_1GiB*1024)))
			return result
		}
		if v >= (pattern.SIZE_1GiB) {
			result = fmt.Sprintf("%.2f GiB", float64(float64(v)/float64(pattern.SIZE_1GiB)))
			return result
		}
		if v >= (pattern.SIZE_1MiB) {
			result = fmt.Sprintf("%.2f MiB", float64(float64(v)/float64(pattern.SIZE_1MiB)))
			return result
		}
		if v >= (pattern.SIZE_1KiB) {
			result = fmt.Sprintf("%.2f KiB", float64(float64(v)/float64(pattern.SIZE_1KiB)))
			return result
		}
		result = fmt.Sprintf("%v Bytes", v)
		return result
	}
	v := new(big.Int).SetBytes(value.Bytes())
	v.Quo(v, new(big.Int).SetUint64((pattern.SIZE_1GiB * 1024 * 1024 * 1024)))
	result = fmt.Sprintf("%v EiB", v)
	return result
}
