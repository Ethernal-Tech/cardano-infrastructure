package sendtx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddrToMetaDataAddr(t *testing.T) {
	t.Run("address", func(t *testing.T) {
		require.Equal(t, []string{
			"addr_test1yz2fxv2umyhttkxyxp8x0dlpdt3k6c", "wng5pxj3jhsydzerkr0vd4msrxnuwnccdxlhdjar", "77j6lg0wypcc9uar5d2shsf5r8qx",
		}, AddrToMetaDataAddr(
			"addr_test1yz2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzerkr0vd4msrxnuwnccdxlhdjar77j6lg0wypcc9uar5d2shsf5r8qx"))
	})

	t.Run("address with 0x prefix", func(t *testing.T) {
		require.Equal(t, []string{
			"addr_test1yz2fxv2umyhttkxyxp8x0dlpdt3k6c", "wng5pxj3jhsydzerkr0vd4msrxnuwnccdxlhdjar", "77j6lg0wypcc9uar5d2shsf5r8qx",
		}, AddrToMetaDataAddr(
			"0xaddr_test1yz2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzerkr0vd4msrxnuwnccdxlhdjar77j6lg0wypcc9uar5d2shsf5r8qx"))
	})

	t.Run("address with 0X prefix", func(t *testing.T) {
		require.Equal(t, []string{
			"addr_test1yz2fxv2umyhttkxyxp8x0dlpdt3k6c", "wng5pxj3jhsydzerkr0vd4msrxnuwnccdxlhdjar", "77j6lg0wypcc9uar5d2shsf5r8qx",
		}, AddrToMetaDataAddr(
			"0xaddr_test1yz2fxv2umyhttkxyxp8x0dlpdt3k6cwng5pxj3jhsydzerkr0vd4msrxnuwnccdxlhdjar77j6lg0wypcc9uar5d2shsf5r8qx"))
	})

	t.Run("short with 0x prefix", func(t *testing.T) {
		require.Equal(t, []string{
			"test",
		}, AddrToMetaDataAddr("test"))
	})

	t.Run("empty", func(t *testing.T) {
		require.Nil(t, AddrToMetaDataAddr(""))
	})
}
