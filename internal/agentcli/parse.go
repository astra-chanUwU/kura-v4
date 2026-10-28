package agentcli

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Args is a parsed agent invocation. Flags are present only when the
// caller passed them, so an empty value is different from an omitted flag.
type Args struct {
	Pos   []string
	flags map[string]string
}

func (a Args) Has(name string) bool {
	_, ok := a.flags[name]
	return ok
}

func (a Args) Get(name string) string {
	return a.flags[name]
}

// Only rejects flags the command does not accept.
func (a Args) Only(allowed ...string) error {
	ok := map[string]bool{}
	for _, name := range allowed {
		ok[name] = true
	}
	for name := range a.flags {
		if !ok[name] {
			return fmt.Errorf("unknown flag --%s", name)
		}
	}
	return nil
}

// Int reads a flag, or returns def when it was omitted.
func (a Args) Int(name string, def, min, max int) (int, error) {
	if !a.Has(name) {
		return def, nil
	}
	n, err := strconv.Atoi(a.Get(name))
	if err != nil {
		return 0, fmt.Errorf("--%s must be an integer", name)
	}
	if n < min || n > max {
		return 0, fmt.Errorf("--%s must be from %d to %d", name, min, max)
	}
	return n, nil
}

func parseArgs(args []string) (Args, error) {
	out := Args{flags: map[string]string{}}
	for i := 0; i < len(args); i++ {
		tok := args[i]
		if tok == "--" {
			out.Pos = append(out.Pos, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(tok, "--") {
			out.Pos = append(out.Pos, tok)
			continue
		}
		name, value, hasValue := strings.Cut(tok[2:], "=")
		if !validFlag(name) {
			return Args{}, fmt.Errorf("unknown flag --%s", name)
		}
		if !hasValue {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				return Args{}, fmt.Errorf("missing value for --%s", name)
			}
			i++
			value = args[i]
		}
		if _, exists := out.flags[name]; exists {
			return Args{}, fmt.Errorf("repeated flag --%s", name)
		}
		out.flags[name] = value
	}
	return out, nil
}

func validFlag(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' {
			return false
		}
	}
	return true
}
