package console

import (
	"os"

	"github.com/spf13/cobra"
)

// Query your own details on-chain
func Command_State_Runfunc(cmd *cobra.Command, args []string) {
	// //Parse command arguments and  configuration file
	// parseFlags(cmd)

	// api, err := chain.NewRpcClient(configs.C.RpcAddr)
	// if err != nil {
	// 	fmt.Printf("\x1b[%dm[err]\x1b[0m Connection error: %v\n", 41, err)
	// 	os.Exit(1)
	// }
	// //Query your own information on the chain
	// mData, err := chain.GetMinerInfo(api)
	// if err != nil {
	// 	if err.Error() == chain.ERR_Empty {
	// 		log.Printf("[err] Not found: %v\n", err)
	// 		os.Exit(1)
	// 	}
	// 	log.Printf("[err] Query error: %v\n", err)
	// 	os.Exit(1)
	// }
	// mData.Collaterals.Div(new(big.Int).SetBytes(mData.Collaterals.Bytes()), big.NewInt(1000000000000))
	// addr := fmt.Sprintf("%d.%d.%d.%d:%d", mData.Ip.Value[0], mData.Ip.Value[1], mData.Ip.Value[2], mData.Ip.Value[3], mData.Ip.Port)
	// var power, space float32
	// var power_unit, space_unit string
	// count := 0
	// for mData.Power.BitLen() > int(16) {
	// 	mData.Power.Div(new(big.Int).SetBytes(mData.Power.Bytes()), big.NewInt(1024))
	// 	count++
	// }
	// if mData.Power.Int64() > 1024 {
	// 	power = float32(mData.Power.Int64()) / float32(1024)
	// 	count++
	// } else {
	// 	power = float32(mData.Power.Int64())
	// }
	// switch count {
	// case 0:
	// 	power_unit = "Byte"
	// case 1:
	// 	power_unit = "KiB"
	// case 2:
	// 	power_unit = "MiB"
	// case 3:
	// 	power_unit = "GiB"
	// case 4:
	// 	power_unit = "TiB"
	// case 5:
	// 	power_unit = "PiB"
	// case 6:
	// 	power_unit = "EiB"
	// case 7:
	// 	power_unit = "ZiB"
	// case 8:
	// 	power_unit = "YiB"
	// case 9:
	// 	power_unit = "NiB"
	// case 10:
	// 	power_unit = "DiB"
	// default:
	// 	power_unit = fmt.Sprintf("DiB(%v)", count-10)
	// }
	// count = 0
	// for mData.Space.BitLen() > int(16) {
	// 	mData.Space.Div(new(big.Int).SetBytes(mData.Space.Bytes()), big.NewInt(1024))
	// 	count++
	// }
	// if mData.Space.Int64() > 1024 {
	// 	space = float32(mData.Space.Int64()) / float32(1024)
	// 	count++
	// } else {
	// 	space = float32(mData.Space.Int64())
	// }

	// switch count {
	// case 0:
	// 	space_unit = "Byte"
	// case 1:
	// 	space_unit = "KiB"
	// case 2:
	// 	space_unit = "MiB"
	// case 3:
	// 	space_unit = "GiB"
	// case 4:
	// 	space_unit = "TiB"
	// case 5:
	// 	space_unit = "PiB"
	// case 6:
	// 	space_unit = "EiB"
	// case 7:
	// 	space_unit = "ZiB"
	// case 8:
	// 	space_unit = "YiB"
	// case 9:
	// 	space_unit = "NiB"
	// case 10:
	// 	space_unit = "DiB"
	// default:
	// 	power_unit = fmt.Sprintf("DiB(%v)", count-10)
	// }

	// //print your own details
	// fmt.Printf("MinerId: C%v\nState: %v\nStorageSpace: %.2f %v\nUsedSpace: %.2f %v\nPledgeTokens: %v TCESS\nServiceAddr: %v\n",
	// 	mData.PeerId, string(mData.State), power, power_unit, space, space_unit, mData.Collaterals, string(addr))
	os.Exit(0)
}
