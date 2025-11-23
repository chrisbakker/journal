package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/chrisbakker/journal/api"
	"github.com/chrisbakker/journal/config"
	db "github.com/chrisbakker/journal/generated"
	"github.com/chrisbakker/journal/ollama"
	"github.com/chrisbakker/journal/vectorservice"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AppResources holds reloadable application resources
type AppResources struct {
	mu           sync.RWMutex
	config       *config.Config
	dbpool       *pgxpool.Pool
	queries      *db.Queries
	ollamaClient *ollama.Client
	vectorSvc    *vectorservice.VectorService
	ctx          context.Context
	cancel       context.CancelFunc
}

func main() {
	runServer()
}

// Reload reloads configuration and reconnects to all resources
func (app *AppResources) Reload() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	log.Println("🔄 Reloading configuration...")

	// Stop existing vector service
	if app.vectorSvc != nil {
		app.vectorSvc.Stop()
	}

	// Cancel existing context
	if app.cancel != nil {
		app.cancel()
	}

	// Close existing database connection
	if app.dbpool != nil {
		app.dbpool.Close()
	}

	// Reload configuration
	newCfg := config.Load()
	validationResult := newCfg.Validate()
	if !validationResult.Valid {
		log.Println("❌ Configuration validation failed:")
		log.Println(validationResult.FormatErrorsForDisplay())
		return fmt.Errorf("invalid configuration")
	}

	// Create new context
	ctx, cancel := context.WithCancel(context.Background())

	// Reconnect to database
	dbpool, err := pgxpool.New(ctx, newCfg.Database.URL)
	if err != nil {
		cancel()
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := dbpool.Ping(ctx); err != nil {
		dbpool.Close()
		cancel()
		return fmt.Errorf("failed to ping database: %w", err)
	}
	log.Println("✅ Reconnected to database successfully")

	// Create new queries
	queries := db.New(dbpool)

	// Reinitialize Ollama client
	ollamaClient := ollama.NewClient(newCfg.LLM.OllamaBaseURL)
	log.Printf("✅ Reinitialized Ollama client at %s", newCfg.LLM.OllamaBaseURL)

	// Reinitialize vector service
	vectorSvc := vectorservice.New(
		queries,
		ollamaClient,
		newCfg.LLM.UpdateInterval,
		10,
	)

	// Start background vector update service if enabled
	if newCfg.LLM.EnableVectorSearch {
		vectorSvc.Start(ctx)
		log.Println("✅ Restarted background vector update service")
	}

	// Update all resources
	app.config = newCfg
	app.dbpool = dbpool
	app.queries = queries
	app.ollamaClient = ollamaClient
	app.vectorSvc = vectorSvc
	app.ctx = ctx
	app.cancel = cancel

	log.Println("✅ Configuration reloaded successfully!")
	return nil
}

// Get methods for safe concurrent access
func (app *AppResources) getQueries() *db.Queries {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.queries
}

func (app *AppResources) getConfig() *config.Config {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.config
}

func (app *AppResources) getVectorService() *vectorservice.VectorService {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.vectorSvc
}

func (app *AppResources) getOllamaClient() *ollama.Client {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.ollamaClient
}

func runServer() {
	// Check if config file exists
	envExists, err := config.CheckEnvFile()
	if err != nil {
		log.Printf("⚠️  Error checking config file: %v\n", err)
	}

	// Show config file location
	configPath, _ := config.GetConfigPath()
	if configPath != "" {
		log.Printf("📁 Config file: %s", configPath)
	}

	// Load configuration (with defaults if file doesn't exist)
	cfg := config.Load()

	// Initialize application resources
	app := &AppResources{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// If config doesn't exist or is invalid, start with minimal setup
	// The frontend will detect this via API 503 responses and show config wizard
	validationResult := cfg.Validate()
	if !envExists || !validationResult.Valid {
		if !envExists {
			log.Println("⚠️  No configuration file found")
		} else {
			log.Println("⚠️  Configuration validation failed:")
			log.Println(validationResult.FormatErrorsForDisplay())
		}
		log.Println("Starting server. Configure via web interface.")
	}

	// Try to connect to database (may fail if not configured)
	var dbpool *pgxpool.Pool
	var queries *db.Queries
	var ollamaClient *ollama.Client
	var vectorSvc *vectorservice.VectorService

	if validationResult.Valid {
		dbpool, err = pgxpool.New(ctx, cfg.Database.URL)
		if err != nil {
			log.Printf("⚠️  Unable to connect to database: %v\n", err)
			log.Println("Please configure database settings via web interface.")
		} else {
			// Test connection
			if err := dbpool.Ping(ctx); err != nil {
				log.Printf("⚠️  Unable to ping database: %v\n", err)
				dbpool.Close()
				dbpool = nil
			} else {
				log.Println("Connected to database successfully")

				// Create queries
				queries = db.New(dbpool)

				// Initialize Ollama client
				ollamaClient = ollama.NewClient(cfg.LLM.OllamaBaseURL)
				log.Printf("Initialized Ollama client at %s", cfg.LLM.OllamaBaseURL)

				// Initialize vector service
				vectorSvc = vectorservice.New(
					queries,
					ollamaClient,
					cfg.LLM.UpdateInterval,
					10, // batch size
				)

				// Start background vector update service if enabled
				if cfg.LLM.EnableVectorSearch {
					vectorSvc.Start(ctx)
					log.Println("Started background vector update service")
				}
			}
		}
	}

	// Store resources in app
	app.config = cfg
	app.dbpool = dbpool
	app.queries = queries
	app.ollamaClient = ollamaClient
	app.vectorSvc = vectorSvc
	app.ctx = ctx
	app.cancel = cancel

	// Set up Gin
	if cfg.Server.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORS.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: cfg.CORS.AllowCredentials,
		MaxAge:           cfg.CORS.MaxAge,
	}))

	// API routes - use closures to always get current resources from app
	apiGroup := router.Group("/api")
	{
		// Configuration - triggers internal reload
		apiGroup.POST("/config", func(c *gin.Context) {
			handler := &api.Handler{}
			handler.SaveConfig(c, app)
		})

		// Helper to check if resources are available
		requireResources := func(c *gin.Context) bool {
			if app.getQueries() == nil {
				c.JSON(503, gin.H{
					"error":   "Configuration required",
					"message": "The application requires configuration. Please configure via settings.",
				})
				return false
			}
			return true
		}

		// Entries - dynamically get resources
		apiGroup.GET("/days/:date/entries", func(c *gin.Context) {
			if !requireResources(c) {
				return
			}
			handler := api.NewHandler(app.getQueries(), app.getConfig().App.DefaultTimezone, app.getVectorService(), app.getOllamaClient())
			handler.ListEntriesForDay(c)
		})
		apiGroup.POST("/entries", func(c *gin.Context) {
			if !requireResources(c) {
				return
			}
			handler := api.NewHandler(app.getQueries(), app.getConfig().App.DefaultTimezone, app.getVectorService(), app.getOllamaClient())
			handler.CreateEntry(c)
		})
		apiGroup.PATCH("/entries/:id", func(c *gin.Context) {
			if !requireResources(c) {
				return
			}
			handler := api.NewHandler(app.getQueries(), app.getConfig().App.DefaultTimezone, app.getVectorService(), app.getOllamaClient())
			handler.UpdateEntry(c)
		})
		apiGroup.DELETE("/entries/:id", func(c *gin.Context) {
			if !requireResources(c) {
				return
			}
			handler := api.NewHandler(app.getQueries(), app.getConfig().App.DefaultTimezone, app.getVectorService(), app.getOllamaClient())
			handler.DeleteEntry(c)
		})

		// Search
		apiGroup.GET("/search", func(c *gin.Context) {
			if !requireResources(c) {
				return
			}
			handler := api.NewHandler(app.getQueries(), app.getConfig().App.DefaultTimezone, app.getVectorService(), app.getOllamaClient())
			handler.SearchEntries(c)
		})

		// Chat (Phase 3 - RAG)
		apiGroup.POST("/chat", func(c *gin.Context) {
			if !requireResources(c) {
				return
			}
			handler := api.NewHandler(app.getQueries(), app.getConfig().App.DefaultTimezone, app.getVectorService(), app.getOllamaClient())
			handler.Chat(c)
		})

		// Attachments
		apiGroup.POST("/entries/:id/attachments", func(c *gin.Context) {
			if !requireResources(c) {
				return
			}
			handler := api.NewHandler(app.getQueries(), app.getConfig().App.DefaultTimezone, app.getVectorService(), app.getOllamaClient())
			handler.UploadAttachment(c)
		})
		apiGroup.GET("/attachments/:id", func(c *gin.Context) {
			if !requireResources(c) {
				return
			}
			handler := api.NewHandler(app.getQueries(), app.getConfig().App.DefaultTimezone, app.getVectorService(), app.getOllamaClient())
			handler.GetAttachment(c)
		})
		apiGroup.DELETE("/attachments/:id", func(c *gin.Context) {
			if !requireResources(c) {
				return
			}
			handler := api.NewHandler(app.getQueries(), app.getConfig().App.DefaultTimezone, app.getVectorService(), app.getOllamaClient())
			handler.DeleteAttachment(c)
		})

		// Calendar
		apiGroup.GET("/months/:yearmonth/entry-days", func(c *gin.Context) {
			if !requireResources(c) {
				return
			}
			handler := api.NewHandler(app.getQueries(), app.getConfig().App.DefaultTimezone, app.getVectorService(), app.getOllamaClient())
			handler.GetDaysWithEntries(c)
		})

		// Export
		apiGroup.GET("/export", func(c *gin.Context) {
			if !requireResources(c) {
				return
			}
			handler := api.NewHandler(app.getQueries(), app.getConfig().App.DefaultTimezone, app.getVectorService(), app.getOllamaClient())
			handler.ExportEntries(c)
		})
	}

	// Serve SPA
	if cfg.SPA.Mode == "embed" {
		// Production: use embedded files (not yet implemented)
		// For now, fallback to filesystem mode
		log.Println("Warning: embed mode not yet implemented, using filesystem mode")
		router.NoRoute(func(c *gin.Context) {
			c.File(cfg.SPA.Dir + "/index.html")
		})
		router.Static("/assets", cfg.SPA.Dir+"/assets")
	} else {
		// Development: serve from filesystem
		router.NoRoute(func(c *gin.Context) {
			c.File(cfg.SPA.Dir + "/index.html")
		})
		router.Static("/assets", cfg.SPA.Dir+"/assets")
	}

	// Start server
	addr := ":" + cfg.Server.Port
	log.Printf("Starting server on %s (SPA mode: %s)\n", addr, cfg.SPA.Mode)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v\n", err)
	}
}
