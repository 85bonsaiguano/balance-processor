package routes

import (
	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	"github.com/amirhossein-jamali/balance-processor/internal/infrastructure/adapter/api/handler"
	"github.com/amirhossein-jamali/balance-processor/internal/infrastructure/adapter/api/middleware"
	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all the routes for the API
func SetupRoutes(
	router *gin.Engine,
	transactionHandler *handler.TransactionHandler,
	userHandler *handler.UserHandler,
) {
	// User routes
	userRoutes := router.Group("/user")
	{
		// GET /user/:userId/balance
		userRoutes.GET("/:userId/balance", userHandler.GetBalance)

		// POST /user/:userId/transaction
		userRoutes.POST("/:userId/transaction", transactionHandler.ProcessTransaction)
	}
}

// SetupMiddlewares configures global middlewares for the API
func SetupMiddlewares(router *gin.Engine, logger coreport.Logger) {
	// Apply middlewares in the correct order
	router.Use(middleware.ErrorHandler(logger))
	router.Use(middleware.Logger(logger))
	router.Use(middleware.CORS())
}
