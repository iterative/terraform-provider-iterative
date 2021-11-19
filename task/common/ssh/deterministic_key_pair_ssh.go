package ssh

import (
	"bytes"
	"crypto"

	"golang.org/x/crypto/ssh"

	"github.com/cloudflare/gokey"
)

func NewDeterministicSSHKeyPair(key, realm string) (*DeterministicSSHKeyPair, error) {
	privateKey, err := gokey.GetKey(key, realm, nil, gokey.RSA4096, true)
	if err != nil {
		return nil, err
	}

	d := new(DeterministicSSHKeyPair)
	d.key = privateKey
	return d, nil
}

type DeterministicSSHKeyPair struct {
	key crypto.PrivateKey
}

func (d *DeterministicSSHKeyPair) PrivateKey() (ssh.Signer, error) {
	return ssh.NewSignerFromKey(d.key)
}

func (d *DeterministicSSHKeyPair) PublicKey() (ssh.PublicKey, error) {
	private, err := d.PrivateKey()
	if err != nil {
		return nil, err
	}
	return private.PublicKey(), nil
}

func (d *DeterministicSSHKeyPair) PrivateString() (string, error) {
	var buf bytes.Buffer
	if err := gokey.EncodeToPem(d.key, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (d *DeterministicSSHKeyPair) PublicString() (string, error) {
	public, err := d.PublicKey()
	if err != nil {
		return "", err
	}
	return string(ssh.MarshalAuthorizedKey(public)), nil
}
