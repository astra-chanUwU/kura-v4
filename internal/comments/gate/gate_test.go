package gate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvaluate(t *testing.T) {
	cases := []struct {
		name string
		body string
		want Decision
	}{
		{name: "empty", body: "", want: allow()},
		{name: "whitespace", body: " \n\t  ", want: allow()},
		{name: "only zero-width", body: "\u200b\u200c\u200d\ufeff\u2060", want: allow()},
		{name: "praise", body: "beautiful work, the lighting really sings", want: allow()},
		{name: "ten out of ten", body: "10/10", want: allow()},
		{name: "resolution", body: "1920x1080", want: allow()},
		{name: "resolution p", body: "1080p", want: allow()},
		{name: "year", body: "from 2019", want: allow()},
		{name: "iso date", body: "2026-11-19", want: allow()},
		{name: "compact date", body: "20261119", want: allow()},
		{name: "chapter", body: "chapter 12", want: allow()},
		{name: "rgb", body: "rgb 255 128 64", want: allow()},
		{name: "rgb uppercase", body: "RGB 255 128 64", want: allow()},
		{name: "bare mention", body: "@artist nice colors", want: allow()},
		{name: "telegram denial", body: "I don't use telegram", want: allow()},
		{name: "telegram alone", body: "telegram", want: allow()},
		{name: "whatsapp noisy", body: "whatsapp is noisy", want: allow()},
		{name: "whatsapp company", body: "whatsapp.company is a long word", want: allow()},
		{name: "strong signal", body: "a strong signal", want: allow()},
		{name: "signal boost", body: "signal boost", want: allow()},
		{name: "signal alone", body: "signal", want: allow()},
		{name: "signal is not mine", body: "my signal isn't for sale", want: allow()},
		{name: "uuid", body: "550e8400-e29b-41d4-a716-446655440000", want: allow()},
		{name: "uuid numeric tail", body: "deadbeef-dead-beef-abcd-123456789012", want: allow()},
		{name: "uuid all digits", body: "12345678-1234-1234-1234-123456789012", want: allow()},
		{name: "bare issue tag", body: "issue#1234", want: allow()},
		{name: "issue tag in a sentence", body: "please look at issue#1234", want: allow()},
		{name: "discord alone", body: "discord is fun", want: allow()},
		{name: "discord.ggg", body: "discord.ggg is not a link", want: allow()},
		{name: "seven hyphen digits", body: "123-4567", want: allow()},
		{name: "eight contiguous", body: "12345678", want: allow()},
		{name: "nine contiguous", body: "123456789", want: allow()},
		{name: "sixteen digits", body: "1234567890123456", want: allow()},
		{name: "seven number words", body: "one two three four five six seven", want: allow()},
		{name: "eighty is not eight", body: "eighty one two three four five six seven", want: allow()},
		{name: "number words interrupted", body: "oh no one two three four five six seven", want: allow()},
		{name: "call me without a number", body: "call me", want: allow()},
		{name: "number without digits", body: "a number of people liked this", want: allow()},
		{name: "not.me", body: "see not.me today", want: allow()},
		{name: "at.me", body: "visit at.me today", want: allow()},
		{name: "proton.me", body: "proton.me is an email host", want: allow()},
		{name: "notwhatsapp.com", body: "notwhatsapp.com", want: allow()},
		{name: "snap alone", body: "oh snap", want: allow()},
		{name: "kik alone", body: "kik is a chat app", want: allow()},
		{name: "kik.media", body: "kik.media is not a contact link", want: allow()},
		{name: "nightgown mention", body: "the nightgown study @artist", want: allow()},
		{name: "phone context just outside", body: "phone " + strings.Repeat("x", 40) + "12345678", want: allow()},
		{name: "headphones are not phone context", body: "headphones 12345678", want: allow()},
		{name: "hotel is not a tel link", body: "hotel:12345678", want: allow()},
		{name: "date beside a number word", body: "one 2026-11-19", want: allow()},

		{name: "ten digits", body: "5551234567", want: block("phone")},
		{name: "fifteen digits", body: "123456789012345", want: block("phone")},
		{name: "hyphen phone", body: "555-123-4567", want: block("phone")},
		{name: "dot phone", body: "555.123.4567", want: block("phone")},
		{name: "spaced international", body: "+1 202 555 0123", want: block("phone")},
		{name: "paren phone", body: "(555) 123-4567", want: block("phone")},
		{name: "slash phone", body: "555/123/4567", want: block("phone")},
		{name: "en dash phone", body: "555\u2013123\u20134567", want: block("phone")},
		{name: "em dash phone", body: "555\u2014123\u20144567", want: block("phone")},
		{name: "tel link", body: "tel:+1-555-123-4567", want: block("phone")},
		{name: "tel eight digits", body: "tel:12345678", want: block("phone")},
		{name: "leading plus eight", body: "+12345678", want: block("phone")},
		{name: "eight with separator", body: "1234-5678", want: block("phone")},
		{name: "eight with spaces", body: "1234 5678", want: block("phone")},
		{name: "nine with separators", body: "123-456-789", want: block("phone")},
		{name: "phone context before", body: "my phone is 12345678", want: block("phone")},
		{name: "phone context after", body: "12345678 is my phone", want: block("phone")},
		{name: "phone context just inside", body: "phone " + strings.Repeat("x", 39) + "12345678", want: block("phone")},
		{name: "mobile context", body: "mobile 123456789", want: block("phone")},
		{name: "telephone context", body: "my telephone is 12345678", want: block("phone")},
		{name: "cellphone context", body: "cellphone 12345678", want: block("phone")},
		{name: "plus with a space", body: "+ 12345678", want: block("phone")},
		{name: "call me with number", body: "call me 12345678", want: block("phone")},
		{name: "eight number words", body: "one two three four five six seven eight", want: block("phone")},
		{name: "oh number words", body: "oh one two three four five six seven", want: block("phone")},
		{name: "zero number words", body: "zero one two three four five six seven", want: block("phone")},
		{name: "mixed number words", body: "one 2 three 4 five 6 seven 8", want: block("phone")},
		{name: "comma number words", body: "one, two, three, four, five, six, seven, eight", want: block("phone")},
		{name: "adjacent mixed number words", body: "one2three4five6seven8", want: block("phone")},
		{name: "fullwidth digits", body: "５５５１２３４５６７", want: block("phone")},
		{name: "fullwidth plus", body: "＋１２３４５６７８", want: block("phone")},
		{name: "superscript digits", body: "\u00b9\u00b2\u00b3\u2074\u2075\u2076\u2077\u2078\u2079\u2070", want: block("phone")},
		{name: "zero-width inside phone", body: "555\u200b1234567", want: block("phone")},
		{name: "spaced eight is not rgb", body: "255 128 64", want: block("phone")},

		{name: "https t.me", body: "https://t.me/example_user", want: block("telegram")},
		{name: "t.me", body: "t.me/example", want: block("telegram")},
		{name: "telegram.me", body: "telegram.me/example", want: block("telegram")},
		{name: "telegram.dog", body: "telegram.dog/example", want: block("telegram")},
		{name: "telegram.im", body: "telegram.im/example", want: block("telegram")},
		{name: "tg scheme", body: "tg://resolve?domain=example", want: block("telegram")},
		{name: "t dot me", body: "t dot me / example_user", want: block("telegram")},
		{name: "t bracket dot me", body: "t[.]me/example_user", want: block("telegram")},
		{name: "t dot word me", body: "t[dot]me/example_user", want: block("telegram")},
		{name: "t paren dot me", body: "t (dot) me/example_user", want: block("telegram")},
		{name: "zero-width inside t.me", body: "t\u200b.me/example_user", want: block("telegram")},
		{name: "every zero-width inside t.me", body: "t\u200b\u200c\u200d\ufeff\u2060.me/example_user", want: block("telegram")},
		{name: "fullwidth t.me", body: "ｔ．ｍｅ／ｅｘａｍｐｌｅ", want: block("telegram")},
		{name: "mixed case t.me", body: "HTTPS://T.ME/Example_User", want: block("telegram")},
		{name: "my telegram is handle", body: "my telegram is @example_user", want: block("telegram")},
		{name: "my telegram is bare handle", body: "my telegram is example_user", want: block("telegram")},
		{name: "tg id", body: "tg id 123456789", want: block("telegram")},
		{name: "tg handle", body: "tg @example_user", want: block("telegram")},
		{name: "add me on telegram", body: "add me on telegram", want: block("telegram")},
		{name: "spaced add me on telegram", body: "add  me   on   telegram", want: block("telegram")},
		{name: "spaced telegram letters", body: "t e l e g r a m @example_user", want: block("telegram")},
		{name: "fullwidth telegram word", body: "my ｔｅｌｅｇｒａｍ is @example_user", want: block("telegram")},
		{name: "spaced add me on telegram letters", body: "add me on t e l e g r a m", want: block("telegram")},
		{name: "tg with phone", body: "tg 1234567890", want: block("phone", "telegram")},
		{name: "find me on telegram", body: "find me on telegram", want: block("telegram")},
		{name: "cyrillic e in t.me", body: "t.m\u0435/example_user", want: block("telegram")},

		{name: "wa.me", body: "https://wa.me/15551234567", want: block("phone", "whatsapp")},
		{name: "wa bracket me", body: "wa[.]me/15551234567", want: block("phone", "whatsapp")},
		{name: "chat.whatsapp.com", body: "https://chat.whatsapp.com/AbCdEfGh", want: block("whatsapp")},
		{name: "api.whatsapp.com", body: "https://api.whatsapp.com/send?phone=15551234567", want: block("phone", "whatsapp")},
		{name: "whatsapp.com", body: "https://whatsapp.com/download", want: block("whatsapp")},
		{name: "add me on whatsapp", body: "add me on whatsapp", want: block("whatsapp")},
		{name: "whatsapp with short number", body: "whatsapp 1234567", want: block("whatsapp")},
		{name: "whatsapp with eight digits", body: "whatsapp 12345678", want: block("phone", "whatsapp")},

		{name: "discord.gg", body: "https://discord.gg/artRoom", want: block("discord")},
		{name: "discord.com/invite", body: "https://discord.com/invite/artRoom", want: block("discord")},
		{name: "discordapp.com/invite", body: "discordapp.com/invite/artRoom", want: block("discord")},
		{name: "discord dot gg", body: "discord dot gg/artRoom", want: block("discord")},
		{name: "discord tag", body: "my discord is coolname#1234", want: block("discord")},
		{name: "discord tag elsewhere", body: "discord is okay but ping cool_user#0042", want: block("discord")},
		{name: "my discord is bare handle", body: "my discord is coolname", want: block("discord")},

		{name: "signal.me", body: "https://signal.me/#p/abc", want: block("signal")},
		{name: "signal.me short", body: "signal.me/abc", want: block("signal")},
		{name: "add me on signal", body: "add me on signal", want: block("signal")},
		{name: "my signal is", body: "my signal is luna", want: block("signal")},

		{name: "snapchat.com/add", body: "https://snapchat.com/add/example", want: block("snapchat")},
		{name: "add me on snapchat", body: "add me on snapchat", want: block("snapchat")},
		{name: "my snap is", body: "my snap is example", want: block("snapchat")},
		{name: "snap colon", body: "snap: example", want: block("snapchat")},

		{name: "kik.me", body: "https://kik.me/example", want: block("kik")},
		{name: "add me on kik", body: "add me on kik", want: block("kik")},
		{name: "my kik is", body: "my kik is example", want: block("kik")},

		{name: "email", body: "painter@example.com", want: flag("email")},
		{name: "email in a sentence", body: "reach me at painter@example.com.", want: flag("email")},
		{name: "email with a long local part", body: "abc1234567890@gmail.com", want: flag("email")},
		{name: "email beside telegram denial", body: "I don't use telegram, mail me at painter@example.com", want: flag("email")},
		{name: "dm me", body: "dm me", want: flag("solicitation")},
		{name: "dms are open", body: "dms are open", want: flag("solicitation")},
		{name: "dms open", body: "dms open", want: flag("solicitation")},
		{name: "text me", body: "text me", want: flag("solicitation")},
		{name: "hmu", body: "hmu", want: flag("solicitation")},
		{name: "hit me up", body: "hit me up", want: flag("solicitation")},
		{name: "pm me", body: "pm me", want: flag("solicitation")},
		{name: "message me", body: "message me", want: flag("solicitation")},
		{name: "inbox me", body: "inbox me", want: flag("solicitation")},
		{name: "add me", body: "add me", want: flag("solicitation")},
		{name: "add me on discord", body: "add me on discord", want: block("discord")},

		{name: "phone and email", body: "painter@example.com 5551234567", want: block("phone", "email")},
		{name: "email and dm me", body: "dm me at painter@example.com", want: flag("email", "solicitation")},
		{name: "phone email and text me", body: "text me painter@example.com 5551234567", want: block("phone", "email", "solicitation")},
		{name: "telegram add me keeps email", body: "add me on telegram, mail painter@example.com", want: block("telegram", "email")},
		{name: "telegram add me plus another add me", body: "add me on telegram and also add me", want: block("telegram", "solicitation")},
		{name: "text me with short number", body: "text me 12345678", want: block("phone", "solicitation")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, Evaluate(tc.body))
		})
	}
}

func allow() Decision { return Decision{Action: ActionAllow} }

func block(categories ...string) Decision {
	return Decision{Action: ActionBlock, Categories: categories}
}

func flag(categories ...string) Decision {
	return Decision{Action: ActionFlag, Categories: categories}
}
