/*
   Copyright 2022 CESS (Cumulus Encrypted Storage System) authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package node

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/gin-gonic/gin"
)

func (n *Node) GetSign(c *gin.Context) {
	var (
		err error
	)
	val, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, nil)
		return
	}

	rtnValue := strings.TrimPrefix(string(val), "\"")
	rtnValue = strings.TrimSuffix(rtnValue, "\"")

	fmt.Println("Get string(sign): ", rtnValue)

	go func() { configs.Ch_Sign <- rtnValue }()
	c.JSON(http.StatusOK, nil)
	return
}
