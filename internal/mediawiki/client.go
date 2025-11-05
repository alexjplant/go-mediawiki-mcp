package mediawiki

import (
	"os"

	"gitlab.com/melancholera/go-mwclient"
)

type Client struct {
	mw *mwclient.Client
}

func NewClient(apiURL string) (*Client, error) {
	userAgent := os.Getenv("MW_USER_AGENT")
	if userAgent == "" {
		userAgent = "go-mediawiki-mcp/1.0"
	}

	mw, err := mwclient.New(apiURL, userAgent)
	if err != nil {
		return nil, err
	}

	return &Client{mw: mw}, nil
}

func (c *Client) GetPage(title string) (string, string, error) {
	return c.mw.GetPageByName(title)
}

