package logic

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chihqiang/infra-go/hash"

	"chihqiang/dskpanel/config"
)

// ErrInvalidCredentials 账号或密码错误。
var ErrInvalidCredentials = errors.New("invalid username or password")

// AuthLogic 登录鉴权逻辑。
// 单账号，用户名密码来自配置文件（auth.username / auth.password），不建用户表。
type AuthLogic struct {
	cfg config.Auth
}

// NewAuthLogic 创建登录鉴权逻辑。
func NewAuthLogic(cfg config.Auth) *AuthLogic {
	return &AuthLogic{cfg: cfg}
}

// tokenPayload token 载荷。
type tokenPayload struct {
	Username string `json:"username"`
	ExpireAt int64  `json:"expire_at"` // unix 秒
}

// LoginRequest 登录请求。
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResult 登录结果。
type LoginResult struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	ExpireAt int64  `json:"expire_at"`
}

// Login 校验账号密码，签发 token。
func (l *AuthLogic) Login(req *LoginRequest) (*LoginResult, error) {
	// 校验账号密码（支持 bcrypt 哈希或明文）。
	ok := l.verifyAccount(req.Username, req.Password)
	if !ok {
		return nil, ErrInvalidCredentials
	}

	expireAt := time.Now().Add(l.cfg.TokenTTL).Unix()
	payload := tokenPayload{Username: req.Username, ExpireAt: expireAt}
	token, err := l.sign(payload)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		Token:    token,
		Username: req.Username,
		ExpireAt: expireAt,
	}, nil
}

// Verify 校验 token，返回用户名。校验失败返回错误。
func (l *AuthLogic) Verify(token string) (string, error) {
	if token == "" {
		return "", ErrInvalidCredentials
	}
	payload, err := l.parse(token)
	if err != nil {
		return "", err
	}
	if time.Now().Unix() > payload.ExpireAt {
		return "", errors.New("token expired")
	}
	return payload.Username, nil
}

// verifyAccount 校验单账号用户名与密码。
func (l *AuthLogic) verifyAccount(username, password string) bool {
	if l.cfg.Username != username {
		return false
	}
	// 支持 bcrypt 哈希或明文。
	if strings.HasPrefix(l.cfg.Password, "$2a$") || strings.HasPrefix(l.cfg.Password, "$2b$") {
		return hash.BcryptMatch(l.cfg.Password, password)
	}
	return hmac.Equal([]byte(l.cfg.Password), []byte(password))
}

// sign 生成 token：base64url(payload) + "." + hmac-sha256 签名。
func (l *AuthLogic) sign(payload tokenPayload) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(raw)
	sig := l.hmac(body)
	return fmt.Sprintf("%s.%s", body, sig), nil
}

// parse 解析并校验 token。
func (l *AuthLogic) parse(token string) (*tokenPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, ErrInvalidCredentials
	}
	body, sig := parts[0], parts[1]
	if !hmac.Equal([]byte(l.hmac(body)), []byte(sig)) {
		return nil, ErrInvalidCredentials
	}
	raw, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	var payload tokenPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, ErrInvalidCredentials
	}
	return &payload, nil
}

// hmac 计算 token 签名。
func (l *AuthLogic) hmac(body string) string {
	mac := hmac.New(sha256.New, []byte(l.cfg.Secret))
	mac.Write([]byte(body))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
