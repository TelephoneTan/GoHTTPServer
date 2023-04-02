package types

import (
	"github.com/TelephoneTan/GoHTTPServer/util"
	"regexp"
	"strings"
)

type WordList [][]string

func (w *WordList) Join() string {
	for _, words := range *w {
		if len(words) == 0 {
			continue
		}
		ok := true
		for _, word := range words {
			ok = ok && util.IsPureASCII(word)
			if !ok {
				break
			}
		}
		if !ok {
			continue
		}
		return strings.Join(words, ".")
	}
	return ""
}

func (w *WordList) Match(s string) bool {
	var sb strings.Builder
	sb.WriteString(`(?i)^.^`)
	for _, words := range *w {
		sb.WriteString(`|^` + strings.Join(words, `[.\-_,，。]*`) + `$`)
	}
	return regexp.MustCompile(sb.String()).MatchString(s)
}
