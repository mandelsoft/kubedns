package common

func String(s *string, def string) string {
	if s == nil {
		return def
	}
	return *s
}
