package util

func ShallowCloneSlice[T any](src []T) []T {
	return append(make([]T, 0), src...)
}

func IsPureASCII(s string) bool {
	for _, c := range s {
		if c < 0 || c > 127 {
			return false
		}
	}
	return true
}

func New[T any](t *T, init ...func(*T)) *T {
	if len(init) > 0 {
		init[0](t)
	}
	return t
}
