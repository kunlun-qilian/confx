package confx

import (
	"runtime"
	"strings"
)

type Password string

func (p Password) String() string {
	return string(p)
}

func (p Password) SecurityString() string {
	var r []rune
	for range []rune(string(p)) {
		r = append(r, []rune("-")...)
	}
	return string(r)
}

func ShouldReplacePath(s string) string {
	if runtime.GOOS != "windows" {
		return s
	}
	return strings.ReplaceAll(s, `\`, `/`)
}
