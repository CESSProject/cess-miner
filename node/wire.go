/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/CESSProject/cess-miner/configs"
	"github.com/CESSProject/cess-miner/node/web"
	"github.com/CESSProject/cess-miner/pkg/utils"
	"github.com/CESSProject/p2p-go/out"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/pkg/errors"

	sdkgo "github.com/CESSProject/cess-go-sdk"
	"github.com/CESSProject/cess-go-sdk/chain"
	sconfig "github.com/CESSProject/cess-go-sdk/config"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
)

func InitNode() *Node {
	wire.Build(
		GetNode,
		InitWebServer,
		InitMiddlewares,
		web.NewFileHandler,
		InitChainClient,
	)
	return &Node{}
}

func InitChainClient(n *Node, sip string) {
	cli, err := sdkgo.New(
		context.Background(),
		sdkgo.Name(configs.Name),
		sdkgo.ConnectRpcAddrs(n.ReadRpcEndpoints()),
		sdkgo.Mnemonic(n.ReadMnemonic()),
		sdkgo.TransactionTimeout(configs.TimeToWaitEvent),
	)
	if err != nil {
		out.Err(fmt.Sprintf("[sdkgo.New] %v", err))
		os.Exit(1)
	}
	defer cli.Close()

	err = cli.InitExtrinsicsNameForMiner()
	if err != nil {
		out.Err("The rpc address does not match the software version, please check the rpc address.")
		os.Exit(1)
	}

	err = checkRpcSynchronization(cli)
	if err != nil {
		out.Err("Failed to sync block: network error")
		os.Exit(1)
	}

	err = checkVersion(cli)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}

	n.ExpendersInfo, err = cli.QueryExpenders(-1)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}

	err = checkMiner(cli, sip)
	if err != nil {
		out.Err(err.Error())
		os.Exit(1)
	}

	n.ChainClient = cli
}

func checkMiner(cli *chain.ChainClient, sip string) error {
	register, decTib, oldRegInfo, err := checkRegistrationInfo(cli, n.ReadSignatureAccount(), n.ReadStakingAcc(), n.ReadUseSpace())
	if err != nil {
		return errors.Wrap(err, "[checkMiner]")
	}
	var p *node.Pois
	var rsakey *node.RSAKeyPair
	var minerPoisInfo = &pb.MinerPoisInfo{}
	switch register {
	case Unregistered:
		_, err = registerMiner(cli, n.ReadStakingAcc(), n.ReadEarningsAcc(), sip, decTib)
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}

		time.Sleep(chain.BlockInterval * 10)

		for i := 0; i < 3; i++ {
			rsakey, err = registerPoisKey(cli, peernode, teeRecord, minerPoisInfo, wspace, cfg.ReadPriorityTeeList())
			if err != nil {
				if !strings.Contains(err.Error(), "storage miner is not registered") {
					out.Err(err.Error())
					os.Exit(1)
				}
				time.Sleep(chain.BlockInterval)
				continue
			}
			break
		}
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}

		p, err = node.NewPOIS(
			wspace.GetPoisDir(),
			wspace.GetSpaceDir(),
			wspace.GetPoisAccDir(),
			expender,
			true, 0, 0,
			int64(cfg.ReadUseSpace()*1024), 32,
			runtime.GetCpuCores(),
			minerPoisInfo.KeyN,
			minerPoisInfo.KeyG,
			cli.GetSignatureAccPulickey(),
		)
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}

	case UnregisteredPoisKey:
		err = wspace.Build(peernode.Workspace())
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}
		runtime.SetMinerState(string(oldRegInfo.State))
		for i := 0; i < 3; i++ {
			rsakey, err = registerPoisKey(cli, peernode, teeRecord, minerPoisInfo, wspace, cfg.ReadPriorityTeeList())
			if err != nil {
				if !strings.Contains(err.Error(), "storage miner is not registered") {
					out.Err(err.Error())
					os.Exit(1)
				}
				time.Sleep(chain.BlockInterval)
				continue
			}
			break
		}
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}

		err = updateMinerRegistertionInfo(cli, oldRegInfo, cfg.ReadUseSpace(), cfg.ReadStakingAcc(), cfg.ReadEarningsAcc())
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}

		p, err = node.NewPOIS(
			wspace.GetPoisDir(),
			wspace.GetSpaceDir(),
			wspace.GetPoisAccDir(),
			expender,
			true, 0, 0,
			int64(cfg.ReadUseSpace()*1024), 32,
			runtime.GetCpuCores(),
			minerPoisInfo.KeyN,
			minerPoisInfo.KeyG,
			cli.GetSignatureAccPulickey(),
		)
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}

	case Registered:
		err = wspace.Build(peernode.Workspace())
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}

		runtime.SetMinerState(string(oldRegInfo.State))
		err = updateMinerRegistertionInfo(cli, oldRegInfo, cfg.ReadUseSpace(), cfg.ReadStakingAcc(), cfg.ReadEarningsAcc())
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}
		_, spaceProofInfo := oldRegInfo.SpaceProofInfo.Unwrap()
		minerPoisInfo.Acc = []byte(string(spaceProofInfo.Accumulator[:]))
		minerPoisInfo.Front = int64(spaceProofInfo.Front)
		minerPoisInfo.Rear = int64(spaceProofInfo.Rear)
		minerPoisInfo.KeyN = []byte(string(spaceProofInfo.PoisKey.N[:]))
		minerPoisInfo.KeyG = []byte(string(spaceProofInfo.PoisKey.G[:]))
		minerPoisInfo.StatusTeeSign = []byte(string(oldRegInfo.TeeSig[:]))

		p, err = node.NewPOIS(
			wspace.GetPoisDir(),
			wspace.GetSpaceDir(),
			wspace.GetPoisAccDir(),
			expender, false,
			int64(spaceProofInfo.Front),
			int64(spaceProofInfo.Rear),
			int64(cfg.ReadUseSpace()*1024), 32,
			runtime.GetCpuCores(),
			minerPoisInfo.KeyN,
			minerPoisInfo.KeyG,
			cli.GetSignatureAccPulickey(),
		)
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}

		saveAllTees(cli, peernode, teeRecord)

		buf, err := queryPodr2KeyFromTee(peernode, teeRecord.GetAllTeeEndpoint(), cli.GetSignatureAccPulickey(), cfg.ReadPriorityTeeList())
		if err != nil {
			out.Err(err.Error())
			os.Exit(1)
		}

		rsakey, err = node.NewRsaKey(buf)
		if err != nil {
			out.Err(fmt.Sprintf("Init rsa public key err: %v", err))
			os.Exit(1)
		}

		wspace.SaveRsaPublicKey(buf)
		spaceProofInfo = chain.SpaceProofInfo{}
		buf = nil

	default:
		out.Err("system err")
		os.Exit(1)
	}
}

func registerMiner(cli *chain.ChainClient, stakingAcc, earningsAcc, sip string, decTib uint64) (string, error) {
	if stakingAcc != "" && stakingAcc != cli.GetSignatureAcc() {
		out.Ok(fmt.Sprintf("Specify staking account: %s", stakingAcc))
		txhash, err := cli.RegnstkAssignStaking(earningsAcc, []byte(sip), stakingAcc, uint32(decTib))
		if err != nil {
			if txhash != "" {
				err = fmt.Errorf("[%s] %v", txhash, err)
			}
			return txhash, err
		}
		out.Ok(fmt.Sprintf("Storage miner registration successful: %s", txhash))
		return txhash, nil
	}

	txhash, err := cli.RegnstkSminer(earningsAcc, []byte(sip), uint64(decTib*chain.StakingStakePerTiB), uint32(decTib))
	if err != nil {
		if txhash != "" {
			err = fmt.Errorf("[%s] %v", txhash, err)
		}
		return txhash, err
	}
	out.Ok(fmt.Sprintf("Storage miner registration successful: %s", txhash))
	return txhash, nil
}

func checkRpcSynchronization(cli *chain.ChainClient) error {
	out.Tip("Waiting to synchronize the main chain...")
	var err error
	var syncSt chain.SysSyncState
	for {
		syncSt, err = cli.SystemSyncState()
		if err != nil {
			return err
		}
		if syncSt.CurrentBlock == syncSt.HighestBlock {
			out.Ok(fmt.Sprintf("Synchronization the main chain completed: %d", syncSt.CurrentBlock))
			break
		}
		out.Tip(fmt.Sprintf("In the synchronization main chain: %d ...", syncSt.CurrentBlock))
		time.Sleep(time.Second * time.Duration(utils.Ternary(int64(syncSt.HighestBlock-syncSt.CurrentBlock)*6, 30)))
	}
	return nil
}

func checkVersion(cli *chain.ChainClient) error {
	chainVersion, err := cli.SystemVersion()
	if err != nil {
		return errors.New("failed to query the chain version: rpc connection is down")
	}
	chain := cli.GetNetworkEnv()
	if strings.Contains(chain, configs.TestNet) {
		tmps := strings.Split(chainVersion, "-")
		for _, v := range tmps {
			if strings.Contains(v, ".") {
				values := strings.Split(v, ".")
				if len(values) == 3 {
					if values[0] != configs.ChainVersionStr[0] || values[1] != configs.ChainVersionStr[1] {
						return fmt.Errorf("chain version number is not v%d.%d, please check your rpc service", configs.ChainVersionInt[0], configs.ChainVersionInt[1])
					}
					versionI, err := strconv.Atoi(values[2])
					if err == nil {
						if versionI < configs.ChainVersionInt[2] {
							return fmt.Errorf("chain version number is lower than v%d.%d.%d, please check your rpc service", configs.ChainVersionInt[0], configs.ChainVersionInt[1], configs.ChainVersionInt[2])
						}
					}
				}
			}
		}
	}
	return nil
}

func checkRegistrationInfo(cli *chain.ChainClient, signatureAcc, stakingAcc string, useSpace uint64) (int, uint64, *chain.MinerInfo, error) {
	minerInfo, err := cli.QueryMinerItems(cli.GetSignatureAccPulickey(), -1)
	if err != nil {
		if err.Error() != chain.ERR_Empty {
			return Unregistered, 0, &minerInfo, err
		}
		decTib := useSpace / sconfig.SIZE_1KiB
		if useSpace%sconfig.SIZE_1KiB != 0 {
			decTib += 1
		}
		token := decTib * chain.StakingStakePerTiB
		accInfo, err := cli.QueryAccountInfo(cli.GetSignatureAcc(), -1)
		if err != nil {
			if err.Error() != chain.ERR_Empty {
				return Unregistered, decTib, &minerInfo, fmt.Errorf("failed to query signature account information: %v", err)
			}
			return Unregistered, decTib, &minerInfo, errors.New("signature account does not exist, possible cause: 1.balance is empty 2.wrong rpc address")
		}
		token_cess, _ := new(big.Int).SetString(fmt.Sprintf("%d%s", token, chain.TokenPrecision_CESS), 10)
		if stakingAcc == "" || stakingAcc == signatureAcc {
			if accInfo.Data.Free.CmpAbs(token_cess) < 0 {
				return Unregistered, decTib, &minerInfo, fmt.Errorf("signature account balance less than %d %s", token, cli.GetTokenSymbol())
			}
		} else {
			stakingAccInfo, err := cli.QueryAccountInfo(stakingAcc, -1)
			if err != nil {
				if err.Error() != chain.ERR_Empty {
					return Unregistered, decTib, &minerInfo, fmt.Errorf("failed to query staking account information: %v", err)
				}
				return Unregistered, decTib, &minerInfo, fmt.Errorf("staking account does not exist, possible: 1.balance is empty 2.wrong rpc address")
			}
			if stakingAccInfo.Data.Free.CmpAbs(token_cess) < 0 {
				return Unregistered, decTib, &minerInfo, fmt.Errorf("staking account balance less than %d %s", token, cli.GetTokenSymbol())
			}
		}
		return Unregistered, decTib, &minerInfo, nil
	}
	if !minerInfo.SpaceProofInfo.HasValue() {
		return UnregisteredPoisKey, 0, &minerInfo, nil
	}
	return Registered, 0, &minerInfo, nil
}

func InitWebServer(n *Node, mdls []gin.HandlerFunc, userHdl *web.FileHandler) string {
	n.Engine = gin.Default()
	n.Engine.Use(mdls...)
	userHdl.RegisterRoutes(n.Engine)
	go func() {
		err := n.Engine.Run(fmt.Sprintf(":%d", n.ReadServicePort()))
		if err != nil {
			log.Fatal(err)
		}
	}()
	ip, err := GetLocalIP()
	if err != nil {
		log.Fatal(err)
	}
	ip = fmt.Sprint("http://%s:%d", ip, n.ReadServicePort())
	// TODO
	// sip:=Encrypted(ip)
	return ip
}

func GetLocalIP() (string, error) {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		return "", errors.Wrap(err, "[net.Interfaces]")
	}
	for i := 0; i < len(netInterfaces); i++ {
		if (netInterfaces[i].Flags & net.FlagUp) != 0 {
			addrs, _ := netInterfaces[i].Addrs()
			for _, address := range addrs {
				if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && !ipnet.IP.IsPrivate() && !ipnet.IP.IsUnspecified() {
					// IPv4
					if ipnet.IP.To4() != nil {
						return ipnet.IP.String(), nil
					}
					// IPv6
					if ipnet.IP.To16() != nil {
						return ipnet.IP.String(), nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("No available ip address found")
}

func InitMiddlewares() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		cors.New(cors.Config{
			AllowAllOrigins: true,
			AllowHeaders:    []string{"Content-Type", "Account", "Message", "Signature"},
			AllowMethods:    []string{"POST", "GET", "OPTION"},
		}),
		func(ctx *gin.Context) {
			ok, err := VerifySignature(ctx)
			if !ok || err != nil {
				ctx.AbortWithStatus(http.StatusForbidden)
				return
			}
			ctx.Next()
		},
	}
}

func VerifySignature(ctx *gin.Context) (bool, error) {
	account := ctx.Request.Header.Get("Account")
	message := ctx.Request.Header.Get("Message")
	signature := ctx.Request.Header.Get("Signature")
	return sutils.VerifySR25519WithPublickey(message, []byte(signature), account)
}
