package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v3"

	duckduckgo "github.com/kadirgun/duckduck-go"
)

func main() {
	cmd := &cli.Command{
		Name:  "duckduckgo-cli",
		Usage: "Search DuckDuckGo from the command line",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "query",
				Aliases:  []string{"q"},
				Usage:    "search query",
				Required: true,
			},
			&cli.IntFlag{
				Name:    "count",
				Aliases: []string{"n"},
				Value:   10,
				Usage:   "number of results (max 50)",
				Validator: func(v int) error {
					if v < 1 {
						return fmt.Errorf("count must be at least 1")
					}
					if v > 50 {
						return fmt.Errorf("count must be at most 50")
					}
					return nil
				},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			query := cmd.String("query")
			count := cmd.Int("count")

			client := duckduckgo.New()
			results, err := client.Search(query, count)
			if err != nil {
				return fmt.Errorf("search failed: %w", err)
			}

			output := map[string]interface{}{
				"query":   query,
				"count":   len(results),
				"results": results,
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(output); err != nil {
				return fmt.Errorf("failed to encode output: %w", err)
			}

			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
