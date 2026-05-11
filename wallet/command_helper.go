package wallet

import (
	"bytes"
	"os"
	"os/exec"
	"strconv"
)

const (
	FilePermission        = 0750
	cardanoCliEnv         = "CARDANO_CLI_BINARY"
	defaultCardanoCliName = "cardano-cli"
)

type runCommandError struct {
	desc string
	base error
}

func (rce runCommandError) Error() string {
	if rce.desc != "" {
		return rce.desc
	}

	return rce.base.Error()
}

func ResolveCardanoCliBinary(name ...string) string {
	if len(name) > 0 && name[0] != "" {
		return name[0]
	}

	if bin := os.Getenv(cardanoCliEnv); bin != "" {
		return bin
	}
	// fallback
	return defaultCardanoCliName
}

func runCommand(binary string, args []string, envVariables ...string) (string, error) {
	var (
		stdErrBuffer bytes.Buffer
		stdOutBuffer bytes.Buffer
	)

	cmd := exec.Command(binary, args...)
	cmd.Stderr = &stdErrBuffer
	cmd.Stdout = &stdOutBuffer

	cmd.Env = append(os.Environ(), envVariables...)

	err := cmd.Run()

	if stdErrBuffer.Len() > 0 {
		return "", runCommandError{desc: stdErrBuffer.String()}
	} else if err != nil {
		return "", runCommandError{base: err}
	}

	return stdOutBuffer.String(), nil
}

func getTestNetMagicArgs(testnetMagic uint) []string {
	if testnetMagic == 0 || testnetMagic == MainNetProtocolMagic {
		return []string{"--mainnet"}
	}

	return []string{"--testnet-magic", strconv.FormatUint(uint64(testnetMagic), 10)}
}
