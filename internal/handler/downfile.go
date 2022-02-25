package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"storage-mining/configs"
	"storage-mining/internal/logger"
	"storage-mining/tools"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klauspost/reedsolomon"
	"github.com/pkg/errors"
)

func DownloadHandler(c *gin.Context) {
	var rsp = configs.RespMsg{
		Code: -1,
		Msg:  "",
		Data: nil,
	}

	sector := c.Param("hash")
	filename := c.Query("filename")
	logger.InfoLogger.Sugar().Infof("Start download [%v][%v]", filename, sector)
	path := filepath.Join(configs.Confile.FileSystem.DfsInstallPath, "files", sector)
	chunkdirs, err := walkDirDataShards(path)
	if err != nil {
		rsp.Msg = err.Error()
		logger.ErrLogger.Sugar().Errorf("The file to download is fail, filehash: %s ,error: %v", sector, "recover file is fully please wait for a min")
		c.JSON(http.StatusNotAcceptable, rsp)
		return
	}

	if len(chunkdirs) == 1 {
		sealedname := sector + ".cess"
		oldpath := filepath.Join(configs.Confile.FileSystem.DfsInstallPath, "files", sector, sealedname)
		newpath := filepath.Join(configs.Confile.FileSystem.DfsInstallPath, "files", configs.Cache, filename)
		_, err = os.Stat(newpath)
		if err == nil {
			h, err := calcFileHash(newpath)
			if err != nil {
				os.Remove(newpath)
				err = copyFile(newpath, oldpath)
				if err != nil {
					rsp.Msg = err.Error()
					c.JSON(http.StatusInternalServerError, rsp)
					return
				}
			} else {
				if h != sector {
					os.Remove(newpath)
					err = copyFile(newpath, oldpath)
					if err != nil {
						rsp.Msg = err.Error()
						c.JSON(http.StatusInternalServerError, rsp)
						return
					}
				}
			}
		} else {
			err = copyFile(newpath, oldpath)
			if err != nil {
				rsp.Msg = err.Error()
				c.JSON(http.StatusInternalServerError, rsp)
				return
			}
		}
		fn := tools.EscapeURISpecialCharacters(filename)
		downurl := "http://" + configs.Confile.MinerData.ServiceIpAddr + ":" + fmt.Sprintf("%v", configs.Confile.MinerData.FilePort) + "/group1/" + configs.Cache + "/" + fn
		rsp.Code = 0
		rsp.Msg = "success"
		rsp.Data = downurl
		logger.InfoLogger.Sugar().Infof("Search file successful [%v] ", sector)
		c.JSON(http.StatusOK, rsp)
		return
	}

	pardirs, err := walkParShards(path)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("get ParShards data fail:%v", err.Error())
	}

	err = DecodeFile(chunkdirs, pardirs, sector, filename)
	if err != nil {
		rsp.Msg = err.Error()
		logger.ErrLogger.Sugar().Errorf("The file to download is fail, filehash: %s ,error: %v", sector, err.Error())
		c.JSON(http.StatusNotAcceptable, rsp)
		return
	} else {
		fn := tools.EscapeURISpecialCharacters(filename)
		downurl := "http://" + configs.Confile.MinerData.ServiceIpAddr + ":" + fmt.Sprintf("%v", configs.Confile.MinerData.FilePort) + "/group1/" + configs.Cache + "/" + fn
		rsp.Code = 0
		rsp.Msg = "success"
		rsp.Data = downurl
		logger.InfoLogger.Sugar().Infof("Search file successful [%v]", sector)
		c.JSON(http.StatusOK, rsp)
		return
	}
}

func ReqFastDfs(url, filepath string, params map[string]string) (int, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return 0, err
	}
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	formFile, err := writer.CreateFormFile("file", params["file"])
	if err != nil {
		return 0, err
	}

	_, err = io.Copy(formFile, file)
	if err != nil {
		return 0, err
	}

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}

	err = writer.Close()
	if err != nil {
		return 0, err
	}
	request, err := http.NewRequest("POST", url, body)
	request.Header.Add("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	resp, err := client.Do(request)

	return resp.StatusCode, err
}

func CheckFileHash(path, hash string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	hashm := calcFileHash2(f)
	if hash == hashm {
		return true
	}
	return false
}

func calcFileHash2(file *os.File) string {
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		logger.ErrLogger.Sugar().Errorf("account file hash fail:%v", err)
	}

	return hex.EncodeToString(h.Sum(nil))
}

func calcFileHash(fpath string) (string, error) {
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

func DecodeFile(chunks, par []string, sector, filename string) error {
	enc, err := reedsolomon.New(len(chunks), len(par))
	if err != nil {
		return errors.Wrap(err, "reedsolomon.New err")
	}
	shards := make([][]byte, len(chunks)+len(par))
	chunks = append(chunks, par...)
	for i := range shards {
		shards[i], err = ioutil.ReadFile(chunks[i])
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("read file error when decode file:%v", err.Error())
			shards[i] = nil
		}
	}
	ok, err := enc.Verify(shards)
	if ok {
		logger.InfoLogger.Sugar().Infof("DecodeFile:%v Success! No reconstruction needed", sector)
	} else {
		logger.InfoLogger.Sugar().Infof("Verification file:%v failed. Reconstructing now...", sector)
		err = enc.Reconstruct(shards)
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("Reconstruct failed error:%v", err)
			return errors.New("Reconstruct failed error:" + err.Error())
		}
		ok, err = enc.Verify(shards)
		if !ok {
			logger.ErrLogger.Sugar().Errorf("Verification failed after reconstruction, data likely corrupted.")
			return errors.New("Verification failed after reconstruction, data likely corrupted.")
		}
	}

	recoverFilePath := filepath.Join(configs.Confile.FileSystem.DfsInstallPath, "files", configs.Cache)
	_, err = os.Stat(recoverFilePath)
	if err != nil {
		err = os.MkdirAll(recoverFilePath, os.ModePerm)
		if err != nil {
			return errors.Wrap(err, "os.MkdirAll err")
		}
	}
	recvrfile := filepath.Join(recoverFilePath, filename)
	f, err := os.Stat(recvrfile)
	if err == nil && !f.IsDir() {
		return nil
	}
	if err == nil {
		if f.IsDir() {
			os.RemoveAll(recvrfile)
		}
	}

	fn, err := os.Create(recvrfile)
	if err != nil {
		return errors.Wrap(err, "os.Create err")
	}
	defer fn.Close()

	err = enc.Join(fn, shards, len(shards[0])*(len(chunks)-len(par)))
	if err != nil {
		return errors.Wrap(err, "enc.Join err")
	}

	return nil
}

func CheckRecoverNum() int {
	workPath, err := os.Getwd()
	if err != nil {
		return 0
	}
	_, err = os.Stat(workPath)
	if err != nil {
		return 0
	}
	for i := 1; i < 100; i++ {
		recvrfilepath := filepath.Join(workPath, "recover_tmp"+strconv.Itoa(i))
		_, err = os.Stat(recvrfilepath)
		if err != nil {
			return i
		} else {
			continue
		}
	}
	return 0
}

func RemoveRecoverFile(n int) {
	workPath, err := os.Getwd()
	if err != nil {
	}
	_, err = os.Stat(workPath)
	if err != nil {
	}

	recvrfilepath := filepath.Join(workPath, "recover_tmp"+strconv.Itoa(n))
	_, err = os.Stat(recvrfilepath)
	if err != nil {
		return
	} else {
		err = os.RemoveAll(recvrfilepath)
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("remove all recover_tmp%v fail", strconv.Itoa(n))
		}
	}

}

func walkDirDataShards(filePath string) ([]string, error) {
	dirs := make([]string, 0)

	files, err := ioutil.ReadDir(filePath)
	if err != nil {
		logger.ErrLogger.Error("Script To Walk Dir error.")
		return dirs, err
	} else {
		for _, v := range files {
			if !v.IsDir() {
				if strings.Contains(v.Name(), "_") {
					fpath := filepath.Join(filePath, v.Name())
					os.Remove(fpath)
					continue
				}
				ispar := strings.Split(v.Name(), ".")
				if strings.HasPrefix(ispar[len(ispar)-1], "cess") {
					path := filepath.Join(filePath, v.Name())
					dirs = append(dirs, path)
					return dirs, nil
				}
				if !strings.HasPrefix(ispar[len(ispar)-1], "r") {
					path := filepath.Join(filePath, v.Name())
					dirs = append(dirs, path)
				}
			}
		}
	}
	orderDirs := make([]string, len(dirs))
	for i := 0; i < len(dirs); i++ {
		ispar := strings.Split(dirs[i], ".")
		index, err := strconv.Atoi(ispar[len(ispar)-1])
		if err != nil {
			return nil, err
		}
		orderDirs[index] = dirs[i]
	}
	return orderDirs, nil
}

func walkParShards(filePath string) ([]string, error) {
	dirs := make([]string, 0)
	files, err := ioutil.ReadDir(filePath)
	if err != nil {
		logger.ErrLogger.Error("Script To Walk Dir error.")
		return dirs, err
	} else {
		for _, v := range files {
			if !v.IsDir() {
				ispar := strings.Split(v.Name(), ".")
				if strings.HasPrefix(ispar[len(ispar)-1], "r") {
					path := filepath.Join(filePath, v.Name())
					dirs = append(dirs, path)
				}
			}
		}
	}

	orderDirs := make([]string, len(dirs))
	for i := 0; i < len(dirs); i++ {
		ispar := strings.Split(dirs[i], ".")
		index_int := strings.TrimLeft(ispar[len(ispar)-1], "r")
		num, err := strconv.Atoi(index_int)
		if err != nil {
			return nil, err
		}
		orderDirs[num] = dirs[i]
	}
	return orderDirs, nil
}

//func walkDirOrderly(filePath string) ([]string, error) {
//	dirs := make([]string, 0)
//	chunkslice := make([]int, 0)
//	chunkmap := make(map[int]*fs.FileInfo)
//	files, err := ioutil.ReadDir(filePath)
//	if err != nil {
//		logger.ErrLogger.Error("File Sector Path error.")
//		return dirs, err
//	} else {
//
//		for _, v := range files {
//			if !v.IsDir() {
//				num := strings.Split(v.Name(), ".")
//				chunknum, err := strconv.Atoi(num[len(num)-1])
//				if err != nil {
//					continue
//				}
//				chunkslice = append(chunkslice, chunknum)
//				chunkmap[chunknum] = &v
//			}
//		}
//		quickSort(chunkslice, 0, len(chunkslice))
//		for _, v := range chunkslice {
//			if fileinfo, ok := chunkmap[v]; ok {
//				path := filepath.Join(filePath, (*fileinfo).Name())
//				dirs = append(dirs, path)
//			}
//		}
//	}
//	return dirs, nil
//}
func quickSort(p []int, start, end int) {
	x, i, j := p[start], start+1, end
	for i <= j {
		for i <= j && p[i] <= x {
			i++
		}
		for i <= j && p[j] >= x {
			j--
		}
		if i < j {
			p[i], p[j] = p[j], p[i]
			i++
			j--
		}
	}
	if start != j {
		p[start], p[j] = p[j], x
	}
	if start < j-1 {
		quickSort(p, start, j-1)
	}
	if j+1 < end {
		quickSort(p, j+1, end)
	}
}

func copyFile(dstName, srcName string) (err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}
