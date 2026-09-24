package agent

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
)

const encryptedConfigHeader = "TNLXAGENTENC1\n"

// This protects the config from casual viewing on disk. Set
// TNLX_AGENT_CONFIG_KEY to replace the built-in project key.
const defaultConfigKey = "tnlx-upgrade-agent-config-v1"

func IsEncryptedConfig(data []byte) bool {
	return bytes.HasPrefix(bytes.TrimSpace(data), []byte(encryptedConfigHeader[:len(encryptedConfigHeader)-1]))
}

func EncryptConfigBytes(plain []byte) ([]byte, error) {
	block, err := aes.NewCipher(configKey())
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	sealed := gcm.Seal(nonce, nonce, plain, nil)
	out := make([]byte, 0, len(encryptedConfigHeader)+base64.StdEncoding.EncodedLen(len(sealed))+1)
	out = append(out, encryptedConfigHeader...)
	out = base64.StdEncoding.AppendEncode(out, sealed)
	out = append(out, '\n')
	return out, nil
}

func DecryptConfigBytes(encrypted []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(encrypted)
	header := []byte(encryptedConfigHeader[:len(encryptedConfigHeader)-1])
	if !bytes.HasPrefix(trimmed, header) {
		return nil, errors.New("agent config is not encrypted")
	}
	payload := bytes.TrimSpace(bytes.TrimPrefix(trimmed, header))
	sealed := make([]byte, base64.StdEncoding.DecodedLen(len(payload)))
	n, err := base64.StdEncoding.Decode(sealed, payload)
	if err != nil {
		return nil, fmt.Errorf("decode encrypted agent config: %w", err)
	}
	sealed = sealed[:n]

	block, err := aes.NewCipher(configKey())
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(sealed) < gcm.NonceSize() {
		return nil, errors.New("encrypted agent config is too short")
	}
	nonce := sealed[:gcm.NonceSize()]
	ciphertext := sealed[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt agent config: %w", err)
	}
	return plain, nil
}

func configKey() []byte {
	keyText := os.Getenv("TNLX_AGENT_CONFIG_KEY")
	if keyText == "" {
		keyText = defaultConfigKey
	}
	sum := sha256.Sum256([]byte(keyText))
	return sum[:]
}
