package main

import (
	"crypto/aes"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ---- AES-128-ECB（PHP openssl aes-256-ecb + 16字节 key 实际为 AES-128-ECB / PKCS7）----
func aesECBDecrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("invalid ciphertext length")
	}
	plain := make([]byte, len(ciphertext))
	for i := 0; i < len(ciphertext); i += aes.BlockSize {
		block.Decrypt(plain[i:i+aes.BlockSize], ciphertext[i:i+aes.BlockSize])
	}
	// PKCS7 unpad
	if len(plain) == 0 {
		return nil, errors.New("empty plaintext")
	}
	pad := int(plain[len(plain)-1])
	if pad == 0 || pad > aes.BlockSize || pad > len(plain) {
		return plain, nil // 无 padding 时直接返回
	}
	for i := len(plain) - pad; i < len(plain); i++ {
		if plain[i] != byte(pad) {
			return plain, nil
		}
	}
	return plain[:len(plain)-pad], nil
}

func aesDecryptString(encrypted, keyStr string) (string, bool) {
	decoded, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", false
	}
	// 与 PHP openssl_decrypt($data, 'aes-256-ecb', $AES_KEY, OPENSSL_RAW_DATA) 一致：
	// key 是 32 字节字符串，直接当 AES-256 的 key 用（不要 hex 解码）。
	plain, err := aesECBDecrypt(decoded, []byte(keyStr))
	if err != nil {
		return "", false
	}
	return string(plain), true
}

func hexKey(hexStr string) ([]byte, error) {
	if len(hexStr)%2 != 0 {
		return nil, errors.New("bad hex key")
	}
	out := make([]byte, len(hexStr)/2)
	for i := 0; i < len(out); i++ {
		hi := hexVal(hexStr[2*i])
		lo := hexVal(hexStr[2*i+1])
		if hi < 0 || lo < 0 {
			return nil, errors.New("bad hex key")
		}
		out[i] = byte(hi<<4 | lo)
	}
	return out, nil
}

func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// ---- 密码哈希（兼容 PHP password_hash / password_verify，$2y$）----
func passwordHash(pw string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(h)
}

func passwordVerify(pw, hash string) bool {
	if hash == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw))
	return err == nil
}

// verifyAuthKey：若 .auth_hash 不存在则用输入创建（与 PHP 行为一致）
func verifyAuthKey(input string) bool {
	input = strings.TrimSpace(input)
	if input == "" {
		return false
	}
	f := dp(authHashFile)
	raw, err := os.ReadFile(f)
	if err != nil || strings.TrimSpace(string(raw)) == "" {
		// 不存在则创建
		if h := passwordHash(input); h != "" {
			_ = atomicWrite(f, h)
			return true
		}
		return false
	}
	return passwordVerify(input, strings.TrimSpace(string(raw)))
}

func changeAuthPassword(current, newPw, confirm string) map[string]any {
	current = strings.TrimSpace(current)
	newPw = strings.TrimSpace(newPw)
	confirm = strings.TrimSpace(confirm)
	if current == "" || newPw == "" || confirm == "" {
		return map[string]any{"ok": false, "msg": "请填写当前密码和新密码"}
	}
	if !verifyAuthKey(current) {
		return map[string]any{"ok": false, "msg": "当前密码错误"}
	}
	if newPw != confirm {
		return map[string]any{"ok": false, "msg": "两次输入的新密码不一致"}
	}
	if len(newPw) < 6 {
		return map[string]any{"ok": false, "msg": "新密码至少需要6位"}
	}
	if subtle.ConstantTimeCompare([]byte(current), []byte(newPw)) == 1 {
		return map[string]any{"ok": false, "msg": "新密码不能和当前密码相同"}
	}
	h := passwordHash(newPw)
	if h == "" {
		return map[string]any{"ok": false, "msg": "密码哈希生成失败"}
	}
	if !atomicWrite(dp(authHashFile), h) {
		return map[string]any{"ok": false, "msg": "密码保存失败，请检查文件权限"}
	}
	return map[string]any{"ok": true, "msg": "密码已修改"}
}

// ---- JWT exp ----
func jwtExp(token string) int {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return 0
	}
	payload := strings.NewReplacer("-", "+", "_", "/").Replace(parts[1])
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return 0
	}
	var data map[string]any
	if json.Unmarshal(decoded, &data) != nil {
		return 0
	}
	return qInt(data["exp"])
}

// jwtExpSafe 返回时间；无 exp 则 0
func jwtExpTime(token string) time.Time {
	e := jwtExp(token)
	if e <= 0 {
		return time.Time{}
	}
	return time.Unix(int64(e), 0)
}
