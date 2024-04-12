package helper

import (
	"errors"

	"github.com/Ethernal-Tech/cardano-infrastructure/secrets"
	"github.com/Ethernal-Tech/cardano-infrastructure/secrets/awsssm"
	"github.com/Ethernal-Tech/cardano-infrastructure/secrets/gcpssm"
	"github.com/Ethernal-Tech/cardano-infrastructure/secrets/hashicorpvault"
	"github.com/Ethernal-Tech/cardano-infrastructure/secrets/local"
)

// CreateSecretsManager returns the secrets manager from the provided config
func CreateSecretsManager(config *secrets.SecretsManagerConfig) (secrets.SecretsManager, error) {
	switch config.Type {
	case secrets.HashicorpVault:
		return hashicorpvault.SecretsManagerFactory(config)
	case secrets.AWSSSM:
		return awsssm.SecretsManagerFactory(config)
	case secrets.GCPSSM:
		return gcpssm.SecretsManagerFactory(config)
	case secrets.Local:
		return local.SecretsManagerFactory(config)
	default:
		return nil, errors.New("unsupported secrets manager")
	}
}

func GetValidatorKey(config *secrets.SecretsManagerConfig) ([]byte, error) {
	mngr, err := CreateSecretsManager(config)
	if err != nil {
		return nil, err
	}

	return mngr.GetSecret(secrets.ValidatorKey)
}
