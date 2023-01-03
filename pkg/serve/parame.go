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

package serve

const (
	Msg_Ping     = 100
	Msg_Auth     = 101
	Msg_File     = 102
	Msg_Down     = 103
	Msg_Confirm  = 104
	Msg_Progress = 105
)

const (
	Msg_OK        = 200
	Msg_OK_FILE   = 201
	Msg_ClientErr = 400
	Msg_NotFound  = 404
	Msg_Forbidden = 403
	Msg_ServerErr = 500
)

const (
	TokenKey_Token = "token_token:"
	TokenKey_Acc   = "token_acc:"
)
