package providers

type ProviderParams struct {
	BaseURL string
	APIKey  string
}

type ProviderOption func(*ProviderParams)

func WithBaseURL(baseURL string) ProviderOption {
	return func(p *ProviderParams) {
		p.BaseURL = baseURL
	}
}

func WithAPIKey(apiKey string) ProviderOption {
	return func(p *ProviderParams) {
		p.APIKey = apiKey
	}
}
