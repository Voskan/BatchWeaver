package runtime

import "fmt"

// sanitizePanic converts a recovered panic value into a short, non-sensitive
// string. It never includes a stack trace or raw key material.
func sanitizePanic(r any) string {
	switch v := r.(type) {
	case error:
		return v.Error()
	case string:
		return v
	default:
		return fmt.Sprintf("%T", r)
	}
}
