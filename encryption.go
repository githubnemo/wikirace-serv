package main

import (
	"fmt"
	"crypto/cipher"
	"crypto/des"
	"encoding/base64"
	"math"
)

const PAGE_CIPHER_KEY_LENGTH = 8

// pad the input bytes and return the amount of padded bytes
func pad(in []byte, sz int) (padded []byte, bytes int) {
	padded = in

	if len(in)%sz != 0 {
		newLen := int(float64(sz) * math.Ceil(float64(len(in))/float64(sz)))
		padded = make([]byte, newLen)

		bytes = newLen - len(in)
		copy(padded, in)
	}

	return padded, bytes
}

type PageCipher struct{
	cipher.Block
}

func (p PageCipher) EncryptPage(page string) string {
	dst, padding := pad([]byte(page), p.BlockSize())

	p.Encrypt(dst, dst)

	return fmt.Sprintf("%d:%s", padding, base64.URLEncoding.EncodeToString(dst))
}

func (p PageCipher) DecryptPage(input string) string {
	var padding int
	var b64page string

	_, err := fmt.Sscanf(input, "%d:%s", &padding, &b64page)

	if err != nil {
		panic(err)
	}

	dst, err := base64.URLEncoding.DecodeString(b64page)

	if err != nil {
		panic(err)
	}

	p.Decrypt(dst, dst)

	return string(dst[:len(dst)-padding])
}

func NewPageCipher(key []byte) (*PageCipher, error) {
	cipher, err := des.NewCipher(key)

	if err != nil {
		return nil, err
	}

	return &PageCipher{cipher}, nil
}