package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/alexjplant/go-mcp-server/internal/mediawiki"
	mcp_golang "github.com/metoro-io/mcp-golang"
)

type Server struct {
	mwClient *mediawiki.Client
}

func NewServer(apiURL string) (*Server, error) {
	mwClient, err := mediawiki.NewClient(apiURL)
	if err != nil {
		return nil, err
	}

	return &Server{mwClient: mwClient}, nil
}

func (s *Server) HTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read body: %v", err), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var req map[string]interface{}
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
			return
		}

		method, ok := req["method"].(string)
		if !ok {
			http.Error(w, "Missing method", http.StatusBadRequest)
			return
		}

		var response interface{}
		switch method {
		case "tools/call":
			result, err := s.handleCallTool(req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			response = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result":  result,
			}
		case "tools/list":
			response = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result":  s.handleListTools(),
			}
		case "initialize":
			response = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result":  s.handleInitialize(),
			}
		default:
			http.Error(w, fmt.Sprintf("Unknown method: %s", method), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
}

func convertContent(content []*mcp_golang.Content) []map[string]interface{} {
	result := make([]map[string]interface{}, len(content))
	for i, c := range content {
		item := map[string]interface{}{
			"type": c.Type,
		}
		if c.TextContent != nil {
			text := c.TextContent.Text
			var jsonData interface{}
			if json.Unmarshal([]byte(text), &jsonData) == nil {
				item["json"] = jsonData
			} else {
				item["text"] = text
			}
		}
		if c.ImageContent != nil {
			item["image"] = c.ImageContent
		}
		if c.EmbeddedResource != nil {
			item["resource"] = c.EmbeddedResource
		}
		result[i] = item
	}
	return result
}

func (s *Server) handleCallTool(req map[string]interface{}) (map[string]interface{}, error) {
	params, ok := req["params"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid params")
	}

	name, ok := params["name"].(string)
	if !ok {
		return nil, fmt.Errorf("missing tool name")
	}

	arguments, ok := params["arguments"].(map[string]interface{})
	if !ok {
		arguments = make(map[string]interface{})
	}

	if name == "get_page" {
		title, ok := arguments["title"].(string)
		if !ok {
			return nil, fmt.Errorf("missing title argument")
		}
		result, err := s.handleGetPage(GetPageArguments{Title: title})
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"content": convertContent(result.Content),
			"isError": false,
		}, nil
	}

	if name == "get_toc" {
		title, ok := arguments["title"].(string)
		if !ok {
			return nil, fmt.Errorf("missing title argument")
		}
		result, err := s.handleGetTOC(GetTOCArguments{Title: title})
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"content": convertContent(result.Content),
			"isError": false,
		}, nil
	}

	if name == "get_section" {
		title, ok := arguments["title"].(string)
		if !ok {
			return nil, fmt.Errorf("missing title argument")
		}
		sectionIndex, ok := arguments["section_index"].(float64)
		if !ok {
			return nil, fmt.Errorf("missing section_index argument")
		}
		result, err := s.handleGetSection(GetSectionArguments{Title: title, SectionIndex: int(sectionIndex)})
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"content": convertContent(result.Content),
			"isError": false,
		}, nil
	}

	return nil, fmt.Errorf("unknown tool: %s", name)
}

func (s *Server) handleListTools() map[string]interface{} {
	return map[string]interface{}{
		"tools": []map[string]interface{}{
			{
				"name":        "get_page",
				"description": "Get the content of a MediaWiki page by title",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"title": map[string]interface{}{
							"type":        "string",
							"description": "The title of the page to retrieve",
						},
					},
					"required": []string{"title"},
				},
			},
			{
				"name":        "get_toc",
				"description": "Get the table of contents (TOC) for a MediaWiki page as a JSON object",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"title": map[string]interface{}{
							"type":        "string",
							"description": "The title of the page to get the TOC for",
						},
					},
					"required": []string{"title"},
				},
			},
			{
				"name":        "get_section",
				"description": "Get a specific section from a MediaWiki page",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"title": map[string]interface{}{
							"type":        "string",
							"description": "The title of the page",
						},
						"section_index": map[string]interface{}{
							"type":        "integer",
							"description": "The index of the section to retrieve (0-based)",
						},
					},
					"required": []string{"title", "section_index"},
				},
			},
		},
	}
}

func (s *Server) handleInitialize() map[string]interface{} {
	return map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "mediawiki-mcp-server",
			"version": "1.0.0",
		},
	}
}

type GetPageArguments struct {
	Title string `json:"title" jsonschema:"required,description=The title of the page to retrieve"`
}

func (s *Server) handleGetPage(arguments GetPageArguments) (*mcp_golang.ToolResponse, error) {
	content, timestamp, err := s.mwClient.GetPage(arguments.Title)
	if err != nil {
		return mcp_golang.NewToolResponse(
			mcp_golang.NewTextContent(fmt.Sprintf("Error retrieving page: %v", err)),
		), nil
	}

	result := fmt.Sprintf("Title: %s\nLast Edited: %s\n\n%s", arguments.Title, timestamp, content)
	return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(result)), nil
}

type GetTOCArguments struct {
	Title string `json:"title" jsonschema:"required,description=The title of the page to get the TOC for"`
}

func (s *Server) handleGetTOC(arguments GetTOCArguments) (*mcp_golang.ToolResponse, error) {
	sections, err := s.mwClient.GetTOC(arguments.Title)
	if err != nil {
		return mcp_golang.NewToolResponse(
			mcp_golang.NewTextContent(fmt.Sprintf("Error retrieving TOC: %v", err)),
		), nil
	}

	tocJSON, err := json.Marshal(sections)
	if err != nil {
		return mcp_golang.NewToolResponse(
			mcp_golang.NewTextContent(fmt.Sprintf("Error marshaling TOC: %v", err)),
		), nil
	}

	return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(string(tocJSON))), nil
}

type GetSectionArguments struct {
	Title        string `json:"title" jsonschema:"required,description=The title of the page"`
	SectionIndex int    `json:"section_index" jsonschema:"required,description=The index of the section to retrieve (0-based)"`
}

func (s *Server) handleGetSection(arguments GetSectionArguments) (*mcp_golang.ToolResponse, error) {
	content, err := s.mwClient.GetSection(arguments.Title, arguments.SectionIndex)
	if err != nil {
		return mcp_golang.NewToolResponse(
			mcp_golang.NewTextContent(fmt.Sprintf("Error retrieving section: %v", err)),
		), nil
	}

	return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(content)), nil
}
