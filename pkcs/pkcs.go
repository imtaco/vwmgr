package pkcs

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"regexp"

	"github.com/pkg/errors"
	"golang.org/x/crypto/pbkdf2"
)

const (
	ITERATIONS = 600_000
)

var (
	rxBWEnc = regexp.MustCompile("^2\\.([^|]+)\\|([^|]+)\\|(.+)$")
	rxBWPk  = regexp.MustCompile("^4\\.([^|]+)$")
)

func DeriveMasterKey(email, password string) []byte {
	return pbkdf2SHA256(
		[]byte(password),
		[]byte(email),
		ITERATIONS,
	)
}

func DerivePasswordHash(masterKey []byte, masterPasswod string) string {
	return Base64Encode(pbkdf2SHA256(
		masterKey,
		[]byte(masterPasswod),
		1,
	))
}

func HashPasswordHash(passwordHash string, salt []byte) []byte {
	return pbkdf2SHA256(
		[]byte(passwordHash),
		salt,
		ITERATIONS,
	)
}

func GenRSAKeyPair() ([]byte, []byte) {
	// Generate RSA Key Pair (2048-bit, E=65537)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		// should not happen
		panic(err)
	}
	// private key in PKCS8
	pkcs8key, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		panic(err)
	}
	// public key in SPKI
	spkiPub, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		panic(err)
	}

	return spkiPub, pkcs8key
}

func PublicKeyInfo(key []byte) (*rsa.PublicKey, error) {
	pubInf, err := x509.ParsePKIXPublicKey(key)
	if err != nil {
		return nil, errors.Wrap(err, "fail to parse public key")
	}
	if pk, ok := pubInf.(*rsa.PublicKey); !ok {
		return nil, errors.New("fail to convert public key")
	} else {
		return pk, nil
	}
}

func PrivateKeyInfo(key []byte) (*rsa.PrivateKey, error) {
	priInf, err := x509.ParsePKCS8PrivateKey(key)
	if err != nil {
		return nil, errors.Wrap(err, "fail to parse private key")
	}
	if pk, ok := priInf.(*rsa.PrivateKey); !ok {
		return nil, errors.New("fail to convert private key")
	} else {
		return pk, nil
	}
}

func BWSymEncrypt(key, plain []byte) string {
	encKey, macKey := deriveEncMacKey(key)
	iv := RandBytes(16)
	encrypted := ase256cbcEncrypt(plain, encKey, iv)
	maced := HMACSha256(macKey, append(iv, encrypted...))

	return fmt.Sprintf(
		"2.%s|%s|%s",
		Base64Encode(iv),
		Base64Encode(encrypted),
		Base64Encode(maced),
	)
}

func BWSymDecrypt(key []byte, cipher string) ([]byte, error) {
	matched := rxBWEnc.FindAllStringSubmatch(cipher, -1)
	if len(matched) == 0 {
		return nil, errors.New(("invalid Bitwarden key format"))
	}
	// decode base64
	b64Decoded, err := Base64DecodeMany(matched[0][1:]...)
	if err != nil {
		return nil, errors.Wrap(err, "fail to decode base64")
	}
	iv, encData, mac := b64Decoded[0], b64Decoded[1], b64Decoded[2]

	encKey, macKey := deriveEncMacKey(key)
	macData := iv
	macData = append(macData, encData...)
	expectedMac := HMACSha256(macKey, macData)

	if len(mac) != len(expectedMac) {
		return nil, errors.New("MAC length mismatch")
	}
	if !bytes.Equal(mac, expectedMac) {
		return nil, errors.New("MAC validation failed - wrong masterKey or tampered data")
	}

	plaintext, err := aes256cbcDecrypt(encData, encKey, iv)
	if err != nil {
		return nil, errors.Wrap(err, "fail to decrypt")
	}
	return plaintext, nil
}

func deriveEncMacKey(key []byte) (encKey []byte, macKey []byte) {
	if len(key) == 32 {
		encKey = hkdfExpand(key, "enc", 32)
		macKey = hkdfExpand(key, "mac", 32)
	} else {
		encKey = key[:32]
		macKey = key[32:]
	}
	return
}

func BWPKEncrypt(data []byte, pub *rsa.PublicKey) string {
	encrypted := pkEncrypt(data, pub)
	return fmt.Sprintf("4.%s", Base64Encode(encrypted))
}

func BWPKDecrypt(cipher string, pri *rsa.PrivateKey) ([]byte, error) {
	matched := rxBWPk.FindAllStringSubmatch(cipher, -1)
	if len(matched) == 0 {
		return nil, errors.New(("invalid Bitwarden key format"))
	}

	encData, err := Base64Decode(matched[0][1])
	if err != nil {
		return nil, errors.Wrap(err, "fail to parse encrypted data")
	}
	plaintext, err := pkDecrypt(encData, pri)
	if err != nil {
		return nil, errors.Wrap(err, "fail to decrypt with privateKey")
	}
	return plaintext, nil
}

func Base64Encode(data []byte) string {
	return base64.StdEncoding.
		WithPadding(base64.StdPadding).
		EncodeToString(data)
}

func Base64Decode(data string) ([]byte, error) {
	return base64.StdEncoding.
		WithPadding(base64.StdPadding).
		DecodeString(data)
}

func Base64DecodeMany(data ...string) ([][]byte, error) {
	var results [][]byte
	for _, d := range data {
		v, err := Base64Decode(d)
		if err != nil {
			return nil, err
		}
		results = append(results, v)
	}
	return results, nil
}

func pbkdf2SHA256(password, salt []byte, itr int) []byte {
	return pbkdf2.Key(
		password,
		salt,
		itr,
		32,
		sha256.New,
	)
}

// written by GPT
func hkdfExpand(prk []byte, info string, length int) []byte {
	hash := sha256.New
	hashLen := hash().Size()

	var result []byte
	var previousBlock []byte

	n := (length + hashLen - 1) / hashLen // Number of blocks to generate
	if n > 255 {
		panic("hkdfExpand: length too large")
	}

	for i := 1; i <= n; i++ {
		h := hmac.New(hash, prk)
		h.Write(previousBlock)
		h.Write([]byte(info))
		h.Write([]byte{byte(i)})
		previousBlock = h.Sum(nil)
		result = append(result, previousBlock...)
	}

	return result[:length]
}

func pkEncrypt(data []byte, pub *rsa.PublicKey) []byte {
	ciphertext, err := rsa.EncryptOAEP(sha1.New(), rand.Reader, pub, data, []byte(""))
	if err != nil {
		panic(err)
	}
	return ciphertext
}

func pkDecrypt(data []byte, pri *rsa.PrivateKey) ([]byte, error) {
	plaintext, err := rsa.DecryptOAEP(sha1.New(), rand.Reader, pri, data, []byte(""))
	if err != nil {
		return nil, errors.Wrap(err, "fail to decrypt by public key")
	}
	return plaintext, nil
}

// written by GPT
func ase256cbcEncrypt(plaintext, key, iv []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	mode := cipher.NewCBCEncrypter(block, iv)

	// PKCS#7 padding
	padLen := aes.BlockSize - (len(plaintext) % aes.BlockSize)
	padding := bytes.Repeat([]byte{byte(padLen)}, padLen)
	padded := append(plaintext, padding...)

	ciphertext := make([]byte, len(padded))
	mode.CryptBlocks(ciphertext, padded)
	return ciphertext
}

func aes256cbcDecrypt(ciphertext, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.New("fail to new AES cipher")
	}
	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	// Remove PKCS#7 padding
	padLen := int(plaintext[len(plaintext)-1])
	if padLen > aes.BlockSize || padLen == 0 {
		return nil, errors.New("invalid padding size")
	}
	for i := 0; i < padLen; i++ {
		if plaintext[len(plaintext)-1-i] != byte(padLen) {
			return nil, errors.New("invalid padding")
		}
	}
	return plaintext[:len(plaintext)-padLen], nil
}

func HMACSha256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(data)
	return mac.Sum(nil)
}

func RandBytes(n int) []byte {
	key := make([]byte, n)
	_, _ = rand.Read(key)
	return key
}
