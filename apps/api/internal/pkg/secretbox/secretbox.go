// Package secretbox 加密要落库的敏感字段（TOTP 密钥、OIDC 客户端密钥）。
//
// 每种用途由主密钥（Security.DataEncryptionKey）经 HKDF 派生独立的 AES-256-GCM 密钥：
// 一种用途的密文不能在另一种用途下解开。密文格式为 "v1:" + base64(nonce ‖ ciphertext)，
// 版本前缀给以后更换算法留出余地。
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"

	"golang.org/x/crypto/hkdf"
)

const prefix = "v1:"

// ErrDecrypt 密文损坏、被篡改，或用另一把主密钥/另一种用途加密
var ErrDecrypt = errors.New("secretbox: cannot decrypt")

// Keyring 持有主密钥，按用途派生加密器。
type Keyring struct {
	master []byte
}

// NewKeyring 用主密钥创建；主密钥为空时返回错误（配置校验会在非开发模式下拦住）。
func NewKeyring(master string) (*Keyring, error) {
	if master == "" {
		return nil, errors.New("secretbox: empty master key")
	}
	return &Keyring{master: []byte(master)}, nil
}

// Box 某一种用途的加密器
type Box struct {
	aead cipher.AEAD
}

// Box 派生 purpose（如 "totp"）专用的加密器。
func (k *Keyring) Box(purpose string) (*Box, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(hkdf.New(sha256.New, k.master, nil, []byte("castor/secretbox/"+purpose)), key); err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Box{aead: aead}, nil
}

// Seal 加密明文
func (b *Box) Seal(plaintext string) (string, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := b.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return prefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Open 解密 Seal 的输出
func (b *Box) Open(ciphertext string) (string, error) {
	encoded, ok := strings.CutPrefix(ciphertext, prefix)
	if !ok {
		return "", ErrDecrypt
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(raw) < b.aead.NonceSize() {
		return "", ErrDecrypt
	}
	plain, err := b.aead.Open(nil, raw[:b.aead.NonceSize()], raw[b.aead.NonceSize():], nil)
	if err != nil {
		return "", ErrDecrypt
	}
	return string(plain), nil
}
