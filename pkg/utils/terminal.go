package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/ssh/terminal"
)

func PasswdWithMask(hint string, defaultVal string, mask string) (string, error) {
	return passwd(hint, defaultVal, mask)
}

func passwd(hint string, defaultVal string, mask string) (string, error) {
	var ioBuf []rune
	if hint != "" {
		fmt.Print(hint)
	}
	if strings.Index(hint, "\n") >= 0 {
		hint = strings.TrimSpace(hint[strings.LastIndex(hint, "\n"):])
	}
	fd := int(os.Stdin.Fd())
	state, err := terminal.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer fmt.Println()
	defer terminal.Restore(fd, state)
	inputReader := bufio.NewReader(os.Stdin)
	for {
		b, _, err := inputReader.ReadRune()
		if err != nil {
			return "", err
		}
		if b == 0x0d {
			strValue := strings.TrimSpace(string(ioBuf))
			if len(strValue) == 0 {
				strValue = defaultVal
			}
			return strValue, nil
		}
		if b == 0x08 || b == 0x7F {
			if len(ioBuf) > 0 {
				ioBuf = ioBuf[:len(ioBuf)-1]
			}
			fmt.Print("\r")
			for i := 0; i < len(ioBuf)+2+len(hint); i++ {
				fmt.Print(" ")
			}
		} else {
			ioBuf = append(ioBuf, b)
		}
		fmt.Print("\r")
		if hint != "" {
			fmt.Print(hint)
		}
	}
}
