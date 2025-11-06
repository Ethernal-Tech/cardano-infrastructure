package sendtx

import (
	"strings"

	infracommon "github.com/Ethernal-Tech/cardano-infrastructure/common"
)

func AddrToMetaDataAddr(addr string) []string {
	addr = strings.TrimPrefix(strings.TrimPrefix(addr, "0x"), "0X")

	return infracommon.SplitString(addr, splitStringLength)
}

func setOrDefault[T comparable](val, def T) T {
	var zero T

	if val == zero {
		return def
	}

	return val
}
