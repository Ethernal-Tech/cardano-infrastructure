package wallet

import (
	"encoding/hex"
	"encoding/json"
	"sort"
)

const (
	PolicyScriptAtLeastType = "atLeast"
	PolicyScriptSigType     = "sig"
	PolicyScriptAfterType   = "after"
)

type PolicyScript struct {
	Type string `json:"type"`

	Required int            `json:"required,omitempty"`
	Scripts  []PolicyScript `json:"scripts,omitempty"`

	KeyHash string `json:"keyHash,omitempty"`
	Slot    uint64 `json:"slot,omitempty"`
}

type PolicyScriptOption func(*PolicyScript)

func NewPolicyScript(keyHashes []string, atLeastSignersCount int, options ...PolicyScriptOption) *PolicyScript {
	ps := &PolicyScript{
		Type:     PolicyScriptAtLeastType,
		Required: atLeastSignersCount,
	}

	for _, opt := range options {
		opt(ps)
	}

	scripts := make([]PolicyScript, len(keyHashes))
	if ps.Slot != 0 {
		scripts = make([]PolicyScript, len(keyHashes)+1)
		scripts[len(keyHashes)] = PolicyScript{
			Type: PolicyScriptAfterType,
			Slot: ps.Slot,
		}
	}

	for i, keyHash := range keyHashes {
		scripts[i] = PolicyScript{
			Type:    PolicyScriptSigType,
			KeyHash: keyHash,
		}
	}

	sort.Slice(scripts, func(i, j int) bool {
		return scripts[i].KeyHash < scripts[j].KeyHash
	})

	ps.Scripts = scripts

	return ps
}

// WithAfter sets the "after" slot condition for the policy script
func WithAfter(slot uint64) PolicyScriptOption {
	return func(ps *PolicyScript) {
		ps.Slot = slot
	}
}

func (ps PolicyScript) GetBytesJSON() ([]byte, error) {
	return json.MarshalIndent(ps, "", "  ")
}

func (ps PolicyScript) GetCount() (cnt int) {
	switch ps.Type {
	case PolicyScriptSigType:
		cnt = 1
	case "any":
		for _, x := range ps.Scripts {
			if subCnt := x.GetCount(); cnt < subCnt {
				cnt = subCnt
			}
		}
	case "all", PolicyScriptAtLeastType:
		for _, x := range ps.Scripts {
			cnt += x.GetCount()
		}
	}

	return cnt
}

// NewPolicyScriptBaseAddress returns base address for policy script IDs
func NewPolicyScriptBaseAddress(
	networkID CardanoNetworkType, policyID, stakePolicyID string,
) (*CardanoAddress, error) {
	policyIDBytes, err := hex.DecodeString(policyID)
	if err != nil {
		return nil, err
	}

	policyIDStakeBytes, err := hex.DecodeString(stakePolicyID)
	if err != nil {
		return nil, err
	}

	return CardanoAddressInfo{
		AddressType: BaseAddress,
		Network:     networkID,
		Payment: &CardanoAddressPayload{
			Payload:  [KeyHashSize]byte(policyIDBytes),
			IsScript: true,
		},
		Stake: &CardanoAddressPayload{
			Payload:  [KeyHashSize]byte(policyIDStakeBytes),
			IsScript: true,
		},
	}.ToCardanoAddress()
}

// NewPolicyScriptEnterpriseAddress returns enterprise address for policy script ID
func NewPolicyScriptEnterpriseAddress(
	networkID CardanoNetworkType, policyID string,
) (*CardanoAddress, error) {
	policyIDBytes, err := hex.DecodeString(policyID)
	if err != nil {
		return nil, err
	}

	return CardanoAddressInfo{
		AddressType: EnterpriseAddress,
		Network:     networkID,
		Payment: &CardanoAddressPayload{
			Payload:  [KeyHashSize]byte(policyIDBytes),
			IsScript: true,
		},
	}.ToCardanoAddress()
}

// NewPolicyScriptRewardAddress returns reward address for this policy script
func NewPolicyScriptRewardAddress(
	networkID CardanoNetworkType, policyID string,
) (*CardanoAddress, error) {
	policyIDBytes, err := hex.DecodeString(policyID)
	if err != nil {
		return nil, err
	}

	return CardanoAddressInfo{
		AddressType: RewardAddress,
		Network:     networkID,
		Stake: &CardanoAddressPayload{
			Payload:  [KeyHashSize]byte(policyIDBytes),
			IsScript: true,
		},
	}.ToCardanoAddress()
}
