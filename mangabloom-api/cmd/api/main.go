package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"mangabloomapi/internal/db"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
)

// Database setup (replace with your own connection details)
func setupDB() (*sql.DB, error) {
	if err := godotenv.Load(".env"); err != nil {
		// This is normal in Docker/Production! Do not panic.
		fmt.Println("No physical .env file found; reading from system environment variables instead.")
	}

	db_url_string := os.Getenv("DB_URL")
	if db_url_string == "" {
		return nil, fmt.Errorf("db_url not found")
	}

	db_url, err := url.Parse(db_url_string)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database url: %w", err)
	}
	fmt.Println("db url is ", db_url)

	db, err := sql.Open("postgres", db_url.String())
	if err != nil {
		return nil, fmt.Errorf("failed to open database pool: %w", err)
	}

	// 🔴 ADD THIS CHECK TO VERIFY CONNECTION
	if err := db.Ping(); err != nil {
		db.Close() // Clean up the opened pool resource if ping fails
		return nil, fmt.Errorf("database is unreachable: %w", err)
	}

	fmt.Println("🎉 Database successfully connected and reachable!")
	return db, nil
}

func main() {
	// Initialize the database
	db_conf, err := setupDB()
	if err != nil {
		panic("Failed to connect to the database: " + err.Error())
	}
	defer db_conf.Close()

	queries := db.New(db_conf)

	// Initialize Echo
	e := echo.New()

	// Middleware
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	// cors middleware
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"https://manga-bloom.vercel.app", "http://localhost:5173"}, // Allow all origins
		AllowMethods: []string{
			echo.GET,
			echo.POST,
			echo.PUT,
			echo.DELETE,
			echo.OPTIONS,
		},
		AllowHeaders: []string{
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization, // Useful if you use tokens
			"ngrok-skip-browser-warning",
			"Access-Control-Allow-Origin", // Include this explicitly
		},
	}))

	e.GET("/covers/*", func(c echo.Context) error {
		file := c.Param("*")
		filePath := "./covers/" + file

		// Check if the file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// Return a 404 error without caching
			return echo.NewHTTPError(http.StatusNotFound, "File not found")
		}

		// Set Cache-Control headers only for successful responses
		c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")

		// Serve the file
		return c.File(filePath)
	})

	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"health": "OK",
		})
	})

	e.GET("/mangas", func(c echo.Context) error {
		return handleGetPaginatedMangas(c, queries)
	})

	e.GET("/manga/:id", func(c echo.Context) error {
		return handleGetMangaByID(c, queries)
	})

	// Start the server
	e.Logger.Fatal(e.Start(":3000"))
}
