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
	PolicyScriptBeforeType  = "before"
)

type PolicyScript struct {
	Type string `json:"type"`

	Required int            `json:"required,omitempty"`
	Scripts  []PolicyScript `json:"scripts,omitempty"`

	KeyHash string `json:"keyHash,omitempty"`
	Slot    uint64 `json:"slot,omitempty"`
}

type policyConfig struct {
	KeyHashes    []string
	BeforeScript *PolicyScript
	AfterScript  *PolicyScript
}

func (c *policyConfig) getScriptsCount() int {
	count := len(c.KeyHashes)

	if c.AfterScript != nil {
		count++
	}

	if c.BeforeScript != nil {
		count++
	}

	return count
}

type PolicyScriptOption func(*policyConfig)

func NewPolicyScript(keyHashes []string, atLeastSignersCount int, options ...PolicyScriptOption) *PolicyScript {
	config := &policyConfig{
		KeyHashes: keyHashes,
	}

	for _, opt := range options {
		opt(config)
	}

	// Build scripts dynamically using append
	scripts := make([]PolicyScript, 0, config.getScriptsCount())

	// Add signature scripts
	for _, keyHash := range config.KeyHashes {
		scripts = append(scripts, PolicyScript{
			Type:    PolicyScriptSigType,
			KeyHash: keyHash,
		})
	}

	// Add time constraint scripts
	if config.AfterScript != nil {
		scripts = append(scripts, *config.AfterScript)
	}

	if config.BeforeScript != nil {
		scripts = append(scripts, *config.BeforeScript)
	}

	// Sort scripts by key hash for consistency
	sort.Slice(scripts, func(i, j int) bool {
		return scripts[i].KeyHash < scripts[j].KeyHash
	})

	return &PolicyScript{
		Type:     PolicyScriptAtLeastType,
		Required: atLeastSignersCount,
		Scripts:  scripts,
	}
}

// WithAfter sets the "after" slot condition for the policy script
func WithAfter(slot uint64) PolicyScriptOption {
	return func(pc *policyConfig) {
		pc.AfterScript = &PolicyScript{
			Type: PolicyScriptAfterType,
			Slot: slot,
		}
	}
}

// WithBefore sets the "before" slot condition for the policy script
func WithBefore(slot uint64) PolicyScriptOption {
	return func(pc *policyConfig) {
		pc.BeforeScript = &PolicyScript{
			Type: PolicyScriptBeforeType,
			Slot: slot,
		}
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
