package helper

import (
	"errors"

	"github.com/Ethernal-Tech/cardano-infrastructure/secrets"
	"github.com/Ethernal-Tech/cardano-infrastructure/secrets/awsssm"
	"github.com/Ethernal-Tech/cardano-infrastructure/secrets/gcpssm"
	"github.com/Ethernal-Tech/cardano-infrastructure/secrets/hashicorpvault"
	"github.com/Ethernal-Tech/cardano-infrastructure/secrets/local"
	"github.com/hashicorp/go-hclog"
)

// SetupLocalSecretsManager is a helper method for boilerplate local secrets manager setup
func SetupLocalSecretsManager(dataDir string) (secrets.SecretsManager, error) {
	return local.SecretsManagerFactory(
		nil, // Local secrets manager doesn't require a config
		&secrets.SecretsManagerParams{
			Logger: hclog.NewNullLogger(),
			Extra: map[string]interface{}{
				secrets.Path: dataDir,
			},
		},
	)
}

// setupHashicorpVault is a helper method for boilerplate hashicorp vault secrets manager setup
func setupHashicorpVault(
	secretsConfig *secrets.SecretsManagerConfig,
) (secrets.SecretsManager, error) {
	return hashicorpvault.SecretsManagerFactory(
		secretsConfig,
		&secrets.SecretsManagerParams{
			Logger: hclog.NewNullLogger(),
		},
	)
}

// setupAWSSSM is a helper method for boilerplate aws ssm secrets manager setup
func setupAWSSSM(
	secretsConfig *secrets.SecretsManagerConfig,
) (secrets.SecretsManager, error) {
	return awsssm.SecretsManagerFactory(
		secretsConfig,
		&secrets.SecretsManagerParams{
			Logger: hclog.NewNullLogger(),
		},
	)
}

// setupGCPSSM is a helper method for boilerplate Google Cloud Computing secrets manager setup
func setupGCPSSM(
	secretsConfig *secrets.SecretsManagerConfig,
) (secrets.SecretsManager, error) {
	return gcpssm.SecretsManagerFactory(
		secretsConfig,
		&secrets.SecretsManagerParams{
			Logger: hclog.NewNullLogger(),
		},
	)
}

// InitCloudSecretsManager returns the cloud secrets manager from the provided config
func InitCloudSecretsManager(secretsConfig *secrets.SecretsManagerConfig) (secrets.SecretsManager, error) {
	var secretsManager secrets.SecretsManager

	switch secretsConfig.Type {
	case secrets.HashicorpVault:
		vault, err := setupHashicorpVault(secretsConfig)
		if err != nil {
			return secretsManager, err
		}

		secretsManager = vault
	case secrets.AWSSSM:
		AWSSSM, err := setupAWSSSM(secretsConfig)
		if err != nil {
			return secretsManager, err
		}

		secretsManager = AWSSSM
	case secrets.GCPSSM:
		GCPSSM, err := setupGCPSSM(secretsConfig)
		if err != nil {
			return secretsManager, err
		}

		secretsManager = GCPSSM
	default:
		return secretsManager, errors.New("unsupported secrets manager")
	}

	return secretsManager, nil
}
