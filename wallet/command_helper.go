package wallet

import (
	"bytes"
	"os"
	"os/exec"
	"strconv"
)

const FilePermission = 0750

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

func ResolveCardanoCliBinary(networkID CardanoNetworkType) string {
	var env, name string

	switch networkID {
	case VectorMainNetNetwork, VectorTestNetNetwork:
		env = "CARDANO_CLI_BINARY_VECTOR"
		name = "vector-cli"
	default:
		env = "CARDANO_CLI_BINARY"
		name = "cardano-cli"
	}

	if bin := os.Getenv(env); bin != "" {
		return bin
	}
	// fallback
	return name
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
