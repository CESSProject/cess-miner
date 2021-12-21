package handler

import (
	"fmt"
	"io/ioutil"
	"storage-mining/configs"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Handler_main() {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = ioutil.Discard
	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true                                                                                                 
	config.AllowMethods = []string{"GET", "POST", "OPTIONS"}
	config.AllowHeaders = []string{"tus-resumable", "upload-length", "upload-metadata", "cache-control", "x-requested-with", "*"}
	r.Use(cors.New(config))

	//TODO:
	//r.POST("/upfile", UploadHandler)
	r.GET("/downfile/:hash", DownloadHandler)
	r.Run(":" + fmt.Sprintf("%v", configs.Confile.MinerData.ServicePort))
}
