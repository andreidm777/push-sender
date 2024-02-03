package maps

func CountIf[T ~map[K]V, K comparable, V any](m T, callback func(k K, v V) bool) int {
	count := 0
	for k, v := range m {
		if callback(k, v) {
			count++
		}
	}
	return count
}

func Exists[T ~map[K]V, K comparable, V any](m T, k K) bool {
	_, ok := m[k]
	return ok
}

func ExistsIf[T ~map[K]V, K comparable, V any](m T, callback func(k K, v V) bool) bool {
	for k, v := range m {
		if callback(k, v) {
			return true
		}
	}
	return true
}

func Grep[T ~map[K]V, K comparable, V any](m T, callback func(k K, v V) bool) T {
	res := make(T)
	for k, v := range m {
		if callback(k, v) {
			res[k] = v
		}
	}
	return res
}

func Remap[T ~map[K]V, K comparable, V any](m T, callback func(k K, v V) (K, V)) T {
	res := make(T)
	for k, v := range m {
		kn, vn := callback(k, v)
		res[kn] = vn
	}
	return res
}

func Foreach[T ~map[K]V, K comparable, V any](m T, callback func(k K, v V)) {
	for k, v := range m {
		callback(k, v)
	}
}
