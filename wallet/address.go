package wallet

import (
	"errors"

	"github.com/Ethernal-Tech/cardano-infrastructure/wallet/bech32"
)

var (
	ErrUnsupportedAddress = errors.New("invalid/unsupported address type")
	ErrInvalidData        = errors.New("invalid data")
)

type CardanoAddressType byte

const (
	UnsupportedAddress CardanoAddressType = 0
	BaseAddress        CardanoAddressType = 1 // 0b0000, 0b0001, 0b0010, 0b0011
	PointerAddress     CardanoAddressType = 2 // 0b0100, 0b0101
	EnterpriseAddress  CardanoAddressType = 3 // 0b0110, 0b0111
	RewardAddress      CardanoAddressType = 4 // 0b1110, 0b1111
)

func GetAddressTypeFromHeader(header byte) CardanoAddressType {
	switch (header & 0xF0) >> 4 {
	case 0b0000, 0b0001, 0b0010, 0b0011:
		return BaseAddress
	case 0b0100, 0b0101:
		return PointerAddress
	case 0b0110, 0b0111:
		return EnterpriseAddress
	case 0b1110, 0b1111:
		return RewardAddress
	default:
		return UnsupportedAddress
	}
}

type CardanoAddress struct {
	raw []byte

	addressParser     cardanoAddressParser
	cachedAddressInfo CardanoAddressInfo
}

func NewCardanoAddress(raw []byte) (*CardanoAddress, error) {
	if len(raw) == 0 {
		return nil, ErrInvalidAddressInfo
	}

	addressParser, err := getAddressParser(GetAddressTypeFromHeader(raw[0]))
	if err != nil {
		return nil, errors.Join(ErrUnsupportedAddress, err)
	}

	if err := addressParser.IsValid(raw); err != nil {
		return nil, err
	}

	return &CardanoAddress{
		raw:           raw,
		addressParser: addressParser,
	}, nil
}

func NewAddress(raw string) (addr *CardanoAddress, err error) {
	var data []byte

	if !IsAddressWithValidPrefix(raw) {
		return nil, ErrUnsupportedAddress // byron not supported data = base58.Decode(raw)
	}

	_, data, err = bech32.DecodeToBase256(raw)
	if err != nil {
		return nil, err
	}

	return NewCardanoAddress(data)
}

func (a *CardanoAddress) GetInfo() CardanoAddressInfo {
	if a.cachedAddressInfo.AddressType != UnsupportedAddress {
		return a.cachedAddressInfo
	}

	a.cachedAddressInfo = a.addressParser.ToCardanoAddressInfo(a.raw)

	return a.cachedAddressInfo
}

func (a *CardanoAddress) GetBytes() []byte {
	return a.raw
}

func (a CardanoAddress) String() string {
	str, _ := bech32.EncodeFromBase256(a.addressParser.GetPrefix(CardanoNetworkType(a.raw[0]&0x0F)), a.raw)

	return str
}

type CardanoAddressInfo struct {
	AddressType  CardanoAddressType
	Network      CardanoNetworkType
	Payment      *CardanoAddressPayload
	Stake        *CardanoAddressPayload
	StakePointer *StakePointer
	Extra        []byte
}

func (cai CardanoAddressInfo) ToCardanoAddress() (*CardanoAddress, error) {
	parser, err := getAddressParser(cai.AddressType)
	if err != nil {
		return nil, err
	}

	return NewCardanoAddress(parser.FromCardanoAddressInfo(cai))
}

func NewBaseAddress(
	network CardanoNetworkType, paymentVerificationKey, stakeVerificationKey []byte,
) (*CardanoAddress, error) {
	paymentHash, err := GetKeyHashBytes(paymentVerificationKey)
	if err != nil {
		return nil, err
	}

	stakeHash, err := GetKeyHashBytes(stakeVerificationKey)
	if err != nil {
		return nil, err
	}

	return CardanoAddressInfo{
		AddressType: BaseAddress,
		Network:     network,
		Payment: &CardanoAddressPayload{
			Payload:  [KeyHashSize]byte(paymentHash),
			IsScript: false,
		},
		Stake: &CardanoAddressPayload{
			Payload:  [KeyHashSize]byte(stakeHash),
			IsScript: false,
		},
	}.ToCardanoAddress()
}

func NewEnterpriseAddress(
	network CardanoNetworkType, verificationKey []byte,
) (*CardanoAddress, error) {
	paymentHash, err := GetKeyHashBytes(verificationKey)
	if err != nil {
		return nil, err
	}

	return CardanoAddressInfo{
		AddressType: EnterpriseAddress,
		Network:     network,
		Payment: &CardanoAddressPayload{
			Payload:  [KeyHashSize]byte(paymentHash),
			IsScript: false,
		},
	}.ToCardanoAddress()
}

func NewRewardAddress(
	network CardanoNetworkType, verificationKey []byte,
) (*CardanoAddress, error) {
	stakeHash, err := GetKeyHashBytes(verificationKey)
	if err != nil {
		return nil, err
	}

	return CardanoAddressInfo{
		AddressType: RewardAddress,
		Network:     network,
		Stake: &CardanoAddressPayload{
			Payload:  [KeyHashSize]byte(stakeHash),
			IsScript: false,
		},
	}.ToCardanoAddress()
}
