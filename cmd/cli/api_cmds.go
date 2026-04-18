package main

import (
	"flag"
	"fmt"
	"os"
)

func cmdAuth(args []string) {
	fs := flag.NewFlagSet("auth", flag.ExitOnError)
	token := fs.String("token", "", "API Token")
	fs.Parse(args)

	if *token == "" {
		fmt.Fprintln(os.Stderr, "Error: missing --token")
		os.Exit(1)
	}

	if err := saveToken(*token); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving token: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Token saved successfully.")
}

func cmdSearch(args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	query := fs.String("query", "", "Search query for SIEM")
	limit := fs.Int("limit", 100, "Maximum number of results")
	fs.Parse(args)

	if *query == "" {
		fmt.Fprintln(os.Stderr, "Error: missing --query")
		os.Exit(1)
	}

	cli := &Client{
		BaseURL: "http://localhost:8080/api/v1",
		Token:   loadToken(),
	}

	if err := cli.Search(*query, *limit); err != nil {
		fmt.Fprintf(os.Stderr, "Search failed: %v\n", err)
		os.Exit(1)
	}
}

func cmdStream(args []string) {
	fs := flag.NewFlagSet("stream", flag.ExitOnError)
	fs.Parse(args)

	cli := &Client{
		BaseURL: "http://localhost:8080/api/v1",
		Token:   loadToken(),
	}

	if err := cli.StreamEvents(); err != nil {
		fmt.Fprintf(os.Stderr, "Stream failed: %v\n", err)
		os.Exit(1)
	}
}
