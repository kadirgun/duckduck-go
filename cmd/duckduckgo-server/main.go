package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	duckduckgo "github.com/kadirgun/duckduck-go"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func main() {
	// Load .env file if exists
	if err := godotenv.Load(); err != nil {
		slog.Debug(".env file not found, using environment variables")
	}

	apiKey := os.Getenv("API_KEY")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	e := echo.New()

	// Middleware
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS("*"))

	// API key middleware (only if API_KEY is set)
	if apiKey != "" {
		e.Use(middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
			Validator: func(c *echo.Context, key string, _ middleware.ExtractorSource) (bool, error) {
				return key == apiKey, nil
			},
		}))
		slog.Info("API key authentication enabled")
	}

	// Health check
	e.GET("/health", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	// Search endpoint
	e.POST("/api/search", searchHandler)

	addr := fmt.Sprintf(":%s", port)
	slog.Info("starting server", "addr", addr)
	if err := e.Start(addr); err != nil {
		slog.Error("server stopped", "error", err)
	}
}

type searchRequest struct {
	Query string `json:"q"`
	Count int    `json:"count"`
}

func searchHandler(c *echo.Context) error {
	var req searchRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	if req.Query == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "field 'q' is required",
		})
	}

	count := req.Count
	if count < 1 {
		count = 10
	} else if count > 50 {
		count = 50
	}

	client := duckduckgo.New()
	results, err := client.Search(req.Query, count)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("search failed: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"query":   req.Query,
		"count":   len(results),
		"results": results,
	})
}
