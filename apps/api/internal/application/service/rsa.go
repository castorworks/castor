package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/castorworks/castor/internal/pkg/constant"
	"github.com/hyperits/gosuite/logger"
	"github.com/redis/go-redis/v9"
)

// RSA 密钥相关配置
const (
	// RSAKeyExpiry RSA 密钥对过期时间（24小时）
	RSAKeyExpiry = 24 * time.Hour
	// RSAKeyGracePeriod 旧密钥保留时间（用于解密正在传输中的数据）
	RSAKeyGracePeriod = 1 * time.Hour
	// RSAPreviousKeyPairID 上一个 RSA 密钥对 ID 的 Redis key
	RSAPreviousKeyPairID = "rsa:previous_keypair_id"
	// encryptedPrivateKeyPrefix 加密私钥的前缀标记，用于区分加密/明文
	encryptedPrivateKeyPrefix = "enc:"
)

// RsaService RSA 服务接口
type RsaService interface {
	GenerateKeyPair(ctx context.Context) (string, error)
	GetPublicKey(ctx context.Context) (string, error)
	Decrypt(ctx context.Context, encryptedData string) (string, error)
	RotateKeyPair(ctx context.Context) (string, error)
}

type rsaService struct {
	redisClient redis.UniversalClient
	aesKey      []byte // AES-256 key derived from PrivateKeySecret, or FallbackSecret when unset (nil if both empty)
}

// NewRsaService 创建 RSA 服务
func NewRsaService(redisClient redis.UniversalClient, keys RsaKeyConfig) RsaService {
	svc := &rsaService{redisClient: redisClient}
	// 优先使用专用密钥；未配置时降级为 JWT 密钥派生（防御深度，非最优但优于明文存储）
	secret := keys.PrivateKeySecret
	if secret == "" {
		secret = keys.FallbackSecret
	}
	if secret != "" {
		hash := sha256.Sum256([]byte(secret))
		svc.aesKey = hash[:]
	}
	return svc
}

// encryptPrivateKey 使用 AES-256-GCM 加密私钥 PEM，返回前缀标记后的 base64 编码
func (svc *rsaService) encryptPrivateKey(plainPEM string) (string, error) {
	key := svc.aesKey
	if key == nil {
		// 无密钥可用，不加密（向后兼容）
		return plainPEM, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plainPEM), nil)
	return encryptedPrivateKeyPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptPrivateKey 解密私钥。如果数据以 encryptedPrivateKeyPrefix 开头则解密，
// 否则视为明文直接返回（向后兼容已存储的明文密钥）。
func (svc *rsaService) decryptPrivateKey(stored string) (string, error) {
	// 检查是否加密存储
	if len(stored) <= len(encryptedPrivateKeyPrefix) ||
		stored[:len(encryptedPrivateKeyPrefix)] != encryptedPrivateKeyPrefix {
		// 明文存储（旧数据或无加密配置），直接返回
		return stored, nil
	}

	cipherB64 := stored[len(encryptedPrivateKeyPrefix):]
	ciphertext, err := base64.StdEncoding.DecodeString(cipherB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted private key: %w", err)
	}

	key := svc.aesKey
	if key == nil {
		return "", errors.New("no encryption key available to decrypt stored private key")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt private key (key may have rotated): %w", err)
	}

	return string(plaintext), nil
}

func (svc *rsaService) GenerateKeyPair(ctx context.Context) (string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", fmt.Errorf("failed to generate RSA key pair: %v", err)
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %v", err)
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	keyPairID := fmt.Sprintf("rsa_keypair_%d", time.Now().UnixNano())

	// 加密私钥后再存储到 Redis
	storedPrivateKey, err := svc.encryptPrivateKey(string(privateKeyPEM))
	if err != nil {
		return "", fmt.Errorf("failed to encrypt private key: %v", err)
	}

	pipe := svc.redisClient.Pipeline()
	pipe.Set(ctx, fmt.Sprintf("%s:private", keyPairID), storedPrivateKey, RSAKeyExpiry+RSAKeyGracePeriod)
	pipe.Set(ctx, fmt.Sprintf("%s:public", keyPairID), string(publicKeyPEM), RSAKeyExpiry+RSAKeyGracePeriod)
	pipe.Set(ctx, constant.RSA_CURRENT_KEY_PAIR_ID, keyPairID, RSAKeyExpiry)

	if _, err := pipe.Exec(ctx); err != nil {
		return "", fmt.Errorf("failed to store key pair in Redis: %v", err)
	}

	logger.Infof("Generated new RSA key pair: %s, expires in: %v", keyPairID, RSAKeyExpiry)
	return string(publicKeyPEM), nil
}

// RotateKeyPair 轮换 RSA 密钥对
func (svc *rsaService) RotateKeyPair(ctx context.Context) (string, error) {
	// 获取当前密钥对 ID，将其设为上一个密钥对
	currentKeyPairID, err := svc.redisClient.Get(ctx, constant.RSA_CURRENT_KEY_PAIR_ID).Result()
	if err != nil && err != redis.Nil {
		return "", fmt.Errorf("failed to get current key pair ID: %v", err)
	}

	if currentKeyPairID != "" {
		// 保存上一个密钥对 ID，用于解密正在传输中的数据
		if err := svc.redisClient.Set(ctx, RSAPreviousKeyPairID, currentKeyPairID, RSAKeyGracePeriod).Err(); err != nil {
			logger.Warnf("Failed to save previous key pair ID: %v", err)
		}
	}

	// 生成新密钥对
	return svc.GenerateKeyPair(ctx)
}

func (svc *rsaService) GetPublicKey(ctx context.Context) (string, error) {
	keyPairID, err := svc.redisClient.Get(ctx, constant.RSA_CURRENT_KEY_PAIR_ID).Result()
	if err != nil {
		if err == redis.Nil {
			// 当前密钥对不存在或已过期，生成新密钥对
			return svc.GenerateKeyPair(ctx)
		}
		return "", fmt.Errorf("failed to get current key pair ID: %v", err)
	}

	publicKey, err := svc.redisClient.Get(ctx, fmt.Sprintf("%s:public", keyPairID)).Result()
	if err != nil {
		if err == redis.Nil {
			// 公钥不存在，生成新密钥对
			return svc.GenerateKeyPair(ctx)
		}
		return "", fmt.Errorf("failed to get public key: %v", err)
	}

	if _, err := svc.decryptUsablePrivateKey(ctx, keyPairID); err != nil {
		logger.Warnf("Current RSA key pair %s is unusable, regenerating: %v", keyPairID, err)
		return svc.GenerateKeyPair(ctx)
	}

	return publicKey, nil
}

func (svc *rsaService) Decrypt(ctx context.Context, encryptedData string) (string, error) {
	// 首先尝试使用当前密钥对解密
	keyPairID, err := svc.redisClient.Get(ctx, constant.RSA_CURRENT_KEY_PAIR_ID).Result()
	if err != nil && err != redis.Nil {
		return "", fmt.Errorf("failed to get current key pair ID: %v", err)
	}

	if keyPairID != "" {
		result, err := svc.decryptWithKeyPair(ctx, keyPairID, encryptedData)
		if err == nil {
			return result, nil
		}
		logger.Debugf("Failed to decrypt with current key pair %s: %v", keyPairID, err)
	}

	// 如果当前密钥解密失败，尝试使用上一个密钥对解密（处理轮换期间的请求）
	previousKeyPairID, err := svc.redisClient.Get(ctx, RSAPreviousKeyPairID).Result()
	if err != nil && err != redis.Nil {
		return "", fmt.Errorf("failed to get previous key pair ID: %v", err)
	}

	if previousKeyPairID != "" && previousKeyPairID != keyPairID {
		result, err := svc.decryptWithKeyPair(ctx, previousKeyPairID, encryptedData)
		if err == nil {
			return result, nil
		}
		logger.Debugf("Failed to decrypt with previous key pair %s: %v", previousKeyPairID, err)
	}

	return "", errors.New("failed to decrypt data with any available key pair")
}

// decryptWithKeyPair 使用指定密钥对解密数据
func (svc *rsaService) decryptWithKeyPair(ctx context.Context, keyPairID string, encryptedData string) (string, error) {
	privateKey, err := svc.decryptUsablePrivateKey(ctx, keyPairID)
	if err != nil {
		return "", err
	}

	encryptedBytes, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted data: %v", err)
	}

	decryptedBytes, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, encryptedBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt data: %v", err)
	}

	return string(decryptedBytes), nil
}

func (svc *rsaService) decryptUsablePrivateKey(ctx context.Context, keyPairID string) (*rsa.PrivateKey, error) {
	storedPrivateKey, err := svc.redisClient.Get(ctx, fmt.Sprintf("%s:private", keyPairID)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get private key: %v", err)
	}

	// 解密存储的私钥（如果是明文存储则直接使用）
	privateKeyPEM, err := svc.decryptPrivateKey(storedPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt stored private key: %v", err)
	}

	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, errors.New("failed to decode private key PEM")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	return privateKey, nil
}
