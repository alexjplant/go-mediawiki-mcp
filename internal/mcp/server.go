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
			item["text"] = c.TextContent.Text
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
