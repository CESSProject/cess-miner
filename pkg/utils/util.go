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
	"io/fs"
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

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

type MountPathInfo struct {
	Path  string
	Total uint64
	Free  uint64
}

type FileDetail struct {
	Path    string
	ModTime string
	Size    int64
	IsDir   bool
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
	result, err := filepath.Glob(filepath.Join(path, "*"))
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
	result, err := filepath.Glob(filepath.Join(path, "*"))
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

// Get the total size of all files in a directory and subdirectories
func DirFileDetail(path string, count uint32) ([]FileDetail, error) {
	var fileDetails = make([]FileDetail, 0)
	var fstat fs.FileInfo
	result, err := filepath.Glob(filepath.Join(path, "*"))
	if err != nil {
		return nil, err
	}
	for _, v := range result {
		fstat, err = os.Stat(v)
		if err != nil {
			continue
		}

		fileDetails = append(fileDetails, FileDetail{
			Path:    v,
			ModTime: fstat.ModTime().Local().Format(time.DateTime),
			Size:    fstat.Size(),
			IsDir:   fstat.IsDir(),
		})

		if count > 0 {
			if len(fileDetails) >= int(count) {
				break
			}
		}
	}
	return fileDetails, nil
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
	for i := length - 1; i >= 0; i-- {
		j := rand.New(rand.NewSource(time.Now().Unix())).Intn(length)
		swap(i, j)
	}
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

var GlobalTransport = &http.Transport{
	DisableKeepAlives: true,
}

var ipRegex = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)

func ContainsIpv4(str string) bool {
	matches := ipRegex.FindString(str)
	ipAddr := net.ParseIP(matches)
	return ipAddr != nil && strings.Contains(matches, ".")
}

func FreeLocalPort(port uint16) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second*3)
	if err != nil {
		return true
	}
	conn.Close()
	return false
}
