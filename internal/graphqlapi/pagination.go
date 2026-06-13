package graphqlapi

func normalizePage(page *int) int {
	if page == nil || *page < 1 {
		return 1
	}
	return *page
}

func normalizePageSize(pageSize *int) int {
	if pageSize == nil || *pageSize <= 0 {
		return 24
	}
	if *pageSize > 100 {
		return 100
	}
	return *pageSize
}
