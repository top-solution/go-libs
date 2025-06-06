package keys

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	jwt.StandardClaims
	Role       []string               `json:"roles,omitempty"`
	Username   string                 `json:"username,omitempty"`
	Firstname  string                 `json:"firstname,omitempty"`
	Lastname   string                 `json:"lastname,omitempty"`
	AppID      string                 `json:"appID,omitempty"`
	AppRoleMap map[string][]string    `json:"appRoleMap,omitempty"`
	Extra      map[string]interface{} `json:"extra,omitempty"`
	Email      string                 `json:"email,omitempty"`
}

type JWT struct {
	PublicKey  *rsa.PublicKey
	PrivateKey *rsa.PrivateKey
}

var ErrInvalidToken = errors.New("invalid token")

// StartPublicKeyRefresh launchs a goroutine that reads and stores a public key from a public URL, refreshing it periodically
func (j *JWT) StartPublicKeyRefresh(ctx context.Context, url string) {
	go func() {
		// Fetch public key right away
		err := j.ReadPublicKeyFromURL(url)
		if err != nil {
			slog.Error("StartPublicKeyRefresh: unable to read public key", "url", url, "err", err)
		}
		// 	Fetch public key once a day
		for range time.Tick(time.Hour * 24) {
			err := j.ReadPublicKeyFromURL(url)
			if err != nil {
				slog.Error("StartPublicKeyRefresh: unable to read public key", "url", url, "err", err)
			}
		}
	}()
}

// ReadPublicKey reads and stores a public key from a public URL
func (j *JWT) ReadPublicKeyFromURL(url string) error {

	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("create req: %w", err)
	}

	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("read pubkey from url: %w", err)
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)

	}
	if response.StatusCode != 200 {
		return fmt.Errorf("unable to read key: %d", response.StatusCode)
	}

	j.PublicKey, err = jwt.ParseRSAPublicKeyFromPEM([]byte(body))
	return err
}

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

func (j *JWT) TokenFromClaims(claims Claims) (string, error) {
	token := jwt.New(jwt.GetSigningMethod("RS256"))
	token.Claims = claims

	result, err := token.SignedString(j.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return result, nil
}

func (j *JWT) TokenFromMap(data map[string]interface{}) (string, error) {
	token := jwt.New(jwt.GetSigningMethod("RS256"))
	token.Claims = jwt.MapClaims(data)

	result, err := token.SignedString(j.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return result, nil
}

func (j *JWT) ParseAndValidateToken(tokenString string) (claims Claims, err error) {
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return j.PublicKey, nil
	}, jwt.WithJSONNumber())
	if err != nil {
		return claims, fmt.Errorf("%w: %s", ErrInvalidToken, err.Error())
	}

	if !token.Valid {
		return claims, ErrInvalidToken
	}

	return claims, nil
}
