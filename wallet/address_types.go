package wallet

import (
	"encoding/hex"
	"fmt"
)

type CardanoAddressPayload struct {
	Payload  [KeyHashSize]byte `cbor:"1,keyasint,omitempty"`
	IsScript bool              `cbor:"0,keyasint,omitempty"`
}

func (sc CardanoAddressPayload) String() string {
	return hex.EncodeToString(sc.Payload[:])
}

type StakePointer struct {
	Slot      uint64
	TxIndex   uint64
	CertIndex uint64
}

type cardanoAddressParser interface {
	GetAddressType() CardanoAddressType
	IsValid(bytes []byte) error
	GetPrefix(network CardanoNetworkType) string
	ToCardanoAddressInfo(bytes []byte) CardanoAddressInfo
	FromCardanoAddressInfo(a CardanoAddressInfo) []byte
}

var addressParsers = []cardanoAddressParser{
	&cardanoBaseAddressParser{}, &cardanoPointerAddressParser{},
	&cardanoEnterpriseAddressParser{}, &cardanoRewardAddressParser{},
}

// cardanoBaseAddressParser BaseAddress
// 0000: keyhash28,keyhash28
// 0001: scripthash28,keyhash28
// 0010: keyhash28,scripthash28
// 0011: scripthash28,scripthash28
type cardanoBaseAddressParser struct{}

func (cap cardanoBaseAddressParser) GetAddressType() CardanoAddressType {
	return BaseAddress
}

func (cap cardanoBaseAddressParser) IsValid(bytes []byte) error {
	if len(bytes) < 1+KeyHashSize*2 {
		return fmt.Errorf("%w: expect %d got %d", ErrInvalidData, 1+KeyHashSize*2, len(bytes))
	}

	return nil
}

func (cap cardanoBaseAddressParser) GetPrefix(network CardanoNetworkType) string {
	return network.GetPrefix()
}

func (cap cardanoBaseAddressParser) ToCardanoAddressInfo(bytes []byte) CardanoAddressInfo {
	header, data := (bytes[0]&0xF0)>>4, bytes[1:]

	return CardanoAddressInfo{
		AddressType: BaseAddress,
		Network:     CardanoNetworkType(bytes[0] & 0x0F),
		Payment: &CardanoAddressPayload{
			Payload:  [KeyHashSize]byte(data[:KeyHashSize]),
			IsScript: header&1 > 0,
		},
		Stake: &CardanoAddressPayload{
			Payload:  [KeyHashSize]byte(data[KeyHashSize : KeyHashSize*2]),
			IsScript: header&2 > 0,
		},
		Extra: bytes[2*KeyHashSize:],
	}
}

func (cap cardanoBaseAddressParser) FromCardanoAddressInfo(a CardanoAddressInfo) []byte {
	bytes := make([]byte, KeyHashSize*2+1+len(a.Extra))
	bytes[0] = (toByte(a.Payment.IsScript) << 4) | (toByte(a.Stake.IsScript) << 5) | (byte(a.Network) & 0xf)

	copy(bytes[1:KeyHashSize+1], a.Payment.Payload[:])
	copy(bytes[KeyHashSize+1:], a.Stake.Payload[:])
	copy(bytes[KeyHashSize*2+1:], a.Extra)

	return bytes
}

// cardanoPointerAddressParser
// 0100: keyhash28, 3 variable length uint
// 0101: scripthash28, 3 variable length uint
type cardanoPointerAddressParser struct{}

func (cap cardanoPointerAddressParser) GetAddressType() CardanoAddressType {
	return PointerAddress
}

func (cap cardanoPointerAddressParser) IsValid(bytes []byte) error {
	if len(bytes) < 1+KeyHashSize+1+1+1 { // header + payment + at least one byte for all three pointer parts
		return fmt.Errorf("%w: expect at least %d got %d", ErrInvalidData, 1+KeyHashSize+1+1+1, len(bytes))
	}

	_, err := getStakePointer(bytes[1+KeyHashSize:])

	return err
}

func (cap cardanoPointerAddressParser) GetPrefix(network CardanoNetworkType) string {
	return network.GetPrefix()
}

func (cap cardanoPointerAddressParser) ToCardanoAddressInfo(bytes []byte) CardanoAddressInfo {
	header, data := (bytes[0]&0xF0)>>4, bytes[1:]
	pointer, _ := getStakePointer(data[KeyHashSize:])

	return CardanoAddressInfo{
		AddressType: PointerAddress,
		Network:     CardanoNetworkType(bytes[0] & 0x0F),
		Payment: &CardanoAddressPayload{
			Payload:  [KeyHashSize]byte(data[:KeyHashSize]),
			IsScript: header&1 > 0,
		},
		StakePointer: pointer,
	}
}

func (cap cardanoPointerAddressParser) FromCardanoAddressInfo(a CardanoAddressInfo) []byte {
	variableEncode := func(num uint64) []byte {
		var output []byte

		output = append(output, byte(num)&0x7F)
		num /= 128

		for num > 0 {
			output = append(output, byte(num)&0x7F|0x80)
			num /= 128
		}

		for i, j := 0, len(output)-1; i < j; i, j = i+1, j-1 {
			output[i], output[j] = output[j], output[i]
		}

		return output
	}

	buf := make([]byte, 0, KeyHashSize+1+3)

	header := 0b0100_0000 | (toByte(a.Payment.IsScript) << 4) | (byte(a.Network) & 0xf)
	buf = append(buf, header)
	buf = append(buf, a.Payment.Payload[:]...)
	buf = append(buf, variableEncode(a.StakePointer.Slot)...)
	buf = append(buf, variableEncode(a.StakePointer.TxIndex)...)

	return append(append(buf, variableEncode(a.StakePointer.CertIndex)...), a.Extra...)
}

// cardanoEnterpriseAddressParser EnterpriseAddress
// 0110: keyhash28
// 0111: scripthash28
type cardanoEnterpriseAddressParser struct{}

func (cap cardanoEnterpriseAddressParser) GetAddressType() CardanoAddressType {
	return EnterpriseAddress
}

func (cap cardanoEnterpriseAddressParser) IsValid(bytes []byte) error {
	if len(bytes) != KeyHashSize+1 {
		return fmt.Errorf("%w: expect %d got %d", ErrInvalidData, 1+KeyHashSize, len(bytes))
	}

	return nil
}

func (cap cardanoEnterpriseAddressParser) GetPrefix(network CardanoNetworkType) string {
	return network.GetPrefix()
}

func (cap cardanoEnterpriseAddressParser) ToCardanoAddressInfo(bytes []byte) CardanoAddressInfo {
	header, data := (bytes[0]&0xF0)>>4, bytes[1:]

	return CardanoAddressInfo{
		AddressType: EnterpriseAddress,
		Network:     CardanoNetworkType(bytes[0] & 0x0F),
		Payment: &CardanoAddressPayload{
			Payload:  [KeyHashSize]byte(data[:KeyHashSize]),
			IsScript: header&1 > 0,
		},
		Extra: bytes[KeyHashSize:],
	}
}

func (cap cardanoEnterpriseAddressParser) FromCardanoAddressInfo(a CardanoAddressInfo) []byte {
	bytes := make([]byte, KeyHashSize+1+len(a.Extra))
	bytes[0] = 0b01100000 | (toByte(a.Payment.IsScript) << 4) | (byte(a.Network) & 0xf)

	copy(bytes[1:], a.Payment.Payload[:])
	copy(bytes[1+KeyHashSize:], a.Extra)

	return bytes
}

// cardanoRewardAddressParser RewardAddress
// 0110: keyhash28
// 0111: scripthash28
type cardanoRewardAddressParser struct{}

func (cap cardanoRewardAddressParser) GetAddressType() CardanoAddressType {
	return RewardAddress
}

func (cap cardanoRewardAddressParser) IsValid(bytes []byte) error {
	if len(bytes) != KeyHashSize+1 {
		return fmt.Errorf("%w: expect %d got %d", ErrInvalidData, 1+KeyHashSize, len(bytes))
	}

	return nil
}

func (cap cardanoRewardAddressParser) GetPrefix(network CardanoNetworkType) string {
	return network.GetStakePrefix()
}

func (cap cardanoRewardAddressParser) ToCardanoAddressInfo(bytes []byte) CardanoAddressInfo {
	header, data := (bytes[0]&0xF0)>>4, bytes[1:]

	return CardanoAddressInfo{
		AddressType: RewardAddress,
		Network:     CardanoNetworkType(bytes[0] & 0x0F),
		Payment: &CardanoAddressPayload{
			Payload:  [KeyHashSize]byte(data[:KeyHashSize]),
			IsScript: header&1 > 0,
		},
		Extra: bytes[KeyHashSize:],
	}
}

func (cap cardanoRewardAddressParser) FromCardanoAddressInfo(a CardanoAddressInfo) []byte {
	bytes := make([]byte, KeyHashSize+1+len(a.Extra))
	bytes[0] = 0b1110_0000 | (toByte(a.Stake.IsScript) << 4) | (byte(a.Network) & 0xf)

	copy(bytes[1:], a.Stake.Payload[:])
	copy(bytes[1+KeyHashSize:], a.Extra)

	return bytes
}

func getStakePointer(raw []byte) (*StakePointer, error) {
	readOne := func(raw []byte) (result uint64, bytesReadCnt int, err error) {
		for _, rbyte := range raw {
			result = (result << 7) | uint64(rbyte&0x7F)
			bytesReadCnt++

			if (rbyte & 0x80) == 0 {
				return result, bytesReadCnt, nil
			}
		}

		return 0, 0, ErrInvalidData
	}

	slot, bytesReadCnt, err := readOne(raw)
	if err != nil {
		return nil, err
	}

	txIndex, bytesReadCnt2, err := readOne(raw[bytesReadCnt:])
	if err != nil {
		return nil, err
	}

	certIndex, bytesReadCnt3, err := readOne(raw[bytesReadCnt+bytesReadCnt2:])
	if err != nil {
		return nil, err
	}

	if bytesReadCnt+bytesReadCnt2+bytesReadCnt3 != len(raw) {
		return nil, ErrInvalidData
	}

	return &StakePointer{
		Slot:      slot,
		TxIndex:   txIndex,
		CertIndex: certIndex,
	}, nil
}

func toByte(b bool) byte {
	if !b {
		return 0
	}

	return 1
}

func getAddressParser(addressType CardanoAddressType) (cardanoAddressParser, error) {
	for _, parser := range addressParsers {
		if parser.GetAddressType() == addressType {
			return parser, nil
		}
	}

	return nil, ErrUnsupportedAddress
}
