package resolvers

type PageInfo struct {
	StartCursor string
	EndCursor   string
	HasNextPage bool
}

func NewPageInfo(startCursor string, endCursor string, hasNextPage bool) PageInfo {
	return PageInfo{
		StartCursor: startCursor,
		EndCursor:   endCursor,
		HasNextPage: hasNextPage,
	}
}
