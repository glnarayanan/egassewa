package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const cookieName = "gasmate_session"

var secretKey []byte

func init() {
	secretKey = make([]byte, 32)
	if _, err := rand.Read(secretKey); err != nil {
		panic("failed to generate session secret: " + err.Error())
	}
}

type Session struct {
	UserID   int64  `json:"u"`
	Username string `json:"n"`
	Email    string `json:"e"`
	Role     string `json:"r"` // customer | dealer | admin
}

func SetSession(w http.ResponseWriter, s Session) error {
	payload, err := json.Marshal(s)
	if err != nil {
		return err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	sig := sign(encoded)
	value := encoded + "." + sig
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400 * 7, // 7 days
	})
	return nil
}

func GetSession(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return nil, fmt.Errorf("no session cookie")
	}
	parts := strings.SplitN(cookie.Value, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("malformed cookie")
	}
	encoded, sig := parts[0], parts[1]
	if !hmac.Equal([]byte(sign(encoded)), []byte(sig)) {
		return nil, fmt.Errorf("invalid session signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode session: %w", err)
	}
	var s Session
	if err := json.Unmarshal(payload, &s); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &s, nil
}

func ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func sign(data string) string {
	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
