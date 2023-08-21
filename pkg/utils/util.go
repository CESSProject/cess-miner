/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
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

func GetDirFreeSpace(dir string) (uint64, error) {
	sageStat, err := disk.Usage(dir)
	return sageStat.Free, err
}

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

func RecoverError(err interface{}) string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%v\n", "--------------------panic--------------------")
	fmt.Fprintf(buf, "%v\n", err)
	fmt.Fprintf(buf, "%v\n", string(debug.Stack()))
	return buf.String()
}

func GetSysMemAvailable() (uint64, error) {
	var result uint64
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return 0, errors.Wrapf(err, "[mem.VirtualMemory]")
	}
	result = memInfo.Available
	swapInfo, err := mem.SwapMemory()
	if err != nil {
		return result, nil
	}
	return result + swapInfo.Free, nil
}

func GetSysMemTotle() (uint64, error) {
	var result uint64
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return 0, errors.Wrapf(err, "[mem.VirtualMemory]")
	}
	result = memInfo.Total
	swapInfo, err := mem.SwapMemory()
	if err != nil {
		return result, nil
	}
	return result + swapInfo.Free, nil
}

var globalTransport = &http.Transport{
	DisableKeepAlives: true,
}

func QueryPeers(url string) ([]byte, error) {
	if url == "" {
		return nil, errors.New("invalid url")
	}

	if url[len(url)-1] != byte(47) {
		url += "/"
	}

	req, err := http.NewRequest(http.MethodGet, url+"peers", nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	client.Transport = globalTransport
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed")
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

var regstr = `\d+\.\d+\.\d+\.\d+`
var reg = regexp.MustCompile(regstr)

func FildIpv4(data []byte) (string, bool) {
	result := reg.Find(data)
	return string(result), len(result) > 0
}

func IsIntranetIpv4(ipv4 string) (bool, error) {
	ip := net.ParseIP(ipv4)
	if ip == nil || !strings.Contains(ipv4, ".") {
		return false, errors.New("invalid ipv4")
	}
	if ip.IsLoopback() {
		return true, nil
	}
	if ip.IsPrivate() {
		return true, nil
	}
	return false, nil
}

func RemoveRepeatedAddr(arr []multiaddr.Multiaddr) (newArr []multiaddr.Multiaddr) {
	newArr = make([]multiaddr.Multiaddr, 0)
	for i := 0; i < len(arr); i++ {
		repeat := false
		for j := i + 1; j < len(arr); j++ {
			if arr[i].Equal(arr[j]) {
				repeat = true
				break
			}
		}
		if !repeat {
			newArr = append(newArr, arr[i])
		}
	}
	return newArr
}

// InterfaceIsNIL returns the comparison between i and nil
func InterfaceIsNIL(i interface{}) bool {
	ret := i == nil
	if !ret {
		defer func() {
			recover()
		}()
		va := reflect.ValueOf(i)
		if va.Kind() == reflect.Ptr {
			return va.IsNil()
		}
		return false
	}
	return ret
}
