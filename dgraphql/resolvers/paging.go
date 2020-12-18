package resolvers

type PageInfo struct {
	StartCursor string
	EndCursor   string
}

func NewPageInfo(startCursor string, endCursor string) PageInfo {
	return PageInfo{
		StartCursor: startCursor,
		EndCursor:   endCursor,
	}
}
