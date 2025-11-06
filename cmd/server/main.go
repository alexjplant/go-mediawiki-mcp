package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/alexjplant/go-mcp-server/internal/mcp"
)

func main() {
	apiURL := flag.String("api-url", "https://wiki.fractalaudio.com/wiki/api.php", "MediaWiki API URL")
	flag.Parse()

	if apiURL == nil || *apiURL == "" {
		log.Fatal("MediaWiki API URL is required")
	}

	server, err := mcp.NewServer(*apiURL)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	http.Handle("/mcp", server.HTTPHandler())
	log.Fatal(http.ListenAndServe(":8787", nil))
}
