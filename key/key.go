package key

import (
	"crypto/rsa"
	"fmt"
	"io/fs"

	jwt "github.com/dgrijalva/jwt-go"
)

type JWT struct {
	FS   fs.ReadFileFS
	Name string
}

// GetPubKey return the public key
func (j *JWT) GetPubKey() (*rsa.PublicKey, error) {
	verifyKeyByte, err := j.FS.ReadFile(j.Name + ".rsa.pub")
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}
	return jwt.ParseRSAPublicKeyFromPEM(verifyKeyByte)
}

// GetPrivateKey return the private key
func (j *JWT) GetPrivateKey() (*rsa.PrivateKey, error) {
	signKeyByte, err := j.FS.ReadFile(j.Name + ".rsa")
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	return jwt.ParseRSAPrivateKeyFromPEM(signKeyByte)
}

func (j *JWT) GetNewToken(data map[string]interface{}, name string) (string, error) {
	key, err := j.GetPrivateKey()
	if err != nil {
		return "", fmt.Errorf("get private key: %w", err)
	}
	token := jwt.New(jwt.GetSigningMethod("RS256"))
	token.Claims = jwt.MapClaims(data)

	result, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return result, nil
}

func (j *JWT) GetParsedToken(tokenString string) (map[string]string, error) {
	key, err := j.GetPubKey()
	if err != nil {
		return nil, fmt.Errorf("get public key: %w", err)
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return key, nil
	})

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("token not valid: %w", err)
	}

	result := map[string]string{}
	for k, d := range claims {
		result[k] = fmt.Sprintf("%v", d)
	}

	return result, nil
}
