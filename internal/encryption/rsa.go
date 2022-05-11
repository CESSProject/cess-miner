package encryption

import (
	"cess-bucket/configs"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

func GenKeypair() {
	if err := generateRSAKeyfile(2048); err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
}

func Check_Keypair() {
	privateKeyfile := filepath.Join(configs.BaseDir, configs.PrivateKeyfile)
	publicKeyfile := filepath.Join(configs.BaseDir, configs.PublicKeyfile)

	_, err := os.Stat(privateKeyfile)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
	_, err = os.Stat(publicKeyfile)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
}

// generate key file
func generateRSAKeyfile(bits int) error {
	var (
		err            error
		privateKeyfile = filepath.Join(configs.BaseDir, configs.PrivateKeyfile)
		publicKeyfile  = filepath.Join(configs.BaseDir, configs.PublicKeyfile)
	)
	_, err = os.Stat(privateKeyfile)
	if err == nil {
		os.Remove(privateKeyfile)
	}
	_, err = os.Stat(publicKeyfile)
	if err == nil {
		os.Remove(publicKeyfile)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return err
	}
	X509PrivateKey := x509.MarshalPKCS1PrivateKey(privateKey)
	privateBlock := pem.Block{Type: "RSA PRIVATE KEY", Bytes: X509PrivateKey}

	privateFile, err := os.Create(privateKeyfile)
	if err != nil {
		return err
	}
	defer privateFile.Close()

	err = pem.Encode(privateFile, &privateBlock)
	if err != nil {
		return err
	}

	publicKey := privateKey.PublicKey
	X509PublicKey, err := x509.MarshalPKIXPublicKey(&publicKey)
	if err != nil {
		return err
	}
	publicBlock := pem.Block{Type: "RSA PUBLIC KEY", Bytes: X509PublicKey}

	publicFile, err := os.Create(publicKeyfile)
	if err != nil {
		return err
	}
	defer publicFile.Close()
	err = pem.Encode(publicFile, &publicBlock)
	if err != nil {
		return err
	}
	return nil
}

// Parse private key file
func GetRSAPrivateKey(path string) (*rsa.PrivateKey, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, _ := file.Stat()
	buf := make([]byte, info.Size())
	file.Read(buf)
	block, _ := pem.Decode(buf)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	return privateKey, nil
}

// Parse public key file
func GetRSAPublicKey(path string) (*rsa.PublicKey, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, _ := file.Stat()
	buf := make([]byte, info.Size())
	file.Read(buf)
	block, _ := pem.Decode(buf)
	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		panic(err)
	}
	publicKey := publicKeyInterface.(*rsa.PublicKey)
	return publicKey, nil
}

// Parse private key
func ParsePrivateKey(key []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(key)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	return privateKey, err
}

// Parse public key
func ParsePublicKey(key []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(key)
	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	publicKey := publicKeyInterface.(*rsa.PublicKey)
	return publicKey, nil
}

// Calculate the signature
func CalcSign(msg []byte, privkey *rsa.PrivateKey) ([]byte, error) {
	hash := sha256.New()
	hash.Write(msg)
	bytes := hash.Sum(nil)
	sign, err := rsa.SignPKCS1v15(rand.Reader, privkey, crypto.SHA256, bytes)
	if err != nil {
		return nil, err
	}
	return sign, nil
}

// Verify signature
func VerifySign(msg []byte, sign []byte, pubkey *rsa.PublicKey) bool {
	hash := sha256.New()
	hash.Write(msg)
	bytes := hash.Sum(nil)
	err := rsa.VerifyPKCS1v15(pubkey, crypto.SHA256, bytes, sign)
	return err == nil
}

// public key encryption
// plainText: data to be encrypted
func RSA_Encrypt(content []byte) ([]byte, error) {
	pub, err := os.Open(configs.PublicKeyfile)
	if err != nil {
		return nil, err
	}
	defer pub.Close()
	info, _ := pub.Stat()
	buf := make([]byte, info.Size())
	pub.Read(buf)
	//pem decoding
	block, _ := pem.Decode(buf)
	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	publicKey := publicKeyInterface.(*rsa.PublicKey)
	//encryption
	cipherText, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, content)
	if err != nil {
		return nil, err
	}
	return cipherText, nil
}

// private key decryption
// cipherText: encrypted data
func RSA_Decrypt(cipherText []byte) ([]byte, error) {
	prv, err := os.Open(configs.PrivateKeyfile)
	if err != nil {
		return nil, err
	}
	defer prv.Close()
	info, _ := prv.Stat()
	buf := make([]byte, info.Size())
	prv.Read(buf)
	//pem decoding
	block, _ := pem.Decode(buf)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	//decryption
	content, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, cipherText)
	if err != nil {
		return nil, err
	}
	return content, nil
}
