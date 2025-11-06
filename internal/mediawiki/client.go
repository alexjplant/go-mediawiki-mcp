package mediawiki

import (
	"encoding/json"
	"fmt"
	"os"

	"gitlab.com/melancholera/go-mwclient"
	"gitlab.com/melancholera/go-mwclient/params"
)

type Client struct {
	mw     *mwclient.Client
	apiURL string
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

	return &Client{mw: mw, apiURL: apiURL}, nil
}

func (c *Client) GetPage(title string) (string, string, error) {
	return c.mw.GetPageByName(title)
}

type Section struct {
	TOCLevel int    `json:"toclevel"`
	Level    string `json:"level"`
	Line     string `json:"line"`
	Number   string `json:"number"`
	Index    string `json:"index"`
	FromID   string `json:"fromid"`
	ToID     string `json:"toid"`
	Anchor   string `json:"anchor"`
}

func (c *Client) GetTOC(title string) ([]Section, error) {
	p := params.Values{}
	p.Set("action", "parse")
	p.Set("page", title)
	p.Set("prop", "sections")

	raw, err := c.mw.GetRaw(p)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch TOC: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	parse, ok := result["parse"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format: missing parse")
	}

	sectionsData, ok := parse["sections"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format: missing sections")
	}

	sectionsJSON, err := json.Marshal(sectionsData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sections: %w", err)
	}

	var sections []Section
	if err := json.Unmarshal(sectionsJSON, &sections); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sections: %w", err)
	}

	return sections, nil
}

func (c *Client) GetSection(title string, sectionIndex int) (string, error) {
	p := params.Values{}
	p.Set("action", "parse")
	p.Set("page", title)
	p.Set("section", fmt.Sprintf("%d", sectionIndex))

	raw, err := c.mw.GetRaw(p)
	if err != nil {
		return "", fmt.Errorf("failed to fetch section: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	parse, ok := result["parse"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response format: missing parse")
	}

	text, ok := parse["text"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response format: missing text")
	}

	return text, nil
}
