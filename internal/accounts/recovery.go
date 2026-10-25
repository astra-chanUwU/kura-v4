package accounts

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"strings"
)

// Recovery keys are the last-resort way back into an account when both
// the password and every passkey are lost. A key carries 160 bits of
// entropy (32 base32 characters); only its SHA-256 hash is stored — the
// same treatment as session tokens — and the plaintext is shown once.
const recoveryKeyAlphabet = "abcdefghijklmnopqrstuvwxyz234567"

func GenerateRecoveryKey() (string, []byte, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	chars := make([]byte, 32)
	for i, b := range raw {
		chars[i] = recoveryKeyAlphabet[int(b)%len(recoveryKeyAlphabet)]
	}
	groups := make([]string, 0, 8)
	for i := 0; i < len(chars); i += 4 {
		groups = append(groups, strings.ToUpper(string(chars[i:i+4])))
	}
	key := strings.Join(groups, "-")
	return key, RecoveryKeyHash(key), nil
}

// RecoveryKeyHash hashes the normalized key: case, dashes and spaces the
// user may type do not matter.
func RecoveryKeyHash(key string) []byte {
	h := sha256.Sum256([]byte(normalizeRecoveryKey(key)))
	return h[:]
}

func normalizeRecoveryKey(key string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(key) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func CheckRecoveryKey(key string, storedHash []byte) bool {
	got := RecoveryKeyHash(key)
	return subtle.ConstantTimeCompare(got, storedHash) == 1
}
