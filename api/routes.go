package api

// registerRoutes registers all API routes
func (s *Server) registerRoutes() {
	// API version 1
	api := s.router.Group("/api")
	{
		// Authentication routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/register", s.handleRegister)
			auth.POST("/login", s.handleLogin)
		}

		// Trading/Order routes
		orders := api.Group("/orders")
		{
			orders.GET("/book", s.handleGetOrderBook)
			orders.GET("/recent", s.handleGetRecentTrades)

			// Protected routes
			ordersProtected := orders.Group("")
			ordersProtected.Use(s.AuthMiddleware())
			{
				ordersProtected.POST("/create", s.handleCreateOrder)
				ordersProtected.GET("/my-orders", s.handleGetMyOrders)
				ordersProtected.DELETE("/:order_id", s.handleCancelOrder)
			}
		}

		// Trades routes (public)
		trades := api.Group("/trades")
		{
			trades.GET("/recent", s.handleGetRecentTrades)
			trades.GET("/history", s.handleGetTradeHistory)
		}

		// Wallet routes (protected)
		wallet := api.Group("/wallet")
		wallet.Use(s.AuthMiddleware())
		{
			wallet.GET("/balance", s.handleGetBalance)
			wallet.GET("/address", s.handleGetAddress)
			wallet.POST("/send", s.handleSendTokens)
			wallet.GET("/transactions", s.handleGetTransactions)
		}

		// Light client routes (public)
		lightClient := api.Group("/light-client")
		{
			lightClient.GET("/headers", s.handleGetHeaders)
			lightClient.GET("/headers/:height", s.handleGetHeaderByHeight)
			lightClient.GET("/checkpoint", s.handleGetCheckpoint)
			lightClient.GET("/tx-proof/:txid", s.handleGetTxProof)
			lightClient.POST("/verify-proof", s.handleVerifyProof)
		}

		// Atomic swap routes
		atomicSwap := api.Group("/atomic-swap")
		atomicSwap.Use(s.AuthMiddleware())
		{
			atomicSwap.POST("/prepare", s.handlePrepareSwap)
			atomicSwap.POST("/commit", s.handleCommitSwap)
			atomicSwap.POST("/refund", s.handleRefundSwap)
			atomicSwap.GET("/status/:swap_id", s.handleGetSwapStatus)
			atomicSwap.GET("/my-swaps", s.handleGetMySwaps)
		}

		// DEX Pool routes (public read, protected write)
		pools := api.Group("/pools")
		{
			pools.GET("", s.handleGetPools)
			pools.GET("/:pool_id", s.handleGetPool)
			pools.GET("/:pool_id/liquidity", s.handleGetPoolLiquidity)

			poolsProtected := pools.Group("")
			poolsProtected.Use(s.AuthMiddleware())
			{
				poolsProtected.POST("/add-liquidity", s.handleAddLiquidity)
				poolsProtected.POST("/remove-liquidity", s.handleRemoveLiquidity)
			}
		}

		// Market data routes (public)
		market := api.Group("/market")
		{
			market.GET("/price", s.handleGetPrice)
			market.GET("/stats", s.handleGetMarketStats)
			market.GET("/24h", s.handleGet24HStats)
		}
	}

	// WebSocket endpoint
	s.router.GET("/ws", s.handleWebSocket)

	// Static file serving (for frontend if needed)
	s.router.Static("/static", "./static")
}
