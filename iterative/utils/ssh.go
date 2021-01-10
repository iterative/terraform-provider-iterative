package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"

	"golang.org/x/crypto/ssh"
)

func PrivatePEM() (string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", err
	}

	var privKeyBuf strings.Builder
	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	if err := pem.Encode(&privKeyBuf, privateKeyPEM); err != nil {
		return "", err
	}

	return privKeyBuf.String(), nil
}

func PublicFromPrivatePEM(privateKey string) (string, error) {
	block, _ := pem.Decode([]byte(privateKey))
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	pub, err := ssh.NewPublicKey(&key.PublicKey)
	if err != nil {
		return "", err
	}

	var pubKeyBuf strings.Builder
	pubKeyBuf.Write(ssh.MarshalAuthorizedKey(pub))

	return pubKeyBuf.String(), nil
}
