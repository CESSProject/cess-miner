/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package out

import (
	"fmt"
	"time"
)

const (
	HiBlack = iota + 90
	HiRed
	HiGreen
	HiYellow
	HiBlue
	HiPurple
	HiCyan
	HiWhite
)

const (
	OkPrompt    = "OK"
	WarnPrompt  = "!!"
	ErrPrompt   = "XX"
	InputPrompt = ">>"
	TipPrompt   = "++"
)

const TimeFormat = "2006-01-02 15:04:05"

func Input(msg string) {
	fmt.Println(textInput(), msg)
}

func Tip(msg string) {
	fmt.Println(textTip(), fmt.Sprintf("%v %s", time.Now().Format(TimeFormat), msg))
}

func Err(msg string) {
	fmt.Println(textErr(), fmt.Sprintf("%v %s", time.Now().Format(TimeFormat), msg))
}

func Warn(msg string) {
	fmt.Println(textWarn(), fmt.Sprintf("%v %s", time.Now().Format(TimeFormat), msg))
}

func Ok(msg string) {
	fmt.Println(textOk(), fmt.Sprintf("%v %s", time.Now().Format(TimeFormat), msg))
}

func textTip() string {
	return fmt.Sprintf("\x1b[0;%dm%s\x1b[0m", HiGreen, TipPrompt)
}

func textInput() string {
	return fmt.Sprintf("\x1b[0;%dm%s\x1b[0m", HiBlue, InputPrompt)
}

func textErr() string {
	return fmt.Sprintf("\x1b[0;%dm%s\x1b[0m", HiRed, ErrPrompt)
}

func textOk() string {
	return fmt.Sprintf("\x1b[0;%dm%s\x1b[0m", HiGreen, OkPrompt)
}

func textWarn() string {
	return fmt.Sprintf("\x1b[0;%dm%s\x1b[0m", HiYellow, WarnPrompt)
}
