package auth

import (
	"errors"
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

// PasswordManager 密码管理器
type PasswordManager struct {
	minLength      int
	requireUpper   bool
	requireLower   bool
	requireDigit   bool
	requireSpecial bool
}

// NewPasswordManager 创建密码管理器
func NewPasswordManager() *PasswordManager {
	return &PasswordManager{
		minLength:      8,
		requireUpper:   true,
		requireLower:   true,
		requireDigit:   true,
		requireSpecial: false,
	}
}

// HashPassword 哈希密码
func (pm *PasswordManager) HashPassword(password string) (string, error) {
	if err := pm.ValidatePassword(password); err != nil {
		return "", err
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashedBytes), nil
}

// VerifyPassword 验证密码
func (pm *PasswordManager) VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// ValidatePassword 验证密码强度
func (pm *PasswordManager) ValidatePassword(password string) error {
	if len(password) < pm.minLength {
		return errors.New("密码长度至少8位")
	}

	if pm.requireUpper {
		matched, _ := regexp.MatchString(`[A-Z]`, password)
		if !matched {
			return errors.New("密码必须包含大写字母")
		}
	}

	if pm.requireLower {
		matched, _ := regexp.MatchString(`[a-z]`, password)
		if !matched {
			return errors.New("密码必须包含小写字母")
		}
	}

	if pm.requireDigit {
		matched, _ := regexp.MatchString(`[0-9]`, password)
		if !matched {
			return errors.New("密码必须包含数字")
		}
	}

	if pm.requireSpecial {
		matched, _ := regexp.MatchString(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`, password)
		if !matched {
			return errors.New("密码必须包含特殊字符")
		}
	}

	return nil
}

// GenerateRandomPassword 生成随机密码
func (pm *PasswordManager) GenerateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	password := make([]byte, length)
	for i := range password {
		password[i] = charset[i%len(charset)]
	}
	return string(password)
}
