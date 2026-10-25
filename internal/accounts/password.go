package accounts

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Password storage follows the OWASP Password Storage Cheat Sheet:
// Argon2id in PHC string format, with parameters at or above the OWASP
// minimum (19 MiB memory, 2 iterations, parallelism 1). Kura's target
// parameters exceed that minimum; logins transparently re-hash passwords
// stored with weaker parameters (see NeedsRehash).
const (
	argonTime    = 2
	argonMemory  = 64 * 1024 // KiB
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16
)

// Bounds applied to stored hashes before verifying, so a corrupt or
// hostile stored value cannot make verification allocate without limit.
const (
	maxArgonMemory  = 256 * 1024 // KiB
	maxArgonTime    = 10
	maxArgonThreads = 16
)

type argonParams struct {
	memory  uint32
	time    uint32
	threads uint8
}

var targetParams = argonParams{memory: argonMemory, time: argonTime, threads: argonThreads}

func hashWithParams(password string, p argonParams) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, p.time, p.memory, p.threads, argonKeyLen)
	enc := base64.RawStdEncoding.EncodeToString
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", p.memory, p.time, p.threads, enc(salt), enc(hash)), nil
}

func HashPassword(password string) (string, error) {
	return hashWithParams(password, targetParams)
}

func parseHash(encoded string) (argonParams, []byte, []byte, bool) {
	parts := strings.Split(encoded, "$")
	// ["", "argon2id", "v=19", "m=..,t=..,p=..", salt, hash]
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return argonParams{}, nil, nil, false
	}
	var p argonParams
	for _, kv := range strings.Split(parts[3], ",") {
		pair := strings.SplitN(kv, "=", 2)
		if len(pair) != 2 {
			return argonParams{}, nil, nil, false
		}
		n, err := strconv.ParseUint(pair[1], 10, 32)
		if err != nil {
			return argonParams{}, nil, nil, false
		}
		switch pair[0] {
		case "m":
			p.memory = uint32(n)
		case "t":
			p.time = uint32(n)
		case "p":
			if n > 255 {
				return argonParams{}, nil, nil, false
			}
			p.threads = uint8(n)
		}
	}
	if p.memory == 0 || p.memory > maxArgonMemory || p.time == 0 || p.time > maxArgonTime || p.threads == 0 || p.threads > maxArgonThreads {
		return argonParams{}, nil, nil, false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 8 || len(salt) > 64 {
		return argonParams{}, nil, nil, false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(want) < 16 || len(want) > 64 {
		return argonParams{}, nil, nil, false
	}
	return p, salt, want, true
}

func CheckPassword(password, encoded string) bool {
	p, salt, want, ok := parseHash(encoded)
	if !ok {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, p.time, p.memory, p.threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

// NeedsRehash reports whether a stored hash should be replaced with one
// using the current target parameters (called after a successful login).
func NeedsRehash(encoded string) bool {
	p, _, want, ok := parseHash(encoded)
	if !ok {
		return true
	}
	return p != targetParams || len(want) != argonKeyLen
}

// dummyHash lets login spend the same Argon2id effort for unknown users,
// so response timing does not reveal whether a username exists.
var dummyHash, _ = HashPassword("kura-timing-equalizer")
