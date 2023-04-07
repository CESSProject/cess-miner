package node

import (
	"errors"

	"github.com/CESSProject/cess-bucket/tools"
	"github.com/CESSProject/go-keyring"
)

func VerifySign(pkey, signmsg, sign []byte) (bool, error) {
	if len(signmsg) == 0 || len(sign) < 64 {
		return false, errors.New("Wrong signature")
	}

	ss58, err := tools.EncodeToSS58(pkey)
	if err != nil {
		return false, err
	}

	verkr, _ := keyring.FromURI(ss58, keyring.NetSubstrate{})

	var sign_array [64]byte
	for i := 0; i < 64; i++ {
		sign_array[i] = sign[i]
	}

	// Verify signature
	return verkr.Verify(verkr.SigningContext(signmsg), sign_array), nil
}
