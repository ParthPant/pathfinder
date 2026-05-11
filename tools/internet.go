package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/iamwavecut/go-tavily"
)

var tavilyInitOnce sync.Once
var tavilyClient *tavily.Client

func getTavilyClient() *tavily.Client {
	if tavilyClient != nil {
		return tavilyClient
	}

	tavilyInitOnce.Do(func() {
		tavilyClient = tavily.New(os.Getenv("TAVILY_API_KEY"), nil)
	})
	return tavilyClient
}

// Internet search tool

var InternetSearchTool = FunctionDefinition{
	Type:        "function",
	Name:        "internet_search",
	Description: "Use this tool to perform an internet search at any time. This is usefull when you want to fetch information about things you do not know, or if the user explicitly asks to use this tool.",
	Parameters:  ParamsFor[InternetSearchInput](),
	Strict:      false,
	Function:    InternetSearch,
}

type InternetSearchInput struct {
	Query string `json:"query" tool:"A search query to fetch results from the internet,required"`
}

func InternetSearch(ctx context.Context, params InternetSearchInput) (any, error) {
	client := getTavilyClient()
	if client == nil {
		return nil, errors.New("Tavily client is not initialized.")
	}

	opts := tavily.SearchOptions{
		SearchDepth:   string(tavily.SearchDepthBasic),
		MaxResults:    10,
		IncludeAnswer: true,
		IncludeImages: new(false),
	}
	res, err := client.Search(ctx, params.Query, &opts)
	if err != nil {
		return nil, err
	}

	builder := strings.Builder{}
	fmt.Fprintf(&builder, "Answer: %s\n\n", res.Answer)
	fmt.Fprintf(&builder, "Results:\n\n")
	for i, result := range res.Results {
		fmt.Fprintf(&builder, "[%d] Title: %s\nURL: %s\nContent: %s\n\n", i, result.Title, result.URL, result.Content)
	}

	return builder.String(), nil
}

// Open URL

var OpenURLTool = FunctionDefinition{
	Type:        "function",
	Name:        "open_url",
	Description: "Use this tool to read contents for a web URL.",
	Parameters:  ParamsFor[OpenURLInput](),
	Strict:      false,
	Function:    OpenUrl,
}

type OpenURLInput struct {
	URL string `json:"string" tool:"A web URL,required"`
}

func OpenUrl(ctx context.Context, params OpenURLInput) (any, error) {
	client := getTavilyClient()
	if client == nil {
		return nil, errors.New("Tavily client is not initialized.")
	}

	res, err := client.ExtractSimple(ctx, params.URL)
	if err != nil {
		return nil, err
	}

	builder := strings.Builder{}
	for _, result := range res.Results {
		builder.Write([]byte(result.RawContent))
	}

	return builder.String(), nil
}
