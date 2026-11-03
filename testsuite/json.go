package testsuite

import "encoding/json"

func Jsonify[T any](v T) (s string) {
	b, _ := json.MarshalIndent(v, "", "\t")
	return string(b)
}
