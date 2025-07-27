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

// DB is a global database instance
var DB *gorm.DB

// initializeDatabase initializes the SQLite database and performs migrations
func initializeDatabase() error {
	var err error
	DB, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	// Migrate the schema
	if err := DB.AutoMigrate(&Product{}); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}

// seedDatabase creates sample products if the database is empty
func seedDatabase() error {
	var count int64
	DB.Model(&Product{}).Count(&count)

	if count == 0 {
		// Create some sample products
		products := []Product{
			{Code: "D42", Price: 100},
			{Code: "P99", Price: 200},
		}

		if err := DB.CreateInBatches(products, len(products)).Error; err != nil {
			return fmt.Errorf("failed to seed database: %w", err)
		}

		log.Println("Database seeded with sample products")
	}

	return nil
}

// helloHandler handles the hello_world tool request
func helloHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Hello, %s!", name)), nil
}

// listProductsHandler handles the products resource request
func listProductsHandler(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	var products []Product
	if err := DB.Find(&products).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve products: %w", err)
	}

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
}

// calculateHandler handles the calculate tool request
func calculateHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	default:
		return mcp.NewToolResultError(fmt.Sprintf("unsupported operation: %s", op)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%.2f", result)), nil
}

// setupServer creates and configures the MCP server with tools and resources
func setupServer() *server.MCPServer {
	// Create a new MCP server
	s := server.NewMCPServer(
		"Demo",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// Add hello_world tool
	helloTool := mcp.NewTool("hello_world",
		mcp.WithDescription("Say hello to someone"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the person to greet"),
		),
	)
	s.AddTool(helloTool, helloHandler)

	// Add products resource for listing products
	productsResource := mcp.NewResource("products://list", "Product List",
		mcp.WithResourceDescription("Lists all available products"),
	)
	s.AddResource(productsResource, listProductsHandler)

	// Add calculator tool
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
	s.AddTool(calculatorTool, calculateHandler)

	return s
}

func main() {
	// Initialize database
	if err := initializeDatabase(); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}

	// Seed database with sample data
	if err := seedDatabase(); err != nil {
		log.Printf("Warning: Database seeding failed: %v", err)
	}

	// Setup and start the MCP server
	s := setupServer()

	log.Println("Starting MCP server...")
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
