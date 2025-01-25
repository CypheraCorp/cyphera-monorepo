package actalink

type ActaLinkClient struct {
	apiKey string
}

func NewActaLinkClient(apiKey string) *ActaLinkClient {
	return &ActaLinkClient{apiKey: apiKey}
}
