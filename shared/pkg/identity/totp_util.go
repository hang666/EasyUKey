package identity

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTPConfig TOTP配置结构
type TOTPConfig struct {
	Secret    string        // TOTP密钥
	Issuer    string        // 签发者
	Account   string        // 账户名
	Period    uint          // 时间周期（秒）
	Digits    otp.Digits    // 验证码位数
	Algorithm otp.Algorithm // 哈希算法
}

// ParseTOTPURI 解析TOTP URI
func ParseTOTPURI(uri string) (*TOTPConfig, error) {
	if !strings.HasPrefix(uri, "otpauth://totp/") {
		return nil, errors.New("invalid TOTP URI scheme")
	}

	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	accountInfo := strings.TrimPrefix(u.Path, "/")
	accountParts := strings.SplitN(accountInfo, ":", 2)
	var issuer, account string
	if len(accountParts) == 2 {
		issuer = accountParts[0]
		account = accountParts[1]
	} else {
		account = accountParts[0]
	}

	params := u.Query()

	secret := params.Get("secret")
	if secret == "" {
		return nil, errors.New("missing secret in URI")
	}

	// Optional params
	period := uint(30)
	if p := params.Get("period"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			period = uint(parsed)
		}
	}

	digits := otp.DigitsSix
	if d := params.Get("digits"); d == "8" {
		digits = otp.DigitsEight
	}

	algorithm := otp.AlgorithmSHA1
	switch strings.ToUpper(params.Get("algorithm")) {
	case "SHA256":
		algorithm = otp.AlgorithmSHA256
	case "SHA512":
		algorithm = otp.AlgorithmSHA512
	case "MD5":
		algorithm = otp.AlgorithmMD5
	}

	return &TOTPConfig{
		Secret:    secret,
		Issuer:    issuer,
		Account:   account,
		Period:    period,
		Digits:    digits,
		Algorithm: algorithm,
	}, nil
}

// GenerateTOTPCode 使用配置生成验证码
func GenerateTOTPCode(cfg *TOTPConfig, at time.Time) (string, error) {
	return totp.GenerateCodeCustom(cfg.Secret, at, totp.ValidateOpts{
		Period:    cfg.Period,
		Skew:      1,
		Digits:    cfg.Digits,
		Algorithm: cfg.Algorithm,
	})
}

// VerifyTOTPCode 验证TOTP验证码
func VerifyTOTPCode(cfg *TOTPConfig, code string, at time.Time) (bool, error) {
	return totp.ValidateCustom(code, cfg.Secret, at, totp.ValidateOpts{
		Period:    cfg.Period,
		Skew:      1,
		Digits:    cfg.Digits,
		Algorithm: cfg.Algorithm,
	})
}

// GenerateTOTPSecretURI 生成TOTP密钥URI
func GenerateTOTPSecretURI(issuer string, account string) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: account,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", err
	}
	return key.URL(), nil
}
