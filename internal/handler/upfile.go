package handler

type RespFileType struct {
	Code     int    `json:"code"`
	Msg      string `json:"msg"`
	FileSize uint64 `json:"filesize"`
	FileHash string `json:"filehash"`
}

// func UploadHandler(c *gin.Context) {
// 	var (
// 		err error
// 		rsp = RespFileType{
// 			Code:     -1,
// 			Msg:      "",
// 			FileSize: 0,
// 			FileHash: "",
// 		}
// 	)
// 	file, err := c.FormFile("file")
// 	if err != nil {
// 		rsp.Msg = "invalid form file"
// 		c.JSON(http.StatusBadRequest, rsp)
// 		return
// 	}
// 	sector, _ := c.GetPostForm("filehash") //file hash
// 	filenum, _ := c.GetPostForm("filenum") //chunk number
// 	kind, _ := c.GetPostForm("kind")       //if redundancy chunk or recover file

// 	if sector == "" || filenum == "" || kind == "" {
// 		rsp.Msg = "invalid form data"
// 		c.JSON(http.StatusBadRequest, rsp)
// 	}

// 	hash := calcFileHash1(file)
// 	size := calcFileSize1(file)
// 	if kind == "redundant" {
// 		sector += "/redundant"
// 	}
// 	if kind == "recover" {
// 		sector += "/recover"
// 	}

// 	params := map[string]string{
// 		"file":   filenum,
// 		"output": "json",
// 		"scene":  "SaveFile",
// 		"path":   sector,
// 	}

// 	//upload file to fastdfs
// 	//folder name after sector,filename name after filehash
// 	_, err = SaveFileToFDS(configs.Confile.FileSystem.UpfileUrl, file, params)

// 	if err != nil {
// 		logger.ErrLogger.Sugar().Errorf("update file fail ,error:%s", err)
// 		rsp.Msg = fmt.Sprintf("upload file fail, err:%v", err)
// 		c.JSON(http.StatusBadGateway, rsp)
// 		return
// 	}
// 	rsp.Code = 0
// 	rsp.Msg = "success"
// 	rsp.FileHash = hash
// 	rsp.FileSize = size
// 	logger.InfoLogger.Sugar().Infof("A file is stored, filehash: %s", hash)
// 	fmt.Printf("\x1b[%dm[ok]\x1b[0m A file is stored, filehash: %s\n", 42, hash)
// 	c.JSON(http.StatusOK, rsp)
// }

// func SaveFileToFDS(uri string, file *multipart.FileHeader, params map[string]string) (*http.Response, error) {
// 	body := &bytes.Buffer{}
// 	writer := multipart.NewWriter(body)
// 	part, err := writer.CreateFormFile("file", params["file"])
// 	if err != nil {
// 		return nil, err
// 	}
// 	src, err := file.Open()
// 	defer src.Close()

// 	_, err = io.Copy(part, src)
// 	for key, val := range params {
// 		_ = writer.WriteField(key, val)
// 	}
// 	err = writer.Close()
// 	if err != nil {
// 		return nil, err
// 	}
// 	request, err := http.NewRequest("POST", uri, body)
// 	request.Header.Add("Content-Type", writer.FormDataContentType())
// 	client := &http.Client{}
// 	resp, err := client.Do(request)
// 	return resp, err
// }

// func calcFileHash1(file *multipart.FileHeader) string {
// 	src, err := file.Open()
// 	if err != nil {
// 		logger.ErrLogger.Sugar().Errorf("%v", err)
// 	}
// 	defer src.Close()

// 	h := sha256.New()
// 	if _, err := io.Copy(h, src); err != nil {
// 		logger.ErrLogger.Sugar().Errorf("account file hash fail:%v", err)
// 	}

// 	return hex.EncodeToString(h.Sum(nil))
// }

// func calcFileSize1(file *multipart.FileHeader) uint64 {
// 	return uint64(file.Size)
// }
