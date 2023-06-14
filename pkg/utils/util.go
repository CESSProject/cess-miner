/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

type MountPathInfo struct {
	Path  string
	Total uint64
	Free  uint64
}

// Get the total size of all files in a directory and subdirectories
func DirSize(path string) (uint64, error) {
	var size uint64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size += uint64(info.Size())
		}
		return err
	})
	return size, err
}

// Get the total size of all files in a directory and subdirectories
func Dirs(path string) ([]string, error) {
	var dirs = make([]string, 0)
	result, err := filepath.Glob(path + "/*")
	if err != nil {
		return nil, err
	}
	for _, v := range result {
		f, err := os.Stat(v)
		if err != nil {
			continue
		}
		if f.IsDir() {
			dirs = append(dirs, v)
		}
	}
	return dirs, nil
}

// Get the total size of all files in a directory and subdirectories
func DirFiles(path string, count uint32) ([]string, error) {
	var files = make([]string, 0)
	result, err := filepath.Glob(path + "/*")
	if err != nil {
		return nil, err
	}
	for _, v := range result {
		f, err := os.Stat(v)
		if err != nil {
			continue
		}
		if !f.IsDir() {
			files = append(files, v)
		}
		if count > 0 {
			if len(files) >= int(count) {
				break
			}
		}
	}
	return files, nil
}

// Get a random integer in a specified range
func RandomInRange(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

// Calculate the file hash value
func CalcFileHash(fpath string) (string, error) {
	f, err := os.Open(fpath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// func GetMountPathInfo(mountpath string) (MountPathInfo, error) {
// 	var mp MountPathInfo
// 	pss, err := disk.Partitions(false)
// 	if err != nil {
// 		return mp, err
// 	}

// 	for _, ps := range pss {
// 		us, err := disk.Usage(ps.Mountpoint)
// 		if err != nil {
// 			continue
// 		}
// 		if us.Path == mountpath {
// 			mp.Path = us.Path
// 			mp.Free = us.Free
// 			mp.Total = us.Total
// 			return mp, nil
// 		}
// 	}
// 	return mp, errors.New("mount point not found")
// }

func RandSlice(slice interface{}) {
	rv := reflect.ValueOf(slice)
	if rv.Type().Kind() != reflect.Slice {
		return
	}

	length := rv.Len()
	if length < 2 {
		return
	}

	swap := reflect.Swapper(slice)
	rand.Seed(time.Now().Unix())
	for i := length - 1; i >= 0; i-- {
		j := rand.Intn(length)
		swap(i, j)
	}
	return
}

// ----------------------- Random key -----------------------
const (
	letterIdBits = 6
	letterIdMask = 1<<letterIdBits - 1
	letterIdMax  = 63 / letterIdBits
)
const baseStr = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()[]{}+-*/_=."

// Generate random password
func GetRandomcode(length uint8) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))
	bytes := make([]byte, length)
	l := len(baseStr)
	for i := uint8(0); i < length; i++ {
		bytes[i] = baseStr[r.Intn(l)]
	}
	return string(bytes)
}

func RandStr(n int) string {
	src := rand.NewSource(time.Now().UnixNano())
	sb := strings.Builder{}
	sb.Grow(n)
	// A rand.Int63() generates 63 random bits, enough for letterIdMax letters!
	for i, cache, remain := n-1, src.Int63(), letterIdMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdMax
		}
		if idx := int(cache & letterIdMask); idx < len(baseStr) {
			sb.WriteByte(baseStr[idx])
			i--
		}
		cache >>= letterIdBits
		remain--
	}
	return sb.String()
}

func GetISOWeek() string {
	year, week := time.Now().UTC().ISOWeek()
	return fmt.Sprintf("%d%d", year, week)
}

func OpenedPort(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func Ternary(a, b int64) int64 {
	if a > b {
		return b
	}
	return a
}

func ParseMultiaddrs(domain string) ([]string, error) {
	var result = make([]string, 0)
	var realDns = make([]string, 0)

	addr, err := ma.NewMultiaddr(domain)
	if err == nil {
		_, err = peer.AddrInfoFromP2pAddr(addr)
		if err == nil {
			result = append(result, domain)
			return result, nil
		}
	}

	dnsnames, err := net.LookupTXT(domain)
	if err != nil {
		return result, err
	}

	for _, v := range dnsnames {
		if strings.Contains(v, "ip4") && strings.Contains(v, "tcp") && strings.Count(v, "=") == 1 {
			result = append(result, strings.TrimPrefix(v, "dnsaddr="))
		}
	}

	trims := strings.Split(domain, ".")
	domainname := fmt.Sprintf("%s.%s", trims[len(trims)-2], trims[len(trims)-1])

	for _, v := range dnsnames {
		trims = strings.Split(v, "/")
		for _, vv := range trims {
			if strings.Contains(vv, domainname) {
				realDns = append(realDns, vv)
				break
			}
		}
	}

	for _, v := range realDns {
		dnses, err := net.LookupTXT("_dnsaddr." + v)
		if err != nil {
			continue
		}
		for i := 0; i < len(dnses); i++ {
			if strings.Contains(dnses[i], "ip4") && strings.Contains(dnses[i], "tcp") && strings.Count(dnses[i], "=") == 1 {
				var multiaddr = strings.TrimPrefix(dnses[i], "dnsaddr=")
				result = append(result, multiaddr)
			}
		}
	}

	return result, nil
}

func CopyFile(dst, src string) error {
	fsrc, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fsrc.Close()

	fdst, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer fsrc.Close()

	_, err = io.Copy(fdst, fsrc)
	return err
}
