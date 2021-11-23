package ssh

import (
	"time"

	"golang.org/x/crypto/ssh"
)

func RunCommand(command string, timeout time.Duration, hostAddress string, userName string, privateKey string) (string, error) {
	parsedPrivateKey, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return "", err
	}

	configuration := &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(parsedPrivateKey),
		},
		// Doesn't matter for this use case, but isn't a good practice either.
		// FIXME: IT **DOES** MATTER NOW! FIND A SOLUTION ASAP!
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}

	client, err := ssh.Dial("tcp", hostAddress, configuration)
	if err != nil {
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return "", err
	}

	return string(output), nil
}
