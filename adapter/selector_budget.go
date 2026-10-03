package adapter

import "strings"

const selectorWorkLimit = 10_000_000

func selectorAdmission(query string, stats *treeStats, single bool) error {
	if stats == nil {
		stats = &treeStats{nodes: 1}
	}
	work := len(query) + 1
	if !single {
		if work > selectorWorkLimit/stats.nodes {
			return selectorWorkError()
		}
		work *= stats.nodes
	}
	charge := func(factor int) error {
		factor = max(factor, 1)
		if work > selectorWorkLimit/factor {
			return selectorWorkError()
		}
		work *= factor
		return nil
	}
	var previous byte
	depth := 0
	for i := 0; i < len(query); {
		c := query[i]
		if cssSpace(c) || strings.HasPrefix(query[i:], "/*") {
			next := skipCSSSpace(query, i)
			if previous != 0 && !strings.ContainsRune(">+~,(", rune(previous)) && next < len(query) && !strings.ContainsRune(">+~,)", rune(query[next])) {
				if err := charge(stats.depth + 1); err != nil {
					return err
				}
			}
			i = next
			continue
		}
		switch c {
		case '\\':
			return failure{6, "bounded CSS selectors do not support escapes outside quoted attribute values"}
		case '\'', '"':
			return failure{6, "bounded CSS selectors require strings to occur inside attribute selectors"}
		case '[':
			if depth >= 32 {
				return failure{6, "CSS selector nesting exceeds 32"}
			}
			var quote byte
			for i++; i < len(query); i++ {
				c = query[i]
				if quote != 0 {
					if c == '\\' {
						i++
					} else if c == quote {
						quote = 0
					}
					continue
				}
				if c == ']' {
					break
				}
				if c == '\\' || c == '[' || strings.HasPrefix(query[i:], "#=") {
					return failure{6, "bounded CSS attribute selectors do not support unquoted escapes, nesting or regular expressions"}
				}
				if c == '\'' || c == '"' {
					quote = c
				}
			}
			previous = ']'
			i++
			continue
		case '(':
			depth++
			if depth > 32 {
				return failure{6, "CSS selector nesting exceeds 32"}
			}
		case ')':
			depth--
		case '~', '+':
			if err := charge(stats.children + 1); err != nil {
				return err
			}
		case ':':
			start := i + 1
			end := start
			for end < len(query) && ((query[end] >= 'a' && query[end] <= 'z') || (query[end] >= 'A' && query[end] <= 'Z') || query[end] == '-') {
				end++
			}
			name := strings.ToLower(query[start:end])
			factor := 1
			skipArgument := false
			switch name {
			case "not", "root", "link", "input", "checked":
			case "has":
				factor = stats.nodes
			case "haschild", "first-child", "last-child", "first-of-type", "last-of-type", "only-child", "only-of-type", "empty":
				factor = stats.children + 1
			case "nth-child", "nth-last-child", "nth-of-type", "nth-last-of-type":
				factor = stats.children + 1
				skipArgument = true
			case "lang":
				factor = stats.depth + 1
				skipArgument = true
			case "enabled", "disabled":
				factor = stats.nodes
			default:
				return failure{6, "pseudo-class is not supported by bounded CSS selector admission: " + name}
			}
			if err := charge(factor); err != nil {
				return err
			}
			if skipArgument {
				if depth >= 32 {
					return failure{6, "CSS selector nesting exceeds 32"}
				}
				closing := strings.IndexByte(query[end:], ')')
				if closing < 0 || strings.ContainsRune(query[end:end+closing], '\\') {
					return failure{6, "bounded CSS pseudo-class arguments do not support escapes"}
				}
				i = end + closing + 1
				previous = ')'
				continue
			}
		}
		previous = c
		i++
	}
	return nil
}

func selectorWorkError() error {
	return failure{11, "CSS selector estimated structural work exceeds 10000000 operations"}
}

func cssSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\f'
}

func skipCSSSpace(query string, start int) int {
	for start < len(query) {
		if cssSpace(query[start]) {
			start++
		} else if strings.HasPrefix(query[start:], "/*") {
			end := strings.Index(query[start+2:], "*/")
			if end < 0 {
				return len(query)
			}
			start += end + 4
		} else {
			break
		}
	}
	return start
}
