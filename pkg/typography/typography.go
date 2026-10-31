package typography

import (
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

// smallCapsMap maps Latin letters (both lowercase and uppercase) to Unicode Small Capitals
var smallCapsMap = map[rune]rune{
	'a': 'ᴀ', 'A': 'ᴀ',
	'b': 'ʙ', 'B': 'ʙ',
	'c': 'ᴄ', 'C': 'ᴄ',
	'd': 'ᴅ', 'D': 'ᴅ',
	'e': 'ᴇ', 'E': 'ᴇ',
	'f': 'ғ', 'F': 'ғ',
	'g': 'ɢ', 'G': 'ɢ',
	'h': 'ʜ', 'H': 'ʜ',
	'i': 'ɪ', 'I': 'ɪ',
	'j': 'ᴊ', 'J': 'ᴊ',
	'k': 'ᴋ', 'K': 'ᴋ',
	'l': 'ʟ', 'L': 'ʟ',
	'm': 'ᴍ', 'M': 'ᴍ',
	'n': 'ɴ', 'N': 'ɴ',
	'o': 'ᴏ', 'O': 'ᴏ',
	'p': 'ᴘ', 'P': 'ᴘ',
	'q': 'ǫ', 'Q': 'ǫ',
	'r': 'ʀ', 'R': 'ʀ',
	's': 'ꜱ', 'S': 'ꜱ',
	't': 'ᴛ', 'T': 'ᴛ',
	'u': 'ᴜ', 'U': 'ᴜ',
	'v': 'ᴠ', 'V': 'ᴠ',
	'w': 'ᴡ', 'W': 'ᴡ',
	'x': 'x', 'X': 'x',
	'y': 'ʏ', 'Y': 'ʏ',
	'z': 'ᴢ', 'Z': 'ᴢ',
}

// ToSmallCaps converts plain alphabetic characters to Unicode Small Caps
func ToSmallCaps(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))

	for _, r := range s {
		if sc, ok := smallCapsMap[r]; ok {
			sb.WriteRune(sc)
		} else {
			sb.WriteRune(r)
		}
	}

	return sb.String()
}

// ToSmallCapsHTML converts message text to Small Caps while safely preserving:
// 1. HTML tags (<...>) and their attributes
// 2. HTML entities (&...;)
// 3. Raw user-provided escape blocks (<raw>...</raw>)
// 4. Web URLs (http://, https://, tg://)
func ToSmallCapsHTML(s string) string {
	if s == "" {
		return ""
	}

	var sb strings.Builder
	sb.Grow(len(s))

	runes := []rune(s)
	n := len(runes)
	i := 0

	for i < n {
		// 1. Check for <raw>...</raw> block to preserve user content verbatim
		if i+5 <= n && string(runes[i:i+5]) == "<raw>" {
			i += 5
			rawStart := i
			for i < n {
				if i+6 <= n && string(runes[i:i+6]) == "</raw>" {
					sb.WriteString(string(runes[rawStart:i]))
					i += 6
					break
				}
				i++
			}
			continue
		}

		// 2. Check for HTML tags: <...>
		if runes[i] == '<' {
			tagStart := i
			for i < n && runes[i] != '>' {
				i++
			}
			if i < n && runes[i] == '>' {
				i++
			}
			sb.WriteString(string(runes[tagStart:i]))
			continue
		}

		// 3. Check for HTML entities: &...;
		if runes[i] == '&' {
			entStart := i
			for i < n && runes[i] != ';' && runes[i] != ' ' && runes[i] != '<' {
				i++
			}
			if i < n && runes[i] == ';' {
				i++
				sb.WriteString(string(runes[entStart:i]))
				continue
			}
			// Not a valid entity, rewind
			i = entStart
		}

		// 4. Check for URLs: http:// or https:// or tg://
		isURL := false
		if i+7 <= n && (string(runes[i:i+7]) == "http://" || string(runes[i:i+5]) == "tg://") {
			isURL = true
		} else if i+8 <= n && string(runes[i:i+8]) == "https://" {
			isURL = true
		}

		if isURL {
			urlStart := i
			for i < n && runes[i] != ' ' && runes[i] != '\n' && runes[i] != '<' && runes[i] != '"' && runes[i] != '\'' {
				i++
			}
			sb.WriteString(string(runes[urlStart:i]))
			continue
		}

		// 5. Normal text character
		r := runes[i]
		if sc, ok := smallCapsMap[r]; ok {
			sb.WriteRune(sc)
		} else {
			sb.WriteRune(r)
		}
		i++
	}

	return sb.String()
}

// FormatMarkup walks any inline reply markup and converts all button text to Small Caps
// while leaving callback data and URLs completely intact.
func FormatMarkup(markup tg.ReplyMarkup) tg.ReplyMarkup {
	if markup == nil {
		return nil
	}

	inlineMarkup, ok := markup.(*tg.ReplyInlineMarkup)
	if !ok || inlineMarkup == nil {
		return markup
	}

	for _, row := range inlineMarkup.Rows {
		if row == nil {
			continue
		}
		for _, btn := range row.Buttons {
			switch b := btn.(type) {
			case *tg.KeyboardButtonCallback:
				b.Text = ToSmallCaps(b.Text)
			case *tg.KeyboardButtonURL:
				b.Text = ToSmallCaps(b.Text)
			case *tg.KeyboardButtonObj:
				b.Text = ToSmallCaps(b.Text)
			}
		}
	}

	return inlineMarkup
}
