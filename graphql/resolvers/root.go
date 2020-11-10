package resolvers

// Root is the root resolvers.
type Root struct {
	rpcURL string
	wsURL  string
}

func NewRoot(rpcURL string, wsURL string) *Root {
	return &Root{
		rpcURL: rpcURL,
		wsURL:  wsURL,
	}
}
