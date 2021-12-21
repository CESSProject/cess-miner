package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"storage-mining/internal/logger"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type PolicyBase struct {
	Ver          int    `json:"ver"`
	Expired      int64  `json:"expired"`
	CallbackUrl  string `json:"callback_url,omitempty"`
	CallbackBody string `json:"callback_body,omitempty"`
}

type Policy struct {
	*PolicyBase
	Ext json.RawMessage `json:"ext,omitempty"`
}

type AddExt struct {
	FileName string `json:"file_name"`
	Size     uint64 `json:"size"`
	Md5      string `json:"md5"`
}

type tokenObj struct {
	addr   string
	sign   string
	policy string
	raw    *Policy
}

type callbackResult struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
}

const (
	CESS_SK = "76E18tAYEU2WPLww2DwPvM6"
)

func parseToken(tokenStr string) (*tokenObj, error) {
	if tokenStr == "" {
		return nil, errors.New("empty token")
	}
	//Base64 encode(addr, sign, policy)
	tokenSlice := strings.Split(tokenStr, ":")
	if len(tokenSlice) != 3 {
		return nil, errors.New("token format error")
	}
	//decode policy
	bs, err := base64.URLEncoding.DecodeString(tokenSlice[2])
	if err != nil {
		return nil, errors.Wrap(err, "decode token failed")
	}
	//unmarshal json
	obj := &Policy{}
	err = json.Unmarshal(bs, obj)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal raw string failed")
	}
	return &tokenObj{
		addr:   tokenSlice[0],
		sign:   tokenSlice[1],
		policy: tokenSlice[2],
		raw:    obj,
	}, nil
}

// Verify Token
func VerifyToken(token string) (*Policy, bool, error) {
	// Parse
	to, err := parseToken(token)
	if err != nil {
		return nil, false, errors.New("parse token failed" + err.Error())
	}
	//Expiration
	if time.Now().After(time.Unix(to.raw.Expired, 0)) {
		return nil, false, errors.New("token expired...")
	}
	return to.raw, true, nil
}

// callback
func CallBack(tp *Policy, size, filename, hash string) {
	if tp.CallbackUrl != "" {
		tp.CallbackBody = strings.ReplaceAll(tp.CallbackBody, "$(size)", size)
		tp.CallbackBody = strings.ReplaceAll(tp.CallbackBody, "$(file_name)", filename)
		tp.CallbackBody = strings.ReplaceAll(tp.CallbackBody, "$(hash)", hash)

		err := doCallback(tp.CallbackUrl, tp.CallbackBody)
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("Callback error: %s", err.Error())
			return
		}
	}
}

func doCallback(callbackUrl, callbackBody string) error {
	resp, err := http.DefaultClient.Post(callbackUrl, "application/json", bytes.NewBufferString(callbackBody))
	if err != nil {
		return errors.Wrap(err, "callback failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("callback return errror status code: %d", resp.StatusCode)
	}

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("Callback error: %s", err.Error())
		return err
	}
	cr := &callbackResult{}
	err = json.Unmarshal(bs, cr)
	if err != nil {
		return errors.Wrap(err, "unmarshal callback result failed")
	}

	if cr.Success {
		return nil
	}

	return errors.Errorf("callback return false: %s", cr.Msg)
}
