package price

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
)

var micro = math.Pow10(6)

// https://developers.google.com/authorized-buyers/rtb/response-guide/decrypt-price
type Crypter struct {
	enc, int []byte
}

func NewCrypter(encKey, intKey []byte) (*Crypter, error) {
	if len(encKey) != 32 {
		return nil, fmt.Errorf("ab/crypto/price: encKey length must be 32: %d", len(encKey))
	}
	if len(intKey) != 32 {
		return nil, fmt.Errorf("ab/crypto/price: intKey length must be 32: %d", len(intKey))
	}
	return &Crypter{enc: encKey, int: intKey}, nil
}

func (c *Crypter) encrypt(value []byte, iv []byte) ([]byte, error) {
	// pad = hmac(e_key, iv) *first 8 bytes
	enc := hmac.New(sha1.New, c.enc)
	if _, err := enc.Write(iv); err != nil {
		return nil, fmt.Errorf("ab/crypto/price: hmac write error: %w", err)
	}
	pad := enc.Sum(nil)

	// enc_price = pad <xor> price
	encPrice := make([]byte, 8)
	for i := 0; i < 8; i++ {
		encPrice[i] = pad[i] ^ value[i]
	}

	// signature = hmac(i_key, price || iv) *first 4 bytes
	int := hmac.New(sha1.New, c.int)
	if _, err := int.Write(append(value, iv...)); err != nil {
		return nil, fmt.Errorf("ab/crypto/price: hmac write error: %w", err)
	}
	sig := int.Sum(nil)

	// final_message = iv || enc_price || signature
	msg := make([]byte, 0, 28)
	msg = append(msg, iv...)
	msg = append(msg, encPrice...)
	msg = append(msg, sig[0:4]...)
	return msg, nil
}

func (c *Crypter) Encrypt(value []byte) ([]byte, error) {
	if len(value) != 8 {
		return nil, fmt.Errorf("ab/crypto/price: byte array length must be 8: %d", len(value))
	}

	iv, err := genIV()
	if err != nil {
		return nil, err
	}

	return c.encrypt(value, iv)
}

func (c *Crypter) EncryptMicros(value int64) ([]byte, error) {
	price := make([]byte, 8)
	// https://github.com/google/openrtb-doubleclick/blob/master/doubleclick-core/src/main/java/com/google/doubleclick/crypto/DoubleClickCrypto.java#L440
	// `java.nio.ByteBuffer.wrap(plainData).putLong(PAYLOAD_BASE, priceValue)` writes in big endian format.
	binary.BigEndian.PutUint64(price, uint64(value))
	return c.Encrypt(price)
}

func (c *Crypter) EncryptValue(value float64) ([]byte, error) {
	return c.EncryptMicros(int64(value * micro))
}

func (c *Crypter) Decrypt(data []byte) ([]byte, error) {
	if len(data) != 28 {
		return nil, fmt.Errorf("ab/crypto/price: data length must be 28: %d", len(data))
	}

	// (iv, p, sig) = enc_price *Split up according to fixed lengths.
	iv, p, sig := data[:16], data[16:24], data[24:]

	// price_pad = hmac(e_key, iv)
	enc := hmac.New(sha1.New, c.enc)
	if _, err := enc.Write(iv); err != nil {
		return nil, fmt.Errorf("ab/crypto/price: hmac write error: %w", err)
	}
	pricePad := enc.Sum(nil)

	// price = p <xor> price_pad
	price := make([]byte, 8)
	for i := 0; i < 8; i++ {
		price[i] = p[i] ^ pricePad[i]
	}

	// conf_sig = hmac(i_key, price || iv)
	int := hmac.New(sha1.New, c.int)
	if _, err := int.Write(append(price, iv...)); err != nil {
		return nil, fmt.Errorf("ab/crypto/price: hmac write error: %w", err)
	}
	confSig := int.Sum(nil)

	// success = (conf_sig == sig)
	if !bytes.Equal(sig, confSig[:4]) {
		return nil, fmt.Errorf("ab/crypto/price: signature error: (%q, %q)", sig, confSig[:4])
	}

	return price, nil
}

func (c *Crypter) DecryptMicros(data []byte) (int64, error) {
	price, err := c.Decrypt(data)
	if err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(price)), nil
}

func (c *Crypter) DecryptValue(data []byte) (float64, error) {
	micros, err := c.DecryptMicros(data)
	if err != nil {
		return 0, err
	}
	return float64(micros) / micro, nil
}

func (c *Crypter) EncodeValue(value float64) (string, error) {
	data, err := c.EncryptValue(value)
	if err != nil {
		return "", err
	}
	return Encode(data), nil
}

func (c *Crypter) DecodeValue(cipher string) (float64, error) {
	data, err := Decode(cipher)
	if err != nil {
		return 0, err
	}
	return c.DecryptValue(data)
}

func Encode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func Decode(data string) ([]byte, error) {
	s, err := base64.RawURLEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("ab/crypto/price: cipher data decode error: %w", err)
	}
	return s, nil
}

// https://golang.org/src/crypto/cipher/example_test.go
func genIV() ([]byte, error) {
	iv := make([]byte, 16)
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("ab/crypto/price: initialization vector generate error: %w", err)
	}
	return iv, nil
}
