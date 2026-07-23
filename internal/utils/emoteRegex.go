package utils

import "regexp"

var EmoteRegex = regexp.MustCompile(`<(a?):(\w+):(\d+)>`)

type EmoteMatch struct {
	ID       string
	Name     string
	Animated bool
}

func ExtractEmotes(content string) []EmoteMatch {
	matches := EmoteRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}
	result := make([]EmoteMatch, 0, len(matches))
	for _, m := range matches {
		result = append(result, EmoteMatch{
			ID:       m[3],
			Name:     m[2],
			Animated: m[1] == "a",
		})
	}
	return result
}
