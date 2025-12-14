/**
 * Global Error Handler Middleware
 */

import { logger } from '../utils/logger.js';

/**
 * Global error handler
 * Catches all errors and returns consistent error responses
 */
export const errorHandler = (err, req, res, next) => {
    // Log error
    logger.error('Unhandled error:', {
        error: err.message,
        stack: err.stack,
        path: req.path,
        method: req.method,
        ip: req.ip,
        userAgent: req.get('user-agent')
    });

    // Determine status code
    const statusCode = err.statusCode || err.status || 500;

    // Don't leak error details in production
    const errorMessage = process.env.NODE_ENV === 'production' && statusCode === 500
        ? 'An unexpected error occurred'
        : err.message;

    // Send error response
    res.status(statusCode).json({
        error: err.name || 'Error',
        message: errorMessage,
        ...(process.env.NODE_ENV === 'development' && { stack: err.stack }),
        timestamp: new Date().toISOString(),
        path: req.path
    });
};

/**
 * 404 Not Found handler
 */
export const notFoundHandler = (req, res) => {
    res.status(404).json({
        error: 'Not Found',
        message: `Cannot ${req.method} ${req.path}`,
        path: req.path,
        timestamp: new Date().toISOString()
    });
};
