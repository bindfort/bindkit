package logging

import "strings"

func Redact(value string) string {
	out := value
	for _, marker := range []string{"Bearer ", "api_key=", "apiKey=", "token=", "secret="} {
		idx := strings.Index(out, marker)
		for idx >= 0 {
			start := idx + len(marker)
			end := start
			for end < len(out) && !strings.ContainsAny(string(out[end]), " \t\r\n&") {
				end++
			}
			if end > start {
				out = out[:start] + "REDACTED" + out[end:]
			}
			idx = strings.Index(out[start:], marker)
			if idx >= 0 {
				idx += start
			}
		}
	}
	return out
}
