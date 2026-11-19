// Package gate classifies comment text that shares off-site contact details.
// Block categories must not be published. Flag categories may be published
// and are queued for a moderator.
package gate

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

type Action int

const (
	ActionAllow Action = iota
	ActionFlag
	ActionBlock
)

type Decision struct {
	Action Action
	// Categories that matched, unique, in this exact order:
	// phone, telegram, whatsapp, discord, signal, snapchat, kik, email, solicitation
	Categories []string
}

const (
	catPhone        = "phone"
	catTelegram     = "telegram"
	catWhatsapp     = "whatsapp"
	catDiscord      = "discord"
	catSignal       = "signal"
	catSnapchat     = "snapchat"
	catKik          = "kik"
	catEmail        = "email"
	catSolicitation = "solicitation"

	// nearWindow is the distance, in runes, for phone context and for a
	// platform word paired with a handle, id, or number.
	nearWindow = 40
)

var categoryOrder = []string{
	catPhone, catTelegram, catWhatsapp, catDiscord, catSignal, catSnapchat, catKik, catEmail, catSolicitation,
}

var blockCategory = map[string]bool{
	catPhone: true, catTelegram: true, catWhatsapp: true, catDiscord: true,
	catSignal: true, catSnapchat: true, catKik: true,
}

// Dot-obfuscated separators between a host label and its TLD: ".", "dot",
// "[.]", "[dot]", "(.)", "(dot)", and "t.dot.me" style dots around the word.
const dotJoin = `(?:\[(?:\.|dot)\]|\((?:\.|dot)\)|\.?dot\.?|\.)`

var (
	tMeRe           = hostRE("t", "me")
	telegramHostRe  = hostRE("telegram", "me|dog|im")
	tgSchemeRe      = regexp.MustCompile(`(?:^|[^a-z0-9])tg://`)
	waMeRe          = hostRE("wa", "me")
	whatsappComRe   = hostRE("whatsapp", "com")
	signalMeRe      = hostRE("signal", "me")
	kikMeRe         = hostRE("kik", "me")
	discordGGRe     = hostRE("discord", "gg")
	discordInviteRe = regexp.MustCompile(`(?:^|[^a-z0-9])discord(?:app)?\s*` + dotJoin + `\s*com/invite(?:[^a-z0-9]|$)`)
	snapAddRe       = regexp.MustCompile(`(?:^|[^a-z0-9])snapchat\s*` + dotJoin + `\s*com/add(?:[^a-z0-9]|$)`)

	emailRe      = regexp.MustCompile(`\b[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}\b`)
	uuidRe       = regexp.MustCompile(`\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
	resolutionRe = regexp.MustCompile(`\b\d{3,5}[x×]\d{3,5}\b|\b\d{3,4}p\b`)
	rgbRe        = regexp.MustCompile(`rgb\s*\(?\s*(\d{1,3})[\s,]+(\d{1,3})[\s,]+(\d{1,3})`)
	discordTagRe = regexp.MustCompile(`(?:^|[^a-z0-9._])[a-z0-9._]{2,32}#\d{4}(?:[^0-9]|$)`)
	atHandleRe   = regexp.MustCompile(`@[a-z][a-z0-9_]{3,31}`)
	handleTailRe = regexp.MustCompile(`^@?[a-z0-9_.]{4,32}`)
	idRe         = regexp.MustCompile(`\b\d{5,15}\b`)
	addMeRe      = regexp.MustCompile(`\badd\s+me\b`)

	solicitationRes = []*regexp.Regexp{
		regexp.MustCompile(`\bdm\s+me\b`),
		regexp.MustCompile(`\bdms\s+are\s+open\b`),
		regexp.MustCompile(`\bdms\s+open\b`),
		regexp.MustCompile(`\btext\s+me\b`),
		regexp.MustCompile(`\bhmu\b`),
		regexp.MustCompile(`\bhit\s+me\s+up\b`),
		regexp.MustCompile(`\bpm\s+me\b`),
		regexp.MustCompile(`\bmessage\s+me\b`),
		regexp.MustCompile(`\binbox\s+me\b`),
	}
	phoneContextRes = []*regexp.Regexp{
		regexp.MustCompile(`\b(?:tele|cell)?phone\b`),
		regexp.MustCompile(`\bnumber\b`),
		regexp.MustCompile(`\bcell\b`),
		regexp.MustCompile(`\bmobile\b`),
		regexp.MustCompile(`\bcall\s+me\b`),
		regexp.MustCompile(`\btext\s+me\b`),
		regexp.MustCompile(`\bsms\b`),
		regexp.MustCompile(`\bwhatsapp\b`),
	}
)

// Platforms whose "add me on <name>" phrase is already a block rule.
var addMeBlockPlatforms = []string{"telegram", "whatsapp", "discord", "signal", "snapchat", "kik"}

var numberWords = []string{
	"three", "seven", "eight", "four", "five", "nine", "zero", "one", "two", "six", "oh",
}

func hostRE(label, tld string) *regexp.Regexp {
	return regexp.MustCompile(`(?:^|[^a-z0-9])` + label + `\s*` + dotJoin + `\s*(?:` + tld + `)(?:[^a-z0-9]|$)`)
}

// Evaluate classifies body. It normalizes a copy; callers do not pre-process the text.
func Evaluate(body string) Decision {
	s := normalize(body)
	if strings.TrimSpace(s) == "" {
		return Decision{Action: ActionAllow}
	}
	hits := map[string]bool{}
	if hitPhone(s) {
		hits[catPhone] = true
	}
	if hitTelegram(s) {
		hits[catTelegram] = true
	}
	if hitWhatsapp(s) {
		hits[catWhatsapp] = true
	}
	if hitDiscord(s) {
		hits[catDiscord] = true
	}
	if hitSignal(s) {
		hits[catSignal] = true
	}
	if hitSnap(s) {
		hits[catSnapchat] = true
	}
	if hitKik(s) {
		hits[catKik] = true
	}
	if emailRe.MatchString(s) {
		hits[catEmail] = true
	}
	if hitSolicitation(s) {
		hits[catSolicitation] = true
	}
	return decide(hits)
}

func normalize(s string) string {
	s = norm.NFKC.String(s)
	s = strings.ToLower(s)
	return strings.Map(func(r rune) rune {
		switch r {
		case '\u200b', '\u200c', '\u200d', '\ufeff', '\u2060':
			return -1
		default:
			return foldConfusable(r)
		}
	}, s)
}

// foldConfusable maps letters that still look Latin after NFKC, so a
// Cyrillic "е" inside t.me does not slip past the host check.
func foldConfusable(r rune) rune {
	switch r {
	case 'а', 'ɑ':
		return 'a'
	case 'е', 'ё':
		return 'e'
	case 'о', 'ο', 'ө':
		return 'o'
	case 'р':
		return 'p'
	case 'с', 'ϲ':
		return 'c'
	case 'у':
		return 'y'
	case 'х', 'χ':
		return 'x'
	case 'і', 'ι':
		return 'i'
	case 'ѕ':
		return 's'
	case 'һ':
		return 'h'
	case 'ј':
		return 'j'
	case 'ɡ':
		return 'g'
	case 'ⅼ', 'ӏ':
		return 'l'
	case 'т':
		return 't'
	case 'м':
		return 'm'
	case 'к':
		return 'k'
	case 'ԁ':
		return 'd'
	default:
		return r
	}
}

func decide(hits map[string]bool) Decision {
	var cats []string
	block := false
	for _, c := range categoryOrder {
		if !hits[c] {
			continue
		}
		cats = append(cats, c)
		if blockCategory[c] {
			block = true
		}
	}
	if len(cats) == 0 {
		return Decision{Action: ActionAllow}
	}
	if block {
		return Decision{Action: ActionBlock, Categories: cats}
	}
	return Decision{Action: ActionFlag, Categories: cats}
}

func hitSolicitation(s string) bool {
	for _, re := range solicitationRes {
		if re.MatchString(s) {
			return true
		}
	}
	for _, loc := range addMeRe.FindAllStringIndex(s, -1) {
		if !platformAfterAddMe(s, loc[1], addMeBlockPlatforms...) {
			return true
		}
	}
	return false
}

func hasAddMeOn(s string, words ...string) bool {
	for _, loc := range addMeRe.FindAllStringIndex(s, -1) {
		if platformAfterAddMe(s, loc[1], words...) {
			return true
		}
	}
	return false
}

func platformAfterAddMe(s string, end int, words ...string) bool {
	rest := trimLeft(s[end:])
	if !hasWordPrefix(rest, "on") {
		return false
	}
	rest = trimLeft(rest[len("on"):])
	for _, w := range words {
		if _, ok := matchKeywordAt(rest, w); ok {
			return true
		}
	}
	return false
}

func hitTelegram(s string) bool {
	if tgSchemeRe.MatchString(s) || tMeRe.MatchString(s) || telegramHostRe.MatchString(s) {
		return true
	}
	if hasAddMeOn(s, "telegram") {
		return true
	}
	spans := findKeywords(s, "telegram")
	spans = append(spans, findKeywords(s, "tg")...)
	return myPlatformHandle(s, spans) || pairedHandle(s, spans) || pairedID(s, spans) || meOn(s, spans)
}

func hitWhatsapp(s string) bool {
	spans := findKeywords(s, "whatsapp")
	if waMeRe.MatchString(s) || whatsappComRe.MatchString(s) || hasAddMeOn(s, "whatsapp") || meOn(s, spans) {
		return true
	}
	return pairedDigits(s, spans, 7)
}

func hitDiscord(s string) bool {
	spans := findKeywords(s, "discord")
	if discordGGRe.MatchString(s) || discordInviteRe.MatchString(s) || hasAddMeOn(s, "discord") || meOn(s, spans) {
		return true
	}
	if len(spans) == 0 {
		return false
	}
	return discordTagRe.MatchString(s) || myPlatformHandle(s, spans)
}

func hitSignal(s string) bool {
	spans := findKeywords(s, "signal")
	if signalMeRe.MatchString(s) || hasAddMeOn(s, "signal") || meOn(s, spans) {
		return true
	}
	return myWordIs(s, spans)
}

func hitSnap(s string) bool {
	if snapAddRe.MatchString(s) || hasAddMeOn(s, "snapchat") {
		return true
	}
	spans := findKeywords(s, "snap")
	spans = append(spans, findKeywords(s, "snapchat")...)
	return myPlatformHandle(s, spans) || colonHandle(s, spans) || meOn(s, spans)
}

func hitKik(s string) bool {
	spans := findKeywords(s, "kik")
	if kikMeRe.MatchString(s) || hasAddMeOn(s, "kik") || meOn(s, spans) {
		return true
	}
	return myPlatformHandle(s, spans)
}

// meOn reports a handoff such as "find me on telegram", including a
// platform name whose letters are spaced out.
func meOn(s string, spans [][2]int) bool {
	for _, sp := range spans {
		prefix := strings.TrimRightFunc(s[:sp[0]], unicode.IsSpace)
		if !strings.HasSuffix(prefix, "on") || !boundaryBefore(prefix, len(prefix)-len("on")) {
			continue
		}
		prefix = strings.TrimRightFunc(prefix[:len(prefix)-len("on")], unicode.IsSpace)
		if strings.HasSuffix(prefix, "me") && boundaryBefore(prefix, len(prefix)-len("me")) {
			return true
		}
	}
	return false
}

func myWordIs(s string, spans [][2]int) bool {
	for _, sp := range spans {
		if lastWordIs(s[:sp[0]], "my") && hasWordPrefix(trimLeft(s[sp[1]:]), "is") {
			return true
		}
	}
	return false
}

func myPlatformHandle(s string, spans [][2]int) bool {
	for _, sp := range spans {
		if !lastWordIs(s[:sp[0]], "my") {
			continue
		}
		rest := trimLeft(s[sp[1]:])
		if !hasWordPrefix(rest, "is") {
			continue
		}
		rest = trimLeft(rest[len("is"):])
		if handleTailRe.MatchString(rest) {
			return true
		}
	}
	return false
}

func colonHandle(s string, spans [][2]int) bool {
	for _, sp := range spans {
		rest := trimLeft(s[sp[1]:])
		if !strings.HasPrefix(rest, ":") {
			continue
		}
		rest = trimLeft(rest[1:])
		if handleTailRe.MatchString(rest) {
			return true
		}
	}
	return false
}

func pairedHandle(s string, spans [][2]int) bool {
	if len(spans) == 0 {
		return false
	}
	emails := findSpans(emailRe, s)
	for _, loc := range atHandleRe.FindAllStringIndex(s, -1) {
		if overlaps(loc[0], loc[1], emails) {
			continue
		}
		for _, sp := range spans {
			if nearSpan(s, sp[0], sp[1], loc[0], loc[1], nearWindow) {
				return true
			}
		}
	}
	return false
}

func pairedID(s string, spans [][2]int) bool {
	if len(spans) == 0 {
		return false
	}
	skip := findSpans(emailRe, s)
	skip = append(skip, findSpans(uuidRe, s)...)
	skip = append(skip, findSpans(resolutionRe, s)...)
	for _, loc := range idRe.FindAllStringIndex(s, -1) {
		tok := s[loc[0]:loc[1]]
		if len(tok) == 8 && isYMD(tok) {
			continue
		}
		if overlaps(loc[0], loc[1], skip) {
			continue
		}
		for _, sp := range spans {
			if nearSpan(s, sp[0], sp[1], loc[0], loc[1], nearWindow) {
				return true
			}
		}
	}
	return false
}

func pairedDigits(s string, spans [][2]int, minDigits int) bool {
	if len(spans) == 0 {
		return false
	}
	protected := protectedSpans(s)
	for _, run := range findDigitRuns(s) {
		n := len(run.digits)
		if n < minDigits || n > 15 || isDateRun(run) || overlaps(run.start, run.end, protected) {
			continue
		}
		for _, sp := range spans {
			if nearSpan(s, sp[0], sp[1], run.start, run.end, nearWindow) {
				return true
			}
		}
	}
	return false
}

func hitPhone(s string) bool {
	protected := protectedSpans(s)
	for _, run := range findDigitRuns(s) {
		if phoneRun(s, run, protected) {
			return true
		}
	}
	return hasSpokenPhone(s)
}

// phoneRun reports whether a digit run is a high-confidence phone number.
// 10 to 15 digits count. 8 or 9 count when a separator, a leading plus, a
// tel: prefix, or nearby phone context is present. Calendar dates, UUIDs,
// resolutions, RGB triples, and digits inside an email are ignored.
func phoneRun(s string, run digitRun, protected [][2]int) bool {
	n := len(run.digits)
	if n < 8 || n > 15 || overlaps(run.start, run.end, protected) || isDateRun(run) {
		return false
	}
	if n >= 10 {
		return true
	}
	return run.leadPlus || run.internal || hasTel(s, run.start) || hasPhoneContext(s, run.start, run.end)
}

func hasTel(s string, start int) bool {
	j := start
	for j > 0 {
		r, sz := utf8.DecodeLastRuneInString(s[:j])
		if unicode.IsSpace(r) {
			j -= sz
			continue
		}
		break
	}
	const prefix = "tel:"
	if !strings.HasSuffix(s[:j], prefix) {
		return false
	}
	return boundaryBefore(s, j-len(prefix))
}

func hasPhoneContext(s string, start, end int) bool {
	for _, re := range phoneContextRes {
		for _, loc := range re.FindAllStringIndex(s, -1) {
			if nearSpan(s, loc[0], loc[1], start, end, nearWindow) {
				return true
			}
		}
	}
	return false
}

type digitRun struct {
	start, end int
	digits     string
	groups     []string
	leadPlus   bool
	internal   bool
}

func findDigitRuns(s string) []digitRun {
	var runs []digitRun
	for i := 0; i < len(s); {
		r, sz := utf8.DecodeRuneInString(s[i:])
		if r != '+' && !isDigit(r) {
			i += sz
			continue
		}
		if r == '+' && !boundaryBefore(s, i) {
			i += sz
			continue
		}
		start := i
		lead := false
		if r == '+' {
			j := skipSpaces(s, i+sz)
			if !isDigitAt(s, j) {
				i += sz
				continue
			}
			lead = true
			i = j
		}
		var digits, cur strings.Builder
		var groups []string
		internal := false
		for i < len(s) {
			r, sz = utf8.DecodeRuneInString(s[i:])
			if !isDigit(r) {
				break
			}
			digits.WriteRune(r)
			cur.WriteRune(r)
			i += sz
			if next, ok := sepsThenDigit(s, i); ok {
				internal = true
				groups = append(groups, cur.String())
				cur.Reset()
				i = next
			}
		}
		if cur.Len() > 0 {
			groups = append(groups, cur.String())
		}
		if digits.Len() == 0 {
			i = start + sz
			continue
		}
		runs = append(runs, digitRun{
			start: start, end: i, digits: digits.String(),
			groups: groups, leadPlus: lead, internal: internal,
		})
	}
	return runs
}

func sepsThenDigit(s string, i int) (int, bool) {
	j := i
	saw := false
	for j < len(s) {
		r, sz := utf8.DecodeRuneInString(s[j:])
		if isDigit(r) {
			if saw {
				return j, true
			}
			return 0, false
		}
		if !isPhoneSep(r) {
			return 0, false
		}
		saw = true
		j += sz
	}
	return 0, false
}

func isDateRun(run digitRun) bool {
	if len(run.digits) != 8 {
		return false
	}
	switch len(run.groups) {
	case 1:
		return isYMD(run.digits)
	case 3:
		a, b, c := run.groups[0], run.groups[1], run.groups[2]
		if len(a) == 4 && len(b) == 2 && len(c) == 2 {
			return isYMD(a + b + c)
		}
		if len(a) == 2 && len(b) == 2 && len(c) == 4 {
			return isYMD(c+a+b) || isYMD(c+b+a)
		}
	}
	return false
}

func isYMD(d string) bool {
	if len(d) != 8 {
		return false
	}
	y, err1 := strconv.Atoi(d[:4])
	m, err2 := strconv.Atoi(d[4:6])
	day, err3 := strconv.Atoi(d[6:8])
	if err1 != nil || err2 != nil || err3 != nil {
		return false
	}
	if y < 1900 || y > 2099 {
		return false
	}
	t := time.Date(y, time.Month(m), day, 0, 0, 0, 0, time.UTC)
	return t.Year() == y && int(t.Month()) == m && t.Day() == day
}

func protectedSpans(s string) [][2]int {
	out := findSpans(uuidRe, s)
	out = append(out, findSpans(resolutionRe, s)...)
	out = append(out, rgbSpans(s)...)
	out = append(out, findSpans(emailRe, s)...)
	return out
}

func rgbSpans(s string) [][2]int {
	var out [][2]int
	for _, m := range rgbRe.FindAllStringSubmatchIndex(s, -1) {
		if len(m) < 8 || m[2] < 0 || m[4] < 0 || m[6] < 0 {
			continue
		}
		if rgbPart(s[m[2]:m[3]]) && rgbPart(s[m[4]:m[5]]) && rgbPart(s[m[6]:m[7]]) {
			out = append(out, [2]int{m[0], m[1]})
		}
	}
	return out
}

func rgbPart(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n >= 0 && n <= 255
}

func hasSpokenPhone(s string) bool {
	dates := dateSpans(s)
	for i := 0; i < len(s); {
		r, sz := utf8.DecodeRuneInString(s[i:])
		start := false
		if isDigit(r) {
			start = i == 0 || !isDigit(prevRune(s, i))
		} else if matchNumberWord(s, i) > 0 && boundaryBefore(s, i) {
			start = true
		}
		if start {
			total, words := sequenceAt(s, i, dates)
			if words >= 1 && total >= 8 {
				return true
			}
		}
		i += sz
	}
	return false
}

func dateSpans(s string) [][2]int {
	var out [][2]int
	for _, run := range findDigitRuns(s) {
		if isDateRun(run) {
			out = append(out, [2]int{run.start, run.end})
		}
	}
	return out
}

func inSpan(i int, spans [][2]int) bool {
	for _, sp := range spans {
		if i >= sp[0] && i < sp[1] {
			return true
		}
	}
	return false
}

func sequenceAt(s string, start int, dates [][2]int) (total, words int) {
	i := start
	for i < len(s) {
		if end := matchNumberWord(s, i); end > 0 {
			words++
			total++
			i = end
		} else if i < len(s) && s[i] >= '0' && s[i] <= '9' {
			// A calendar date is not a phone, spelled out beside it or not.
			if inSpan(i, dates) {
				break
			}
			total++
			i++
		} else {
			break
		}
		j := skipSeqSeps(s, i)
		if j > i && j < len(s) && (asciiDigitAt(s, j) || matchNumberWord(s, j) > 0) {
			i = j
			continue
		}
		if i < len(s) && (asciiDigitAt(s, i) || matchNumberWord(s, i) > 0) {
			continue
		}
		break
	}
	return total, words
}

func matchNumberWord(s string, i int) int {
	for _, w := range numberWords {
		if len(s)-i < len(w) || s[i:i+len(w)] != w {
			continue
		}
		end := i + len(w)
		if asciiDigitAt(s, end) || boundaryAfter(s, end) {
			return end
		}
	}
	return 0
}

func findKeywords(s, word string) [][2]int {
	out := findPlain(s, word)
	if len(word) >= 3 {
		out = append(out, findSpaced(s, word)...)
	}
	return out
}

func findPlain(s, word string) [][2]int {
	var out [][2]int
	from := 0
	for from < len(s) {
		rel := strings.Index(s[from:], word)
		if rel < 0 {
			break
		}
		i := from + rel
		end := i + len(word)
		if boundaryBefore(s, i) && boundaryAfter(s, end) {
			out = append(out, [2]int{i, end})
		}
		from = i + 1
	}
	return out
}

func findSpaced(s, word string) [][2]int {
	var out [][2]int
	for i := 0; i < len(s); {
		r, sz := utf8.DecodeRuneInString(s[i:])
		if r == rune(word[0]) && boundaryBefore(s, i) {
			if end, ok := matchSpaced(s, i, word); ok {
				out = append(out, [2]int{i, end})
				i = end
				continue
			}
		}
		i += sz
	}
	return out
}

func matchKeywordAt(s, word string) (int, bool) {
	if word != "" && len(s) >= len(word) && s[:len(word)] == word && boundaryAfter(s, len(word)) {
		return len(word), true
	}
	if len(word) >= 3 {
		if end, ok := matchSpaced(s, 0, word); ok {
			return end, true
		}
	}
	return 0, false
}

func matchSpaced(s string, i int, word string) (int, bool) {
	for n := 0; n < len(word); n++ {
		if i >= len(s) || s[i] != word[n] {
			return 0, false
		}
		i++
		if n == len(word)-1 {
			break
		}
		if i >= len(s) {
			return 0, false
		}
		r, sz := utf8.DecodeRuneInString(s[i:])
		if !isLetterSep(r) {
			return 0, false
		}
		i += sz
		for i < len(s) {
			r, sz = utf8.DecodeRuneInString(s[i:])
			if !isLetterSep(r) {
				break
			}
			i += sz
		}
	}
	if !boundaryAfter(s, i) {
		return 0, false
	}
	return i, true
}

func lastWordIs(s, word string) bool {
	s = strings.TrimRightFunc(s, unicode.IsSpace)
	s = strings.TrimRight(s, ".,:;!?")
	if len(s) < len(word) || s[len(s)-len(word):] != word {
		return false
	}
	return boundaryBefore(s, len(s)-len(word))
}

func hasWordPrefix(s, word string) bool {
	if len(s) < len(word) || s[:len(word)] != word {
		return false
	}
	return boundaryAfter(s, len(word))
}

func findSpans(re *regexp.Regexp, s string) [][2]int {
	locs := re.FindAllStringIndex(s, -1)
	if len(locs) == 0 {
		return nil
	}
	out := make([][2]int, len(locs))
	for i, loc := range locs {
		out[i] = [2]int{loc[0], loc[1]}
	}
	return out
}

func overlaps(start, end int, spans [][2]int) bool {
	for _, sp := range spans {
		if start < sp[1] && sp[0] < end {
			return true
		}
	}
	return false
}

func nearSpan(s string, a0, a1, b0, b1, window int) bool {
	switch {
	case a1 <= b0:
		return utf8.RuneCountInString(s[a1:b0]) <= window
	case b1 <= a0:
		return utf8.RuneCountInString(s[b1:a0]) <= window
	default:
		return true
	}
}

func trimLeft(s string) string {
	return strings.TrimLeftFunc(s, unicode.IsSpace)
}

func skipSpaces(s string, i int) int {
	for i < len(s) {
		r, sz := utf8.DecodeRuneInString(s[i:])
		if !unicode.IsSpace(r) {
			break
		}
		i += sz
	}
	return i
}

func skipSeqSeps(s string, i int) int {
	for i < len(s) {
		r, sz := utf8.DecodeRuneInString(s[i:])
		if !isSeqSep(r) {
			break
		}
		i += sz
	}
	return i
}

func isPhoneSep(r rune) bool {
	switch r {
	case '.', '-', '/', '(', ')', '\u2013', '\u2014':
		return true
	default:
		return unicode.IsSpace(r)
	}
}

func isSeqSep(r rune) bool {
	return isPhoneSep(r) || r == ','
}

func isLetterSep(r rune) bool {
	switch r {
	case '.', '-', '_', '*':
		return true
	default:
		return unicode.IsSpace(r)
	}
}

func isDigit(r rune) bool { return r >= '0' && r <= '9' }

func isDigitAt(s string, i int) bool {
	if i >= len(s) {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s[i:])
	return isDigit(r)
}

func asciiDigitAt(s string, i int) bool {
	return i < len(s) && s[i] >= '0' && s[i] <= '9'
}

func prevRune(s string, i int) rune {
	r, _ := utf8.DecodeLastRuneInString(s[:i])
	return r
}

func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func boundaryBefore(s string, i int) bool {
	if i <= 0 {
		return true
	}
	return !isWordRune(prevRune(s, i))
}

func boundaryAfter(s string, i int) bool {
	if i >= len(s) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(s[i:])
	return !isWordRune(r)
}
