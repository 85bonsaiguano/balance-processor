package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	transactionUseCase "github.com/amirhossein-jamali/balance-processor/internal/domain/usecase/transaction"
	userUseCase "github.com/amirhossein-jamali/balance-processor/internal/domain/usecase/user"

	"github.com/amirhossein-jamali/balance-processor/internal/infrastructure/adapter/api/handler"
	"github.com/amirhossein-jamali/balance-processor/internal/infrastructure/adapter/api/routes"
	"github.com/amirhossein-jamali/balance-processor/internal/infrastructure/adapter/database"
	"github.com/amirhossein-jamali/balance-processor/internal/infrastructure/adapter/database/migration"
	"github.com/amirhossein-jamali/balance-processor/internal/infrastructure/adapter/logger"
	"github.com/amirhossein-jamali/balance-processor/internal/infrastructure/adapter/repository"
	timeProvider "github.com/amirhossein-jamali/balance-processor/internal/infrastructure/adapter/time"
	"github.com/amirhossein-jamali/balance-processor/internal/infrastructure/config"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate essential configuration
	if err := validateConfig(cfg); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Set Gin mode based on environment
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create logger
	appLogger := logger.NewZapLogger(cfg.Environment == "production")

	// Setup database configuration
	dbConfig := &database.Config{
		Driver:          "postgres",
		Host:            cfg.Database.Host,
		Port:            database.ParsePort(cfg.Database.Port),
		Username:        cfg.Database.Username,
		Password:        cfg.Database.Password,
		Database:        cfg.Database.Database,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
		QueryTimeout:    cfg.Database.QueryTimeout,
		LogLevel:        cfg.Logger.Level,
		RetryAttempts:   3,
		RetryDelay:      5,
	}

	// Initialize time provider
	tp := timeProvider.NewRealTimeProvider()

	// Connect to the database
	dbManager := database.NewManager(dbConfig, appLogger, tp)
	var connectErr error
	_, connectErr = dbManager.Connect() // We don't need to store the DB as it's already stored in dbManager
	err = connectErr
	if err != nil {
		appLogger.Error("Failed to connect to database", map[string]any{
			"error": err.Error(),
		})
		os.Exit(1)
	}
	defer dbManager.Close()

	// Initialize repositories
	userRepo := repository.NewUserRepository(dbManager.DB(), tp, appLogger)
	userLockRepo := repository.NewUserLockRepository(dbManager.DB(), tp, appLogger)
	// transactionRepo is used inside the UnitOfWork
	_ = repository.NewTransactionRepository(dbManager.DB(), appLogger)

	// Unit of work (transaction manager)
	uow := database.NewUnitOfWork(dbManager.DB(), appLogger, tp)

	// Run migrations
	migrationMgr := migration.NewMigrationManagerWithTimeProvider(dbManager.DB(), appLogger, tp)
	err = migrationMgr.MigrateAll()
	if err != nil {
		appLogger.Error("Failed to run migrations", map[string]any{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	// Initialize use cases
	userUseCaseImpl := userUseCase.NewUserUseCase(userRepo, tp, appLogger)

	// Lock timeout for transaction processing
	lockTimeout := time.Duration(cfg.Transaction.LockTimeoutMs) * time.Millisecond

	transactionUseCaseImpl := transactionUseCase.NewTransactionService(
		uow,
		userUseCaseImpl,
		userLockRepo,
		tp,
		appLogger,
		lockTimeout,
	)

	// Create default users
	err = migration.CreateDefaultUsers(context.Background(), userUseCaseImpl)
	if err != nil {
		appLogger.Error("Failed to create default users", map[string]any{
			"error": err.Error(),
		})
	}

	// Initialize API handlers
	userHandler := handler.NewUserHandler(userUseCaseImpl, appLogger)
	transactionHandler := handler.NewTransactionHandler(transactionUseCaseImpl, userUseCaseImpl, appLogger)

	// Initialize Gin router
	router := gin.New()

	// Setup middlewares
	routes.SetupMiddlewares(router, appLogger)

	// Setup routes
	routes.SetupRoutes(router, transactionHandler, userHandler)

	// Create HTTP server with configurable timeout values
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:           router,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
	}

	// Start the server in a goroutine
	go func() {
		appLogger.Info("Starting server", map[string]any{
			"port": cfg.Server.Port,
			"env":  cfg.Environment,
		})

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("Failed to start server", map[string]any{
				"error": err.Error(),
			})
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...", nil)

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	// Shutdown TransactionManager cleanly
	if txService, ok := transactionUseCaseImpl.(*transactionUseCase.Service); ok {
		if txManager := txService.GetManager(); txManager != nil {
			appLogger.Info("Shutting down transaction manager...", nil)
			txManager.Shutdown()
		}
	}

	// Shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		appLogger.Error("Server forced to shutdown", map[string]any{
			"error": err.Error(),
		})
	}

	appLogger.Info("Server exited gracefully", nil)
}

// validateConfig ensures all required configuration values are present
func validateConfig(cfg *config.Config) error {
	var missingConfigs []string

	// Validate server configuration
	if cfg.Server.Port == 0 {
		missingConfigs = append(missingConfigs, "server.port")
	}

	if cfg.Server.ReadTimeout == 0 {
		missingConfigs = append(missingConfigs, "server.readTimeout")
	}

	if cfg.Server.WriteTimeout == 0 {
		missingConfigs = append(missingConfigs, "server.writeTimeout")
	}

	if cfg.Server.ShutdownTimeout == 0 {
		missingConfigs = append(missingConfigs, "server.shutdownTimeout")
	}

	// Validate database configuration
	if cfg.Database.Host == "" {
		// In production, check if environment variable exists
		if cfg.Environment == config.Production && os.Getenv("BP_DB_HOST") == "" {
			missingConfigs = append(missingConfigs, "database.host (or BP_DB_HOST environment variable)")
		} else if cfg.Environment != config.Production {
			missingConfigs = append(missingConfigs, "database.host")
		}
	}

	if cfg.Database.Port == "" {
		// In production, check if environment variable exists
		if cfg.Environment == config.Production && os.Getenv("BP_DB_PORT") == "" {
			missingConfigs = append(missingConfigs, "database.port (or BP_DB_PORT environment variable)")
		} else if cfg.Environment != config.Production {
			missingConfigs = append(missingConfigs, "database.port")
		}
	}

	if cfg.Database.Username == "" {
		// In production, check if environment variable exists
		if cfg.Environment == config.Production && os.Getenv("BP_DB_USERNAME") == "" {
			missingConfigs = append(missingConfigs, "database.username (or BP_DB_USERNAME environment variable)")
		} else if cfg.Environment != config.Production {
			missingConfigs = append(missingConfigs, "database.username")
		}
	}

	if cfg.Database.Password == "" {
		// In production, check if environment variable exists
		if cfg.Environment == config.Production && os.Getenv("BP_DB_PASSWORD") == "" {
			missingConfigs = append(missingConfigs, "database.password (or BP_DB_PASSWORD environment variable)")
		} else if cfg.Environment != config.Production {
			missingConfigs = append(missingConfigs, "database.password")
		}
	}

	if cfg.Database.Database == "" {
		// In production, check if environment variable exists
		if cfg.Environment == config.Production && os.Getenv("BP_DB_NAME") == "" {
			missingConfigs = append(missingConfigs, "database.database (or BP_DB_NAME environment variable)")
		} else if cfg.Environment != config.Production {
			missingConfigs = append(missingConfigs, "database.database")
		}
	}

	if cfg.Database.QueryTimeout == 0 {
		missingConfigs = append(missingConfigs, "database.queryTimeout")
	}

	// Validate transaction configuration
	if cfg.Transaction.ConcurrencyLevel == 0 {
		missingConfigs = append(missingConfigs, "transaction.concurrencyLevel")
	}

	if cfg.Transaction.LockTimeoutMs == 0 {
		missingConfigs = append(missingConfigs, "transaction.lockTimeoutMs")
	}

	if cfg.Transaction.MaxRetries == 0 {
		missingConfigs = append(missingConfigs, "transaction.maxRetries")
	}

	// Environment should be set with a valid value
	if cfg.Environment == "" {
		missingConfigs = append(missingConfigs, "environment")
	} else if cfg.Environment != config.Development &&
		cfg.Environment != config.Production &&
		cfg.Environment != config.Test {
		return fmt.Errorf("invalid environment value: %s, must be one of: %s, %s, or %s",
			cfg.Environment, config.Development, config.Production, config.Test)
	}

	// Logger configuration
	if cfg.Logger.Level == "" {
		missingConfigs = append(missingConfigs, "logger.level")
	}

	// Return error with list of missing configurations
	if len(missingConfigs) > 0 {
		return fmt.Errorf("missing required configurations: %v", missingConfigs)
	}

	// If we're in production, do additional validation for sensitive settings
	if cfg.Environment == config.Production {
		var warnings []string

		// Check database security settings
		if strings.ToLower(cfg.Database.SSLMode) != "require" && strings.ToLower(cfg.Database.SSLMode) != "verify-ca" && strings.ToLower(cfg.Database.SSLMode) != "verify-full" {
			warnings = append(warnings, "database.sslMode should be set to 'require', 'verify-ca', or 'verify-full' in production")
		}

		// Check timeout settings
		if cfg.Server.ReadTimeout < 5*time.Second {
			warnings = append(warnings, "server.readTimeout is too low for production")
		}

		if cfg.Server.WriteTimeout < 5*time.Second {
			warnings = append(warnings, "server.writeTimeout is too low for production")
		}

		if len(warnings) > 0 {
			log.Printf("Warning: potential security issues in production configuration: %v", warnings)
		}
	}

	return nil
}
