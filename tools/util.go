package tools

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"time"
	"unsafe"

	"github.com/pkg/errors"
)

// Convert the ip address of integer type to string type
func InetNtoA(ip int64) string {
	return fmt.Sprintf("%d.%d.%d.%d", byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
}

// Convert string type ip address to integer type
func InetAtoN(ip string) (int64, error) {
	ret := big.NewInt(0)
	result := net.ParseIP(ip)
	if result == nil {
		return 0, errors.New("invalid ip")
	}
	return ret.SetBytes(result.To4()).Int64(), nil
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

// Integer to bytes
func IntegerToBytes(n interface{}) ([]byte, error) {
	bytesBuffer := bytes.NewBuffer([]byte{})
	t := reflect.TypeOf(n)
	switch t.Kind() {
	case reflect.Int16:
		binary.Write(bytesBuffer, binary.LittleEndian, n)
		return bytesBuffer.Bytes(), nil
	case reflect.Uint16:
		binary.Write(bytesBuffer, binary.LittleEndian, n)
		return bytesBuffer.Bytes(), nil
	case reflect.Int:
		binary.Write(bytesBuffer, binary.LittleEndian, n)
		return bytesBuffer.Bytes(), nil
	case reflect.Uint:
		binary.Write(bytesBuffer, binary.LittleEndian, n)
		return bytesBuffer.Bytes(), nil
	case reflect.Int32:
		binary.Write(bytesBuffer, binary.LittleEndian, n)
		return bytesBuffer.Bytes(), nil
	case reflect.Uint32:
		binary.Write(bytesBuffer, binary.LittleEndian, n)
		return bytesBuffer.Bytes(), nil
	case reflect.Int64:
		binary.Write(bytesBuffer, binary.LittleEndian, n)
		return bytesBuffer.Bytes(), nil
	case reflect.Uint64:
		binary.Write(bytesBuffer, binary.LittleEndian, n)
		return bytesBuffer.Bytes(), nil
	default:
		return nil, errors.New("unsupported type")
	}
}

// Get the total size of all files in a directory and subdirectories
func DirSize(path string) (uint64, error) {
	var size uint64
	err := filepath.Walk(path, func(s string, info os.FileInfo, err error) error {
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

// Base58 encoded characters
var base58 = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

// Base58 encode
func Base58Encoding(str string) string {
	strByte := []byte(str)
	strTen := big.NewInt(0).SetBytes(strByte)
	var modSlice []byte
	for strTen.Cmp(big.NewInt(0)) > 0 {
		mod := big.NewInt(0)
		strTen58 := big.NewInt(58)
		strTen.DivMod(strTen, strTen58, mod)
		modSlice = append(modSlice, base58[mod.Int64()])
	}

	for _, elem := range strByte {
		if elem != 0 {
			break
		} else if elem == 0 {
			modSlice = append(modSlice, byte('1'))
		}
	}
	ReverseModSlice := ReverseByteArr(modSlice)
	return string(ReverseModSlice)
}

func ReverseByteArr(bytes []byte) []byte {
	for i := 0; i < len(bytes)/2; i++ {
		bytes[i], bytes[len(bytes)-1-i] = bytes[len(bytes)-1-i], bytes[i]
	}
	return bytes
}

// Base58 Decode
func Base58Decoding(str string) string {
	strByte := []byte(str)
	ret := big.NewInt(0)
	for _, byteElem := range strByte {
		index := bytes.IndexByte(base58, byteElem)
		ret.Mul(ret, big.NewInt(58))
		ret.Add(ret, big.NewInt(int64(index)))
	}
	return string(ret.Bytes())
}

// bytes to string
func B2S(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// string to bytes
func S2B(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}

// Create a directory
func CreatDirIfNotExist(dir string) error {
	_, err := os.Stat(dir)
	if err != nil {
		return os.MkdirAll(dir, os.ModeDir)
	}
	return nil
}

// Get the name of a first-level subdirectory in a given directory
func WalkDir(filePath string) ([]string, error) {
	dirs := make([]string, 0)
	files, err := ioutil.ReadDir(filePath)
	if err != nil {
		return dirs, err
	} else {
		for _, v := range files {
			if v.IsDir() {
				dirs = append(dirs, v.Name())
			}
		}
	}
	return dirs, nil
}

// Send a post request to the specified address
func Post(url string, para interface{}) ([]byte, error) {
	body, err := json.Marshal(para)
	if err != nil {
		return nil, err
	}
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	var resp = new(http.Response)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp != nil {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return respBody, err
	}
	return nil, err
}
