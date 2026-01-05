package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/paw-chain/paw/explorer/indexer/internal/indexer"
)

// handleGetIndexingStatus handles GET /api/v1/indexing/status requests
func (s *Server) handleGetIndexingStatus(c *gin.Context) {
	status, err := s.indexer.GetIndexingStatus()
	if err != nil {
		s.log.Error("Failed to get indexing status", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch indexing status",
		})
		return
	}

	// Get additional statistics from database
	stats, err := s.db.GetIndexingStatistics()
	if err != nil {
		s.log.Warn("Failed to get indexing statistics", "error", err)
		// Continue without detailed stats
	}

	response := gin.H{
		"status":               status.Status,
		"is_active":            status.IsActive,
		"last_indexed_height":  status.LastIndexedHeight,
		"current_chain_height": status.CurrentChainHeight,
		"progress_percent":     status.ProgressPercent,
	}

	if stats != nil {
		response["total_blocks_indexed"] = stats.TotalBlocksIndexed
		response["failed_blocks_count"] = stats.FailedBlocksCount
		response["unresolved_failed_blocks"] = stats.UnresolvedFailedBlocks
		if stats.AvgBlocksPerSecond != nil {
			response["avg_blocks_per_second"] = *stats.AvgBlocksPerSecond
		}
		if stats.EstimatedCompletionTime != nil {
			response["estimated_completion"] = stats.EstimatedCompletionTime
		}
	}

	c.JSON(http.StatusOK, response)
}

// handleGetIndexingProgress handles GET /api/v1/indexing/progress requests
func (s *Server) handleGetIndexingProgress(c *gin.Context) {
	progress, err := s.db.GetIndexingProgress()
	if err != nil {
		s.log.Error("Failed to get indexing progress", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch indexing progress",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"progress": progress,
	})
}

// handleGetFailedBlocks handles GET /api/v1/indexing/failed-blocks requests
func (s *Server) handleGetFailedBlocks(c *gin.Context) {
	maxRetries := parseQueryInt(c, "max_retries", 5)

	failedBlocks, err := s.db.GetFailedBlocks(maxRetries)
	if err != nil {
		s.log.Error("Failed to get failed blocks", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch failed blocks",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"failed_blocks": failedBlocks,
		"count":         len(failedBlocks),
	})
}

// handleGetIndexingStatistics handles GET /api/v1/indexing/statistics requests
func (s *Server) handleGetIndexingStatistics(c *gin.Context) {
	stats, err := s.db.GetIndexingStatistics()
	if err != nil {
		s.log.Error("Failed to get indexing statistics", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch indexing statistics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"statistics": stats,
	})
}

// SetupIndexingRoutes adds indexing-related routes to the API
func (s *Server) SetupIndexingRoutes(v1 *gin.RouterGroup, indexer *indexer.Indexer) {
	s.indexer = indexer

	// Indexing routes
	indexing := v1.Group("/indexing")
	{
		indexing.GET("/status", s.handleGetIndexingStatus)
		indexing.GET("/progress", s.handleGetIndexingProgress)
		indexing.GET("/failed-blocks", s.handleGetFailedBlocks)
		indexing.GET("/statistics", s.handleGetIndexingStatistics)
		indexing.GET("/gaps", s.handleGetGaps)
		indexing.POST("/fill-gaps", s.handleFillGaps)
	}
}

// handleGetGaps handles GET /api/v1/indexing/gaps requests
func (s *Server) handleGetGaps(c *gin.Context) {
	status, err := s.indexer.GetGapStatus()
	if err != nil {
		s.log.Error("Failed to get gap status", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch gap status",
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// handleFillGaps handles POST /api/v1/indexing/fill-gaps requests
func (s *Server) handleFillGaps(c *gin.Context) {
	if err := s.indexer.StartGapFilling(); err != nil {
		s.log.Error("Failed to start gap filling", "error", err)
		c.JSON(http.StatusConflict, gin.H{
			"error":   "failed to start gap filling",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Gap filling started in background",
		"status":  "started",
	})
}
