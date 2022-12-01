package tools

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/disk"
)

type MountPathInfo struct {
	Path  string
	Total uint64
	Free  uint64
}

// Write string content to file
func WriteStringtoFile(content, fileName string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	if err != nil {
		return err
	}
	return nil
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

// Get a random integer in a specified range
func RandomInRange(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

// Create a directory
func CreatDirIfNotExist(dir string) error {
	_, err := os.Stat(dir)
	if err != nil {
		return os.MkdirAll(dir, os.ModeDir)
	}
	return nil
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

func RecoverError(err interface{}) string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%v\n", "--------------------panic--------------------")
	fmt.Fprintf(buf, "%v\n", err)
	fmt.Fprintf(buf, "%v\n", string(debug.Stack()))
	return buf.String()
}

func IsIPv4(ipAddr string) bool {
	ip := net.ParseIP(ipAddr)
	return ip != nil && strings.Contains(ipAddr, ".")
}

func GetMountPathInfo(mountpath string) (MountPathInfo, error) {
	var mp MountPathInfo
	pss, err := disk.Partitions(false)
	if err != nil {
		return mp, err
	}

	for _, ps := range pss {
		us, err := disk.Usage(ps.Mountpoint)
		if err != nil {
			continue
		}
		if us.Path == mountpath {
			mp.Path = us.Path
			mp.Free = us.Free
			mp.Total = us.Total
			return mp, nil
		}
	}
	return mp, errors.New("mount point not found")
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

// ----------------------- Random key -----------------------
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

// ClearMemBuf is used to clear membuf
func ClearMemBuf() {
	exec.Command("bash", "-c", "sync;sync;sync;sync;sync;sync;")
	exec.Command("bash", "-c", "echo 1 > /proc/sys/vm/drop_caches")
	exec.Command("bash", "-c", "echo 2 > /proc/sys/vm/drop_caches")
	exec.Command("bash", "-c", "echo 3 > /proc/sys/vm/drop_caches")
}
