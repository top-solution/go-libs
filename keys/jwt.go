package keys

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"io/fs"

	jwt "github.com/dgrijalva/jwt-go"
)

type JWT struct {
	PublicKey  *rsa.PublicKey
	PrivateKey *rsa.PrivateKey
}

var ErrInvalidToken = errors.New("invalid token")

// ReadPublicKey reads and stores a public key used to verify JWTs
func (j *JWT) ReadPublicKey(FS fs.ReadFileFS, path string) error {
	verifyKeyByte, err := FS.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read public key: %w", err)
	}
	j.PublicKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyKeyByte)
	return err
}

// ReadPrivateKey reads and stores a private key used to sign JWTs
func (j *JWT) ReadPrivateKey(FS fs.ReadFileFS, path string) error {
	verifyKeyByte, err := FS.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read private key: %w", err)
	}
	j.PrivateKey, err = jwt.ParseRSAPrivateKeyFromPEM(verifyKeyByte)
	return err
}

func (j *JWT) TokenFromMap(data map[string]interface{}, name string) (string, error) {
	token := jwt.New(jwt.GetSigningMethod("RS256"))
	token.Claims = jwt.MapClaims(data)

	result, err := token.SignedString(j.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return result, nil
}

func (j *JWT) ParseAndValidateToken(tokenString string) (map[string]string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return j.PublicKey, nil
	})

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("%w: %s", ErrInvalidToken, err.Error())
	}

	result := map[string]string{}
	for k, d := range claims {
		result[k] = fmt.Sprintf("%v", d)
	}

	return result, nil
}
