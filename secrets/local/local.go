package local

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Ethernal-Tech/cardano-infrastructure/common"
	"github.com/Ethernal-Tech/cardano-infrastructure/secrets"
)

// LocalSecretsManager is a SecretsManager that
// stores secrets locally on disk
type LocalSecretsManager struct {
	// Path to the base working directory
	path string

	// Map of known secrets and their paths
	secretPathMap map[string]string

	// Mux for the secretPathMap
	secretPathMapLock sync.RWMutex
}

// SecretsManagerFactory implements the factory method
func SecretsManagerFactory(
	config *secrets.SecretsManagerConfig,
) (secrets.SecretsManager, error) {
	if config.Path == "" {
		return nil, errors.New("no path specified for local secrets manager")
	}

	// Set up the base object
	localManager := &LocalSecretsManager{
		secretPathMap: make(map[string]string),
		path:          config.Path,
	}

	// Run the initial setup
	if err := localManager.Setup(); err != nil {
		return nil, err
	}

	return localManager, nil
}

// Setup sets up the local SecretsManager
func (l *LocalSecretsManager) Setup() error {
	// The local SecretsManager initially handles only the
	// validator and networking private keys
	l.secretPathMapLock.Lock()
	defer l.secretPathMapLock.Unlock()

	subDirectories := []string{
		secrets.ConsensusFolderLocal, secrets.NetworkFolderLocal, secrets.CardanoFolderLocal, secrets.OtherFolderLocal}

	// Set up the local directories
	if err := common.SetupDataDir(l.path, subDirectories, 0750); err != nil {
		return err
	}

	// baseDir/consensus/validator.key
	l.secretPathMap[secrets.ValidatorKey] = filepath.Join(
		l.path,
		secrets.ConsensusFolderLocal,
		secrets.ValidatorKeyLocal,
	)

	// baseDir/consensus/validator-bls.key
	l.secretPathMap[secrets.ValidatorBLSKey] = filepath.Join(
		l.path,
		secrets.ConsensusFolderLocal,
		secrets.ValidatorBLSKeyLocal,
	)

	// baseDir/libp2p/libp2p.key
	l.secretPathMap[secrets.NetworkKey] = filepath.Join(
		l.path,
		secrets.NetworkFolderLocal,
		secrets.NetworkKeyLocal,
	)

	// baseDir/cardano/
	l.secretPathMap[secrets.CardanoKeyLocalPrefix] = filepath.Join(
		l.path,
		secrets.CardanoFolderLocal,
	)

	// baseDir/evm/
	l.secretPathMap[secrets.OtherKeyLocalPrefix] = filepath.Join(
		l.path,
		secrets.OtherFolderLocal,
	)

	return nil
}

// GetSecret gets the local SecretsManager's secret from disk
func (l *LocalSecretsManager) GetSecret(name string) ([]byte, error) {
	name, fileName := l.handleCardanoSecretName(name)

	l.secretPathMapLock.RLock()
	secretPath, ok := l.secretPathMap[name]
	l.secretPathMapLock.RUnlock()

	if !ok {
		return nil, secrets.ErrSecretNotFound
	}

	secretPath = filepath.Join(secretPath, fileName)

	// Read the secret from disk
	secret, err := os.ReadFile(secretPath)
	if err != nil {
		return nil, fmt.Errorf(
			"unable to read secret from disk (%s), %w",
			secretPath,
			err,
		)
	}

	return secret, nil
}

// SetSecret saves the local SecretsManager's secret to disk
func (l *LocalSecretsManager) SetSecret(name string, value []byte) error {
	// If the data directory is not specified, skip write
	if l.path == "" {
		return nil
	}

	name, fileName := l.handleCardanoSecretName(name)

	l.secretPathMapLock.Lock()
	secretPath, ok := l.secretPathMap[name]
	l.secretPathMapLock.Unlock()

	if !ok {
		return secrets.ErrSecretNotFound
	}

	secretPath = filepath.Join(secretPath, fileName)

	// Checks for existing secret
	if common.FileExists(secretPath) {
		return fmt.Errorf("%s already initialized", secretPath)
	}

	// Write the secret to disk
	if err := common.SaveFileSafe(secretPath, value, 0440); err != nil {
		return fmt.Errorf(
			"unable to write secret to disk (%s), %w",
			secretPath,
			err,
		)
	}

	return nil
}

// HasSecret checks if the secret is present on disk
func (l *LocalSecretsManager) HasSecret(name string) bool {
	_, err := l.GetSecret(name)

	return err == nil
}

// RemoveSecret removes the local SecretsManager's secret from disk
func (l *LocalSecretsManager) RemoveSecret(name string) error {
	name, fileName := l.handleCardanoSecretName(name)

	l.secretPathMapLock.Lock()
	secretPath, ok := l.secretPathMap[name]
	defer l.secretPathMapLock.Unlock()

	if !ok {
		return secrets.ErrSecretNotFound
	}

	secretPath = filepath.Join(secretPath, fileName)

	// we do not need to remove key from secretPathMap: delete(l.secretPathMap, name)

	if removeErr := os.Remove(secretPath); removeErr != nil {
		return fmt.Errorf("unable to remove secret, %w", removeErr)
	}

	return nil
}

func (l *LocalSecretsManager) handleCardanoSecretName(name string) (string, string) {
	if strings.HasPrefix(name, secrets.CardanoKeyLocalPrefix) {
		return secrets.CardanoKeyLocalPrefix,
			strings.Replace(
				strings.TrimPrefix(name, secrets.CardanoKeyLocalPrefix),
				"_key", ".key", 1,
			)
	} else if strings.HasPrefix(name, secrets.OtherKeyLocalPrefix) {
		return secrets.OtherKeyLocalPrefix,
			strings.Replace(
				strings.TrimPrefix(name, secrets.OtherKeyLocalPrefix),
				"_key", ".key", 1,
			)
	}

	return name, ""
}
