package price_test

import (
	"encoding/base64"
	"math"
	"testing"

	"github.com/mechiru/ab/crypto/price"
	"gotest.tools/v3/assert"
)

func newCrypter() (*price.Crypter, error) {
	encKey, err := base64.URLEncoding.DecodeString("skU7Ax_NL5pPAFyKdkfZjZz2-VhIN8bjj1rVFOaJ_5o=")
	if err != nil {
		return nil, err
	}
	intKey, err := base64.URLEncoding.DecodeString("arO23ykdNqUQ5LEoQ0FVmPkBd7xB5CO89PDZlSjpFxo=")
	if err != nil {
		return nil, err
	}
	return price.NewCrypter(encKey, intKey)
}

func TestCrypterEncryptAndDecrypt(t *testing.T) {
	crypter, err := newCrypter()
	assert.NilError(t, err)

	for _, c := range []struct {
		in int64
	}{
		{100},
		{1900},
		{2700},
	} {
		encrypted, err := crypter.EncryptMicros(c.in)
		assert.NilError(t, err)

		got, err := crypter.DecryptValue(encrypted)
		assert.NilError(t, err)

		assert.Equal(t, got, float64(c.in)/math.Pow10(6))
	}
}

// https://developers.google.com/authorized-buyers/rtb/response-guide/decrypt-price#encododing
func TestCrypterDecode(t *testing.T) {
	crypter, err := newCrypter()
	assert.NilError(t, err)

	for _, c := range []struct {
		in   string
		want float64
	}{
		{"YWJjMTIzZGVmNDU2Z2hpN7fhCuPemCce_6msaw", 100},
		{"YWJjMTIzZGVmNDU2Z2hpN7fhCuPemCAWJRxOgA", 1900},
		{"YWJjMTIzZGVmNDU2Z2hpN7fhCuPemC32prpWWw", 2700},
	} {
		got, err := crypter.DecodeValue(c.in)
		assert.NilError(t, err)
		assert.Equal(t, got, c.want/math.Pow10(6))
	}
}
