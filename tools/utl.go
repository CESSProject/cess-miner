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
	"runtime"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/pkg/errors"
)

func RunOnLinuxSystem() bool {
	return runtime.GOOS == "linux"
}

func RunWithRootPrivileges() bool {
	return os.Geteuid() == 0
}

func SetAllCores() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func InetNtoA(ip int64) string {
	return fmt.Sprintf("%d.%d.%d.%d", byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
}

func InetAtoN(ip string) (int64, error) {
	ret := big.NewInt(0)
	result := net.ParseIP(ip)
	if result == nil {
		return 0, errors.New("invalid ip")
	}
	return ret.SetBytes(result.To4()).Int64(), nil
}

func WriteStringtoFile(content, fileName string) error {
	var (
		err  error
		name string
		//filesuffix string
		//fileprefix string
	)
	name = fileName
	// _, err = os.Stat(name)
	// if err == nil {
	// 	filesuffix = filepath.Ext(name)
	// 	fileprefix = name[0 : len(name)-len(filesuffix)]
	// 	fileprefix = fileprefix + fmt.Sprintf("_%v", strconv.FormatInt(time.Now().UnixNano(), 10))
	// 	name = fileprefix + filesuffix
	// }
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return errors.Wrap(err, "OpenFile err")
	}
	defer f.Close()
	_, err = f.Write([]byte(content))
	if err != nil {
		return errors.Wrap(err, "f.Write err")
	}
	return nil
}

// parse ip
func ParseIpPort(ip string) (string, string, error) {
	if ip != "" {
		ip_port := strings.Split(ip, ":")
		if len(ip_port) == 1 {
			isipv4 := net.ParseIP(ip_port[0])
			if isipv4 != nil {
				return ip + ":15001", ":15001", nil
			}
			return ip_port[0], ":15001", nil
		}
		if len(ip_port) == 2 {
			_, err := strconv.ParseUint(ip_port[1], 10, 16)
			if err != nil {
				return "", "", err
			}
			return ip, ":" + ip_port[1], nil
		}
		return "", "", errors.New(" The IP address is incorrect")
	} else {
		return "", "", errors.New(" The IP address is nil")
	}
}

//Judge whether IP can connect with TCP normally.
//Returning true means normal.
func TestConnectionWithTcp(ip string) bool {
	if ip == "" {
		return false
	}
	tmp := strings.Split(ip, ":")
	address := ""
	if len(tmp) > 1 {
		address = ip
	} else if len(tmp) == 1 {
		address = net.JoinHostPort(ip, "80")
	} else {
		return false
	}
	_, err := net.DialTimeout("tcp", address, 2*time.Second)
	return err == nil
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

func EscapeURISpecialCharacters(in string) string {
	rtn := ""
	rtn = strings.Replace(in, "%", "%25", -1)
	rtn = strings.Replace(rtn, " ", "%20", -1)
	rtn = strings.Replace(rtn, "!", "%21", -1)
	rtn = strings.Replace(rtn, `"`, "%22", -1)
	rtn = strings.Replace(rtn, "#", "%23", -1)
	rtn = strings.Replace(rtn, "$", "%24", -1)
	rtn = strings.Replace(rtn, "&", "%26", -1)
	rtn = strings.Replace(rtn, "'", "%27", -1)
	rtn = strings.Replace(rtn, "(", "%28", -1)
	rtn = strings.Replace(rtn, ")", "%29", -1)
	rtn = strings.Replace(rtn, "*", "%2A", -1)
	rtn = strings.Replace(rtn, "+", "%2B", -1)
	rtn = strings.Replace(rtn, ",", "%2C", -1)
	rtn = strings.Replace(rtn, "/", `%2F`, -1)
	rtn = strings.Replace(rtn, ":", "%3A", -1)
	rtn = strings.Replace(rtn, ";", "%3B", -1)
	rtn = strings.Replace(rtn, "<", "%3C", -1)
	rtn = strings.Replace(rtn, "=", "%3D", -1)
	rtn = strings.Replace(rtn, ">", `%3E`, -1)
	rtn = strings.Replace(rtn, "?", `%3F`, -1)
	rtn = strings.Replace(rtn, "@", "%40", -1)
	rtn = strings.Replace(rtn, `|`, "%7C", -1)
	return rtn
}

func RandomInRange(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

var base58 = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

//Base58 encode
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

//Base58 Decode
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

//
func B2S(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func S2B(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}

//
func CreatDirIfNotExist(dir string) error {
	_, err := os.Stat(dir)
	if err != nil {
		return os.MkdirAll(dir, os.ModeDir)
	}
	return nil
}

//
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

//
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
