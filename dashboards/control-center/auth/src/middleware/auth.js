/**
 * JWT Authentication Middleware
 */

import jwt from 'jsonwebtoken';
import { logger } from '../utils/logger.js';

const JWT_SECRET = process.env.JWT_SECRET || 'paw-dashboard-secret-change-in-production';

/**
 * Verify JWT token from Authorization header
 */
export const verifyToken = (req, res, next) => {
    try {
        // Get token from header
        const authHeader = req.headers.authorization;

        if (!authHeader) {
            return res.status(401).json({
                error: 'Unauthorized',
                message: 'No authorization token provided'
            });
        }

        // Extract token (format: "Bearer <token>")
        const parts = authHeader.split(' ');
        if (parts.length !== 2 || parts[0] !== 'Bearer') {
            return res.status(401).json({
                error: 'Unauthorized',
                message: 'Invalid authorization format. Expected: Bearer <token>'
            });
        }

        const token = parts[1];

        // Verify token
        const decoded = jwt.verify(token, JWT_SECRET);

        // Attach user info to request
        req.user = {
            userId: decoded.userId,
            username: decoded.username,
            role: decoded.role,
            email: decoded.email
        };

        next();
    } catch (error) {
        if (error.name === 'JsonWebTokenError') {
            logger.warn('Invalid JWT token attempt');
            return res.status(401).json({
                error: 'Unauthorized',
                message: 'Invalid token'
            });
        }

        if (error.name === 'TokenExpiredError') {
            logger.info('Expired JWT token attempt');
            return res.status(401).json({
                error: 'Token Expired',
                message: 'Your session has expired. Please login again.'
            });
        }

        logger.error('Token verification error:', error);
        return res.status(500).json({
            error: 'Internal Server Error',
            message: 'Error verifying authentication token'
        });
    }
};

/**
 * Optional authentication middleware
 * Allows request to proceed even if no token is provided
 */
export const optionalAuth = (req, res, next) => {
    try {
        const authHeader = req.headers.authorization;

        if (!authHeader) {
            return next();
        }

        const parts = authHeader.split(' ');
        if (parts.length !== 2 || parts[0] !== 'Bearer') {
            return next();
        }

        const token = parts[1];
        const decoded = jwt.verify(token, JWT_SECRET);

        req.user = {
            userId: decoded.userId,
            username: decoded.username,
            role: decoded.role,
            email: decoded.email
        };

        next();
    } catch (error) {
        // Continue without authentication if token is invalid
        next();
    }
};
