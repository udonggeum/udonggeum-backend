package kakaopay

// Config represents the configuration for the Kakao Pay client
type Config struct {
	// AdminKey is the Kakao Pay admin key for API authentication
	AdminKey string

	// CID is the Client ID (merchant code)
	CID string

	// BaseURL is the Kakao Pay API base URL
	BaseURL string

	// ApprovalURL is the redirect URL for successful payment
	ApprovalURL string

	// FailURL is the redirect URL for failed payment
	FailURL string

	// CancelURL is the redirect URL for cancelled payment
	CancelURL string
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.AdminKey == "" {
		return ErrInvalidRequest
	}
	if c.CID == "" {
		return ErrInvalidRequest
	}
	if c.BaseURL == "" {
		return ErrInvalidRequest
	}
	if c.ApprovalURL == "" {
		return ErrInvalidRequest
	}
	if c.FailURL == "" {
		return ErrInvalidRequest
	}
	if c.CancelURL == "" {
		return ErrInvalidRequest
	}
	return nil
}
