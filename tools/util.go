package tools

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
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
	case reflect.Int8:
		binary.Write(bytesBuffer, binary.LittleEndian, n)
		return bytesBuffer.Bytes(), nil
	case reflect.Uint8:
		binary.Write(bytesBuffer, binary.LittleEndian, n)
		return bytesBuffer.Bytes(), nil
	default:
		return nil, errors.New("unsupported type")
	}
}

// Bytes to Integer
func BytesToInteger(n []byte) (int32, error) {
	var x int32
	bytesBuffer := bytes.NewBuffer(n)
	err := binary.Read(bytesBuffer, binary.LittleEndian, &x)
	return x, err
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

// Get external network ip
func GetExternalIp() (string, error) {
	output, err := exec.Command("bash", "-c", "curl ifconfig.co").Output()
	// output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	result := strings.Replace(string(output), "\n", "", -1)
	return strings.Replace(result, " ", "", -1), nil
}

//
func Split(file *os.File, s int64) (M [][]byte, S int64, N uint64, err error) {
	file.Seek(0, 0)

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, 0, 0, err
	}
	size := fileInfo.Size()
	n := uint64(math.Ceil(float64(size / s)))
	if n == 0 {
		n = 1
	}
	// matrix is indexed as m_ij, so the first dimension has n items and the second has s.
	matrix := make([][]byte, n)
	for i := uint64(0); i < n; i++ {
		piece := make([]byte, s)
		_, err := file.Read(piece)
		if err != nil {
			return nil, 0, 0, err
		}
		matrix[i] = piece
	}
	return matrix, s, n, nil
}

func CalcHash(data []byte) (string, error) {
	if len(data) <= 0 {
		return "", errors.New("data is nil")
	}
	h := sha256.New()
	_, err := h.Write(data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func GetStringWithoutNumbers(in string) string {
	var resu string
	resu = RemoveX(in, strconv.Itoa(0))
	for i := 1; i < 10; i++ {
		resu = RemoveX(resu, strconv.Itoa(i))
	}
	return resu
}

func RemoveX(str string, x string) string {
	var res string
	for i := 0; i < len(str); i++ {
		if string(str[i]) != x {
			res = res + string(str[i])
		}
	}
	return res
}
