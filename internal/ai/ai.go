package ai

type CommitGroup struct {
	Files      []string `json:"files"`
	Subject    string   `json:"subject"`
	Body       string   `json:"body"`
	ChangeType string   `json:"changeType"`
	Scope      string   `json:"scope"`
}

type CommitSuggestion struct {
	Groups []CommitGroup `json:"groups"`
}

type Client struct {
	APIKey string
	Model  string
}

func NewClient(apiKey string, model string) *Client {
	return &Client{
		APIKey: apiKey,
		Model:  model,
	}
}

func (c *Client) GenerateCommitMessage(diff string, context string) (*CommitSuggestion, error) {
	return nil, nil
}
