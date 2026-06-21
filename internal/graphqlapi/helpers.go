package graphqlapi

func derefString(raw *string) string {
	if raw == nil {
		return ""
	}
	return *raw
}
