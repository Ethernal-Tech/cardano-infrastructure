package local

import (
	"os"
	"testing"

	"github.com/Ethernal-Tech/cardano-infrastructure/common"
	"github.com/Ethernal-Tech/cardano-infrastructure/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalSecretsManagerFactory(t *testing.T) {
	// Set up the expected folder structure
	workingDirectory, tempErr := os.MkdirTemp("", "local-secrets-manager")
	if tempErr != nil {
		t.Fatalf("Unable to instantiate local secrets manager directories, %v", tempErr)
	}

	// Set up a clean-up procedure
	t.Cleanup(func() {
		_ = os.RemoveAll(workingDirectory)
	})

	testTable := []struct {
		name          string
		config        *secrets.SecretsManagerConfig
		shouldSucceed bool
	}{
		{
			"Valid configuration with path info",
			&secrets.SecretsManagerConfig{
				Path: workingDirectory,
			},
			true,
		},
		{
			"Invalid configuration without path info",
			&secrets.SecretsManagerConfig{
				Path: "",
			},
			false,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			localSecretsManager, factoryErr := SecretsManagerFactory(testCase.config)
			if testCase.shouldSucceed {
				assert.NotNil(t, localSecretsManager)
				assert.NoError(t, factoryErr)
			} else {
				assert.Nil(t, localSecretsManager)
				assert.Error(t, factoryErr)
			}
		})
	}
}

// getLocalSecretsManager is a helper method for creating an instance of the
// local secrets manager
func getLocalSecretsManager(t *testing.T) secrets.SecretsManager {
	t.Helper()

	// Set up the expected folder structure
	workingDirectory, tempErr := os.MkdirTemp("", "local-secrets-manager")
	if tempErr != nil {
		t.Fatalf("Unable to instantiate local secrets manager directories, %v", tempErr)
	}

	setupErr := common.SetupDataDir(workingDirectory, []string{secrets.ConsensusFolderLocal, secrets.NetworkFolderLocal}, 0770)
	if setupErr != nil {
		t.Fatalf("Unable to instantiate local secrets manager directories, %v", setupErr)
	}

	// Set up a clean-up procedure
	t.Cleanup(func() {
		_ = os.RemoveAll(workingDirectory)
		_ = os.Remove(workingDirectory)
	})

	// Set up an instance of the local secrets manager
	baseConfig := &secrets.SecretsManagerConfig{
		Path: workingDirectory,
	}

	manager, factoryErr := SecretsManagerFactory(baseConfig)
	if factoryErr != nil {
		t.Fatalf("Unable to instantiate local secrets manager, %v", factoryErr)
	}

	assert.NotNil(t, manager)

	return manager
}

func TestLocalSecretsManager_GetSetRemoveSecret(
	t *testing.T,
) {
	testTable := []struct {
		name          string
		secretName    string
		secretValue   []byte
		shouldSucceed bool
	}{
		{
			"Validator key storage",
			secrets.ValidatorKey,
			[]byte("buvac"),
			true,
		},
		{
			"Networking key storage",
			secrets.NetworkKey,
			[]byte("kostolomac"),
			true,
		},
		{
			"Unsupported secret storage",
			"dummySecret",
			[]byte{1},
			false,
		},
		{
			"cardano secrets storage key",
			secrets.CardanoKeyLocalPrefix + "prime_cardano_key",
			[]byte{4, 16},
			true,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			// Get an instance of the secrets manager
			manager := getLocalSecretsManager(t)

			require.False(t, manager.HasSecret(testCase.secretName))

			// Set the secret
			err := manager.SetSecret(testCase.secretName, testCase.secretValue)
			if testCase.shouldSucceed {
				require.NoError(t, err)
				require.True(t, manager.HasSecret(testCase.secretName))

				val, err := manager.GetSecret(testCase.secretName)

				require.NoError(t, err)
				require.Equal(t, testCase.secretValue, val)

				err = manager.RemoveSecret(testCase.secretName)
				require.NoError(t, err)
			} else {
				require.Error(t, err)

				err = manager.RemoveSecret(testCase.secretName)
				require.Error(t, err)
			}
		})
	}
}
