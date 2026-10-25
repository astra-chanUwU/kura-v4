package accounts

import "strings"

// Password policy follows NIST SP 800-63B: length over complexity rules.
// A password here can be the account's only sign-in method, so the
// single-factor minimum of 15 characters applies; up to 128 are allowed
// (NIST requires accepting at least 64). There are no composition rules —
// instead, new passwords are screened against common/breached choices.
const (
	minPasswordLen = 15
	maxPasswordLen = 128
)

func validatePassword(username, password string) error {
	if len(password) < minPasswordLen || len(password) > maxPasswordLen {
		return invalid("Passwords need 15–128 characters — a short sentence you can remember makes a great one.")
	}
	lower := strings.ToLower(password)
	if username != "" && strings.Contains(lower, strings.ToLower(username)) {
		return invalid("That password contains your username — pick something less guessable.")
	}
	if commonPasswords[lower] {
		return invalid("That password is on lists of commonly used passwords — pick a less guessable one.")
	}
	if allSameRune(password) {
		return invalid("That password is a single repeated character — pick a less guessable one.")
	}
	return nil
}

func allSameRune(s string) bool {
	var first rune
	for i, r := range s {
		if i == 0 {
			first = r
		} else if r != first {
			return false
		}
	}
	return len(s) > 0
}

// A small embedded screen of very common passwords (compared lowercase).
// This is a floor, not a full breached-password corpus.
var commonPasswords = map[string]bool{
	"123456": true, "123456789": true, "12345678": true, "12345": true, "1234567": true,
	"1234567890": true, "123456789012345": true, "0123456789": true, "111111": true,
	"password": true, "password1": true, "password123": true, "password1234": true,
	"password12345": true, "password123456": true, "passw0rd": true, "p@ssw0rd": true,
	"qwerty": true, "qwerty123": true, "qwertyuiop": true, "qwertyqwerty": true,
	"abc123": true, "abcd1234": true, "letmein": true, "letmein123": true,
	"admin": true, "administrator": true, "welcome": true, "welcome123": true,
	"login": true, "user": true, "guest": true, "default": true, "changeme": true,
	"iloveyou": true, "iloveyou1": true, "iloveyou123": true, "iloveyouiloveyou": true,
	"monkey": true, "dragon": true, "football": true, "baseball": true,
	"sunshine": true, "princess": true, "superman": true, "batman": true,
	"trustno1": true, "master": true, "shadow": true, "michael": true,
	"charlie": true, "thomas": true, "jordan": true, "hunter": true,
	"computer": true, "internet": true, "secret": true, "summer": true,
	"winter": true, "spring": true, "autumn": true, "hello123": true,
	"freedom": true, "whatever": true, "nicole": true, "jessica": true,
	"andrew": true, "joshua": true, "daniel": true, "matthew": true,
	"pokemon": true, "naruto": true, "lol12345": true, "123qwe": true,
	"qwe123": true, "zxcvbn": true, "zxcvbnm": true, "asdfgh": true,
	"asdfghjkl": true, "1qaz2wsx": true, "1q2w3e4r": true, "1q2w3e": true,
	"q1w2e3r4": true, "aa123456": true, "a123456": true, "abc1234": true,
	"abcd12345": true, "stars": true, "flower": true, "happy123": true,
	"london": true, "paris123": true, "soccer": true, "hockey": true,
	"killer": true, "george": true, "pepper": true, "ginger": true,
	"love1234": true, "lovely": true, "sweetie": true, "diamond": true,
	"golden": true, "silver": true, "bronze": true, "purple": true,
	"orange123": true, "banana": true, "strawberry": true, "chocolate": true,
	"correcthorsebatterystaple": true,
}
