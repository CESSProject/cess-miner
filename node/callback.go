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
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func (n *Node) StartCallback() {
	gin.SetMode(gin.ReleaseMode)
	go func() {
		n.CallBack = gin.Default()
		config := cors.DefaultConfig()
		config.AllowAllOrigins = true
		config.AllowMethods = []string{"POST", "OPTIONS"}
		config.AddAllowHeaders("*")
		n.CallBack.Use(cors.New(config))
		// Add route
		n.AddRoute()
		log.Printf("[START] Callback listening on port %d\n", n.Cfile.GetSgxPortNum())
		// Run
		n.CallBack.Run(":" + fmt.Sprintf("%d", n.Cfile.GetSgxPortNum()))
	}()
}
