package slices

func CountIf[T ~[]V, V any](s T, callback func(v V) bool) int {
	count := 0
	for _, v := range s {
		if callback(v) {
			count++
		}
	}
	return count
}

func Grep[T ~[]V, V any](s T, callback func(v V) bool) T {
	res := make(T, 0, len(s))
	for _, v := range s {
		if callback(v) {
			res = append(res, v)
		}
	}
	return res
}
