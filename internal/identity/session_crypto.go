package identity

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strconv"

	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

type VersionedKey struct {
	Version  uint64
	Material []byte
}

type AESGCMCryptor struct {
	current uint64
	keys    map[uint64][32]byte
	entropy io.Reader
}

func NewAESGCMCryptor(keys []VersionedKey, entropy io.Reader) (*AESGCMCryptor, error) {
	if entropy == nil {
		entropy = rand.Reader
	}
	cryptor := &AESGCMCryptor{keys: make(map[uint64][32]byte), entropy: entropy}
	for _, key := range keys {
		if key.Version == 0 || len(key.Material) != 32 {
			return nil, errors.New("transaction encryption key is invalid")
		}
		if _, duplicate := cryptor.keys[key.Version]; duplicate {
			return nil, errors.New("transaction encryption key version is duplicated")
		}
		var material [32]byte
		copy(material[:], key.Material)
		cryptor.keys[key.Version] = material
		if key.Version > cryptor.current {
			cryptor.current = key.Version
		}
	}
	if cryptor.current == 0 {
		return nil, errors.New("transaction encryption keyring is empty")
	}
	return cryptor, nil
}

func (cryptor *AESGCMCryptor) Encrypt(plaintext []byte) ([]byte, uint64, error) {
	if cryptor == nil || len(plaintext) == 0 || len(plaintext) > 1024 {
		return nil, 0, errors.New("transaction plaintext is invalid")
	}
	key := cryptor.keys[cryptor.current]
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, 0, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, 0, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(cryptor.entropy, nonce); err != nil {
		return nil, 0, err
	}
	ciphertext := aead.Seal(nonce, nonce, plaintext, []byte("atlas-oidc-pkce-v1"))
	return ciphertext, cryptor.current, nil
}

func (cryptor *AESGCMCryptor) Decrypt(ciphertext []byte, version uint64) ([]byte, error) {
	if cryptor == nil || len(ciphertext) == 0 || len(ciphertext) > 2048 {
		return nil, errors.New("transaction ciphertext is invalid")
	}
	key, found := cryptor.keys[version]
	if !found {
		return nil, errors.New("transaction encryption key version is unavailable")
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < aead.NonceSize()+aead.Overhead() {
		return nil, errors.New("transaction ciphertext is truncated")
	}
	nonce := ciphertext[:aead.NonceSize()]
	return aead.Open(nil, nonce, ciphertext[aead.NonceSize():], []byte("atlas-oidc-pkce-v1"))
}

type HMACCSRFProtector struct {
	key [32]byte
}

func NewHMACCSRFProtector(material []byte) (*HMACCSRFProtector, error) {
	if len(material) != 32 {
		return nil, errors.New("CSRF key is invalid")
	}
	protector := &HMACCSRFProtector{}
	copy(protector.key[:], material)
	return protector, nil
}

func (protector *HMACCSRFProtector) Token(sessionID identifier.ID, rotationVersion int64) (string, error) {
	if protector == nil || sessionID.IsZero() || sessionID.Prefix() != "ses" || rotationVersion < 1 {
		return "", errors.New("CSRF token input is invalid")
	}
	mac := hmac.New(sha256.New, protector.key[:])
	_, _ = mac.Write([]byte(sessionID.String()))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(strconv.FormatInt(rotationVersion, 10)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
