package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Product represents a product in the database
type Product struct {
	gorm.Model
	Code  string
	Price uint
}

func main() {
	// Initialize SQLite database
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// Migrate the schema
	db.AutoMigrate(&Product{})

	// Create some sample products
	db.Create(&Product{Code: "D42", Price: 100})
	db.Create(&Product{Code: "P99", Price: 200})

	// Create a new MCP server
	s := server.NewMCPServer(
		"Demo",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// Add tool
	tool := mcp.NewTool("hello_world",
		mcp.WithDescription("Say hello to someone"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the person to greet"),
		),
	)

	// Add tool handler
	s.AddTool(tool, helloHandler)

	// Add resource for listing products
	resource := mcp.NewResource("products://list", "Product List",
		mcp.WithResourceDescription("Lists all available products"),
	)

	s.AddResource(resource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		var products []Product
		db.Find(&products)

		jsonData, err := json.MarshalIndent(products, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal products to JSON: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "products://list",
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		}, nil
	})

	calculatorTool := mcp.NewTool("calculate",
		mcp.WithDescription("Perform basic arithmetic operations"),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("The operation to perform (add, subtract, multiply, divide)"),
			mcp.Enum("add", "subtract", "multiply", "divide"),
		),
		mcp.WithNumber("x",
			mcp.Required(),
			mcp.Description("First number"),
		),
		mcp.WithNumber("y",
			mcp.Required(),
			mcp.Description("Second number"),
		),
	)

	// Add the calculator handler
	s.AddTool(calculatorTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Using helper functions for type-safe argument access
		op, err := request.RequireString("operation")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		x, err := request.RequireFloat("x")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		y, err := request.RequireFloat("y")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var result float64
		switch op {
		case "add":
			result = x + y
		case "subtract":
			result = x - y
		case "multiply":
			result = x * y
		case "divide":
			if y == 0 {
				return mcp.NewToolResultError("cannot divide by zero"), nil
			}
			result = x / y
		}

		return mcp.NewToolResultText(fmt.Sprintf("%.2f", result)), nil
	})

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}

}

func helloHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Hello, %s!", name)), nil
}
