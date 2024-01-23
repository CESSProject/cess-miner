/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TagFileType struct {
	Tag  *pb.Tag `protobuf:"bytes,1,opt,name=tag,proto3" json:"tag,omitempty"`
	USig []byte  `protobuf:"bytes,2,opt,name=u_sig,json=uSig,proto3" json:"u_sig,omitempty"`
}

func (n *Node) serviceTag(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	chainSt := n.GetChainState()
	if !chainSt {
		return
	}

	minerSt := n.GetMinerState()
	if minerSt != pattern.MINER_STATE_POSITIVE &&
		minerSt != pattern.MINER_STATE_FROZEN {
		return
	}

	var ok bool
	var recover bool
	var onChainFlag bool
	var blocknumber uint32
	var txhash string
	var fid string
	var fragmentHash string
	var requestGenTag *pb.RequestGenTag
	var dialOptions []grpc.DialOption
	var teeSign pattern.TeeSignature
	var tagSigInfo pattern.TagSigInfo

	n.SetCalcTagFlag(true)
	defer n.SetCalcTagFlag(false)

	roothashs, err := utils.Dirs(n.GetDirs().FileDir)
	if err != nil {
		n.Stag("err", fmt.Sprintf("[Dirs(%s)] %v", n.GetDirs().FileDir, err))
		return
	}

	teeEndPoints := n.GetPriorityTeeList()
	teeEndPoints = append(teeEndPoints, n.GetAllMarkerTeeEndpoint()...)

	for _, fileDir := range roothashs {
		fid = filepath.Base(fileDir)
		ok, err = n.Has([]byte(Cach_prefix_File + fid))
		if err == nil {
			if !ok {
				continue
			}
		} else {
			n.Report("err", err.Error())
			time.Sleep(time.Second)
			continue
		}

		files, err := utils.DirFiles(fileDir, 0)
		if err != nil {
			n.Stag("err", fmt.Sprintf("[DirFiles(%s)] %v", fid, err))
			time.Sleep(time.Second)
			continue
		}

		for _, f := range files {
			if strings.Contains(f, ".tag") {
				continue
			}
			recover = false
			fragmentHash = filepath.Base(f)
			_, err = os.Stat(f + ".tag")
			if err == nil {
				ok, _ = n.Has([]byte(Cach_prefix_Tag + fid + "." + fragmentHash))
				if !ok {
					fmeta, err := n.QueryFileMetadata(fid)
					if err == nil {
						for _, segment := range fmeta.SegmentList {
							for _, fragment := range segment.FragmentList {
								if sutils.CompareSlice(fragment.Miner[:], n.GetSignatureAccPulickey()) {
									if fragment.Tag.HasValue() {
										ok, block := fragment.Tag.Unwrap()
										if ok {
											err = n.Put([]byte(Cach_prefix_Tag+fid+"."+fragmentHash), []byte(fmt.Sprintf("%d", block)))
											if err != nil {
												n.Stag("err", fmt.Sprintf("[Cache.Put(%s, %s)] %v", Cach_prefix_Tag+fid+"."+fragmentHash, fmt.Sprintf("%d", block), err))
											} else {
												n.Stag("info", fmt.Sprintf("[Cache.Put(%s, %s)]", Cach_prefix_Tag+fid+"."+fragmentHash, fmt.Sprintf("%d", block)))
											}
											break
										}
									} else {
										os.Remove(f + ".tag")
									}
								}
							}
						}
					}
				}
				n.Stag("info", fmt.Sprintf("[Cache.Has(%s)", Cach_prefix_Tag+fid+"."+fragmentHash))
				continue
			}

			buf, err := os.ReadFile(f)
			if err != nil {
				if strings.Contains(err.Error(), "no such file") {
					recover = true
					n.Stag("err", fmt.Sprintf("[%s] Missing a file segment: %s", fid, fragmentHash))
				} else {
					n.Stag("err", fmt.Sprintf("[ReadFile(%s.%s)]: %v", fid, fragmentHash, err))
					continue
				}
			} else {
				if len(buf) != pattern.FragmentSize {
					recover = true
					os.Remove(f)
					n.Stag("err", fmt.Sprintf("[%s.%s] File fragment size [%d] is not equal to %d", fid, fragmentHash, len(buf), pattern.FragmentSize))
				}
			}

			if recover {
				buf, err = n.GetFragmentFromOss(fragmentHash)
				if err != nil {
					n.Stag("err", fmt.Sprintf("Recovering fragment from cess gateway failed: %v", err))
					continue
				}
				if len(buf) < pattern.FragmentSize {
					n.Stag("err", fmt.Sprintf("[%s.%s] Fragment size [%d] received from CESS gateway is wrong", fid, fragmentHash, len(buf)))
					continue
				}
				err = os.WriteFile(f, buf, os.ModePerm)
				if err != nil {
					n.Stag("err", fmt.Sprintf("[%s] [WriteFile(%s)]: %v", fid, fragmentHash, err))
					continue
				}
			}
			requestGenTag = &pb.RequestGenTag{
				FragmentData: buf[:pattern.FragmentSize],
				FragmentName: fragmentHash,
				CustomData:   "",
				FileName:     fid,
				MinerId:      n.GetSignatureAccPulickey(),
			}
			for i := 0; i < len(teeEndPoints); i++ {
				onChainFlag = false
				teeAcc, err := n.GetTeeWorkAccount(teeEndPoints[i])
				if err != nil {
					n.Stag("err", fmt.Sprintf("[GetTeeWorkAccount(%s)] %v", teeEndPoints[i], err))
					continue
				}
				teeAccountID, err := sutils.ParsingPublickey(teeAcc)
				if err != nil {
					n.Stag("err", fmt.Sprintf("[ParsingPublickey] err: %s", err))
					continue
				}
				n.Stag("info", fmt.Sprintf("[%s] Will calc file tag: %v", fid, fragmentHash))
				n.Stag("info", fmt.Sprintf("[%s] Will use tee: %v", fid, teeEndPoints[i]))
				if !strings.Contains(teeEndPoints[i], "443") {
					dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
				} else {
					dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
				}
				genTag, err := n.RequestGenTag(
					teeEndPoints[i],
					requestGenTag,
					time.Duration(time.Minute*20),
					dialOptions,
					nil,
				)
				if err != nil {
					n.Stag("err", fmt.Sprintf("[RequestGenTag] %v", err))
					continue
				}

				if len(genTag.USig) != pattern.TeeSignatureLen {
					n.Stag("err", fmt.Sprintf("[RequestGenTag] invalid USig length: %d", len(genTag.USig)))
					continue
				}

				if len(genTag.TagSigInfo) != pattern.TeeSignatureLen {
					n.Stag("err", fmt.Sprintf("[RequestGenTag] invalid TagSigInfo length: %d", len(genTag.TagSigInfo)))
					continue
				}
				for k := 0; k < pattern.TeeSignatureLen; k++ {
					teeSign[k] = types.U8(genTag.TagSigInfo[k])
				}

				var tfile = &TagFileType{
					Tag:  genTag.Tag,
					USig: genTag.USig,
				}
				buf, err = json.Marshal(tfile)
				if err != nil {
					n.Stag("err", fmt.Sprintf("[json.Marshal] err: %s", err))
					continue
				}
				ok, err := n.GetPodr2Key().VerifyAttest(genTag.Tag.T.Name, genTag.Tag.T.U, genTag.Tag.PhiHash, genTag.Tag.Attest, "")
				if err != nil {
					n.Stag("err", fmt.Sprintf("[VerifyAttest] err: %s", err))
					continue
				}
				if !ok {
					n.Stag("err", "VerifyAttest is false")
					continue
				}
				err = sutils.WriteBufToFile(buf, fmt.Sprintf("%s.tag", f))
				if err != nil {
					n.Stag("err", fmt.Sprintf("[WriteBufToFile] err: %s", err))
					continue
				}

				n.Stag("info", fmt.Sprintf("Calc a service tag: %s", fmt.Sprintf("%s.tag", f)))

				for j := 0; j < pattern.FileHashLen; j++ {
					tagSigInfo.Filehash[j] = types.U8(fid[j])
				}
				tagSigInfo.Miner = types.AccountID(n.GetSignatureAccPulickey())
				tagSigInfo.TeeAcc = types.AccountID(teeAccountID)
				n.Stag("info", fmt.Sprintf("Will report tag: %s.%s", fid, fragmentHash))
				for j := 0; j < 10; j++ {
					txhash, err = n.ReportTagCalculated(teeSign, tagSigInfo)
					if err != nil || txhash == "" {
						n.Stag("err", fmt.Sprintf("ReportTagCalculated[%s.%s]: [%s] %v", fid, fragmentHash, txhash, err))
						time.Sleep(pattern.BlockInterval)
						fmeta, err := n.QueryFileMetadata(fid)
						if err == nil {
							for _, segment := range fmeta.SegmentList {
								for _, fragment := range segment.FragmentList {
									if sutils.CompareSlice(fragment.Miner[:], n.GetSignatureAccPulickey()) {
										if fragment.Tag.HasValue() {
											ok, block := fragment.Tag.Unwrap()
											if ok {
												onChainFlag = true
												err = n.Put([]byte(Cach_prefix_Tag+fid+"."+fragmentHash), []byte(fmt.Sprintf("%d", block)))
												if err != nil {
													n.Stag("err", fmt.Sprintf("[Cache.Put(%s, %s)] %v", Cach_prefix_Tag+fid+"."+fragmentHash, fmt.Sprintf("%d", block), err))
												} else {
													n.Stag("info", fmt.Sprintf("[Cache.Put(%s, %s)]", Cach_prefix_Tag+fid+"."+fragmentHash, fmt.Sprintf("%d", block)))
												}
												break
											}
										}
									}
								}
								if onChainFlag {
									break
								}
							}
						}
						if onChainFlag {
							break
						}
						n.Stag("err", err.Error())
						if (j + 1) >= 10 {
							os.Remove(fmt.Sprintf("%s.tag", f))
							break
						}
						time.Sleep(time.Minute)
						continue
					}
					onChainFlag = true
					n.Stag("info", fmt.Sprintf("ReportTagCalculated[%s.%s]: [%s]", fid, fragmentHash, txhash))
					blocknumber, err = n.QueryBlockHeight(txhash)
					if err != nil {
						n.Stag("err", fmt.Sprintf("[QueryBlockHeight(%s)] %v", txhash, err))
						break
					}
					err = n.Put([]byte(Cach_prefix_Tag+fid+"."+fragmentHash), []byte(fmt.Sprintf("%d", blocknumber)))
					if err != nil {
						n.Stag("err", fmt.Sprintf("[Cache.Put(%s, %s)] %v", Cach_prefix_Tag+fid+"."+fragmentHash, fmt.Sprintf("%d", blocknumber), err))
						break
					}
					n.Stag("info", fmt.Sprintf("Cach.Put[%s.%s]: [%d]", fid, fragmentHash, blocknumber))
					break
				}
				if onChainFlag {
					break
				}
			}
		}
	}
}

// func (n *Node) reportTagCalculated(teeSign pattern.TeeSignature, tagSigInfo pattern.TagSigInfo) (uint32, error) {
// 	var err error
// 	var txhash string
// 	var fid = string(tagSigInfo.Filehash[:])
// 	for j := 0; j < 10; j++ {
// 		txhash, err = n.ReportTagCalculated(teeSign, tagSigInfo)
// 		if err != nil || txhash == "" {
// 			//n.Stag("err", fmt.Sprintf("ReportTagCalculated[%s.%s]: [%s] %v", fid, fragmentHash, txhash, err))
// 			time.Sleep(pattern.BlockInterval)
// 			fmeta, err := n.QueryFileMetadata(fid)
// 			if err != nil{

// 			}

// 			if err == nil {
// 				for _, segment := range fmeta.SegmentList {
// 					for _, fragment := range segment.FragmentList {
// 						if sutils.CompareSlice(fragment.Miner[:], n.GetSignatureAccPulickey()) {
// 							if fragment.Tag.HasValue() {
// 								ok, block := fragment.Tag.Unwrap()
// 								if ok {
// 									onChainFlag = true
// 									err = n.Put([]byte(Cach_prefix_Tag+fid+"."+fragmentHash), []byte(fmt.Sprintf("%d", block)))
// 									if err != nil {
// 										n.Stag("err", fmt.Sprintf("[Cache.Put(%s, %s)] %v", Cach_prefix_Tag+fid+"."+fragmentHash, fmt.Sprintf("%d", block), err))
// 									} else {
// 										n.Stag("info", fmt.Sprintf("[Cache.Put(%s, %s)]", Cach_prefix_Tag+fid+"."+fragmentHash, fmt.Sprintf("%d", block)))
// 									}
// 									break
// 								}
// 							}
// 						}
// 					}
// 					if onChainFlag {
// 						break
// 					}
// 				}
// 			}
// 			if onChainFlag {
// 				break
// 			}
// 			n.Stag("err", err.Error())
// 			if (j + 1) >= 10 {
// 				os.Remove(fmt.Sprintf("%s.tag", f))
// 				break
// 			}
// 			time.Sleep(time.Minute)
// 			continue
// 		}
// 		onChainFlag = true
// 		n.Stag("info", fmt.Sprintf("ReportTagCalculated[%s.%s]: [%s]", fid, fragmentHash, txhash))
// 		blocknumber, err = n.QueryBlockHeight(txhash)
// 		if err != nil {
// 			n.Stag("err", fmt.Sprintf("[QueryBlockHeight(%s)] %v", txhash, err))
// 			break
// 		}
// 		err = n.Put([]byte(Cach_prefix_Tag+fid+"."+fragmentHash), []byte(fmt.Sprintf("%d", blocknumber)))
// 		if err != nil {
// 			n.Stag("err", fmt.Sprintf("[Cache.Put(%s, %s)] %v", Cach_prefix_Tag+fid+"."+fragmentHash, fmt.Sprintf("%d", blocknumber), err))
// 			break
// 		}
// 		n.Stag("info", fmt.Sprintf("Cach.Put[%s.%s]: [%d]", fid, fragmentHash, blocknumber))
// 	}
// 	return nil
// }
