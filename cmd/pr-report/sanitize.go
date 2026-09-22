package main

import (
	"strings"
	"unicode"
)

// terminalText strips ANSI CSI and control strings, including OSC hyperlinks,
// then removes remaining terminal controls and invisible format characters.
func terminalText(s string) string {
	r := []rune(s)
	var b strings.Builder
	for i := 0; i < len(r); i++ {
		c := r[i]
		kind := rune(0)
		if c == 0x1b {
			i++
			if i >= len(r) {
				break
			}
			c = r[i]
			switch c {
			case '[':
				kind = 'c'
			case ']', 'P', 'X', '^', '_':
				kind = 's'
			default:
				for i < len(r) && r[i] >= 0x20 && r[i] <= 0x2f {
					i++
				}
				continue
			}
		}
		if c == 0x9b {
			kind = 'c'
		}
		if c == 0x9d || c == 0x90 || c == 0x98 || c == 0x9e || c == 0x9f {
			kind = 's'
		}
		if kind == 'c' {
			for i++; i < len(r); i++ {
				if r[i] >= 0x40 && r[i] <= 0x7e {
					break
				}
			}
			continue
		}
		if kind == 's' {
			for i++; i < len(r); i++ {
				if r[i] == 7 || r[i] == 0x9c {
					break
				}
				if r[i] == 0x1b && i+1 < len(r) && r[i+1] == '\\' {
					i++
					break
				}
			}
			continue
		}
		if unicode.IsControl(c) || unicode.Is(unicode.Cf, c) || c == '\u2028' || c == '\u2029' {
			continue
		}
		b.WriteRune(c)
	}
	return b.String()
}
func tableTitle(s string) string {
	r := []rune(terminalText(s))
	if len(r) > 60 {
		return string(r[:59]) + "…"
	}
	return string(r)
}
