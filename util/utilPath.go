package util

import (
	"os"
	"strings"
)

func calPathDelimiter(delimiter ...string) string {
	if len(delimiter) > 0 {
		return delimiter[0]
	} else {
		return string(os.PathSeparator)
	}
}

func RemovePathDelimiterPrefix(path string) string {
	if path == "" {
		path = "."
	}
	for strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
		path = string([]rune(path)[1:])
	}
	return path
}

func RemovePathDelimiterSuffix(path string) string {
	if path == "" {
		path = "."
	}
	for strings.HasSuffix(path, "/") || strings.HasSuffix(path, "\\") {
		ra := []rune(path)
		path = string(ra[:len(ra)-1])
	}
	return path
}

func AppendPathDelimiter(path string, delimiter ...string) string {
	return RemovePathDelimiterSuffix(path) + calPathDelimiter(delimiter...)
}

func PrependPathDelimiter(path string, delimiter ...string) string {
	return calPathDelimiter(delimiter...) + RemovePathDelimiterPrefix(path)
}

func JoinPathWith(delimiter string, pathNode ...string) string {
	switch len(pathNode) {
	case 0:
		return "."
	case 1:
		if pathNode[0] == "" {
			return "."
		} else {
			return pathNode[0]
		}
	default:
		var sb strings.Builder
		for i, p := range pathNode {
			if i > 0 {
				p = PrependPathDelimiter(p, delimiter)
			}
			if i < len(pathNode)-1 {
				p = RemovePathDelimiterSuffix(p)
			}
			sb.WriteString(p)
		}
		return sb.String()
	}
}

func JoinPath(pathNode ...string) string {
	return JoinPathWith(string(os.PathSeparator), pathNode...)
}
