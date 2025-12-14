/**
 * Authentication Routes
 * Handles login, registration, token refresh, and verification
 */

import express from 'express';
import jwt from 'jsonwebtoken';
import bcrypt from 'bcrypt';
import Joi from 'joi';
import { logger } from '../utils/logger.js';
import { verifyToken } from '../middleware/auth.js';
import { checkPermission } from '../middleware/rbac.js';

const router = express.Router();

// JWT configuration
const JWT_SECRET = process.env.JWT_SECRET || 'paw-dashboard-secret-change-in-production';
const JWT_EXPIRY = process.env.JWT_EXPIRY || '24h';
const REFRESH_TOKEN_EXPIRY = process.env.REFRESH_TOKEN_EXPIRY || '7d';

// In-memory user store (replace with database in production)
const users = new Map([
    ['admin', {
        id: '1',
        username: 'admin',
        password: await bcrypt.hash('admin123', 10),
        role: 'admin',
        email: 'admin@paw.network',
        createdAt: new Date().toISOString()
    }],
    ['operator', {
        id: '2',
        username: 'operator',
        password: await bcrypt.hash('operator123', 10),
        role: 'operator',
        email: 'operator@paw.network',
        createdAt: new Date().toISOString()
    }],
    ['viewer', {
        id: '3',
        username: 'viewer',
        password: await bcrypt.hash('viewer123', 10),
        role: 'viewer',
        email: 'viewer@paw.network',
        createdAt: new Date().toISOString()
    }]
]);

// Validation schemas
const loginSchema = Joi.object({
    username: Joi.string().alphanum().min(3).max(30).required(),
    password: Joi.string().min(6).required()
});

const registerSchema = Joi.object({
    username: Joi.string().alphanum().min(3).max(30).required(),
    password: Joi.string().min(6).required(),
    email: Joi.string().email().required(),
    role: Joi.string().valid('admin', 'operator', 'viewer').default('viewer')
});

/**
 * POST /api/auth/login
 * Authenticate user and return JWT tokens
 */
router.post('/login', async (req, res) => {
    try {
        // Validate input
        const { error, value } = loginSchema.validate(req.body);
        if (error) {
            return res.status(400).json({
                error: 'Validation Error',
                details: error.details[0].message
            });
        }

        const { username, password } = value;

        // Find user
        const user = users.get(username);
        if (!user) {
            logger.warn(`Login attempt with invalid username: ${username}`);
            return res.status(401).json({
                error: 'Authentication Failed',
                message: 'Invalid username or password'
            });
        }

        // Verify password
        const isValidPassword = await bcrypt.compare(password, user.password);
        if (!isValidPassword) {
            logger.warn(`Failed login attempt for user: ${username}`);
            return res.status(401).json({
                error: 'Authentication Failed',
                message: 'Invalid username or password'
            });
        }

        // Generate access token
        const accessToken = jwt.sign(
            {
                userId: user.id,
                username: user.username,
                role: user.role,
                email: user.email
            },
            JWT_SECRET,
            { expiresIn: JWT_EXPIRY }
        );

        // Generate refresh token
        const refreshToken = jwt.sign(
            {
                userId: user.id,
                username: user.username,
                type: 'refresh'
            },
            JWT_SECRET,
            { expiresIn: REFRESH_TOKEN_EXPIRY }
        );

        // Store refresh token in Redis (if available)
        if (req.app.locals.redis) {
            await req.app.locals.redis.setEx(
                `refresh:${user.id}`,
                7 * 24 * 60 * 60, // 7 days
                refreshToken
            );
        }

        logger.info(`User logged in successfully: ${username}`);

        res.json({
            success: true,
            accessToken,
            refreshToken,
            user: {
                id: user.id,
                username: user.username,
                role: user.role,
                email: user.email
            }
        });
    } catch (error) {
        logger.error('Login error:', error);
        res.status(500).json({
            error: 'Internal Server Error',
            message: 'An error occurred during authentication'
        });
    }
});

/**
 * POST /api/auth/register
 * Register a new user (admin only)
 */
router.post('/register', verifyToken, checkPermission('manage_users'), async (req, res) => {
    try {
        // Validate input
        const { error, value } = registerSchema.validate(req.body);
        if (error) {
            return res.status(400).json({
                error: 'Validation Error',
                details: error.details[0].message
            });
        }

        const { username, password, email, role } = value;

        // Check if user already exists
        if (users.has(username)) {
            return res.status(409).json({
                error: 'User Already Exists',
                message: `Username '${username}' is already taken`
            });
        }

        // Hash password
        const hashedPassword = await bcrypt.hash(password, 10);

        // Create new user
        const newUser = {
            id: (users.size + 1).toString(),
            username,
            password: hashedPassword,
            email,
            role,
            createdAt: new Date().toISOString(),
            createdBy: req.user.username
        };

        users.set(username, newUser);

        logger.info(`New user registered: ${username} (role: ${role}) by ${req.user.username}`);

        res.status(201).json({
            success: true,
            message: 'User created successfully',
            user: {
                id: newUser.id,
                username: newUser.username,
                email: newUser.email,
                role: newUser.role,
                createdAt: newUser.createdAt
            }
        });
    } catch (error) {
        logger.error('Registration error:', error);
        res.status(500).json({
            error: 'Internal Server Error',
            message: 'An error occurred during registration'
        });
    }
});

/**
 * POST /api/auth/refresh
 * Refresh access token using refresh token
 */
router.post('/refresh', async (req, res) => {
    try {
        const { refreshToken } = req.body;

        if (!refreshToken) {
            return res.status(400).json({
                error: 'Bad Request',
                message: 'Refresh token is required'
            });
        }

        // Verify refresh token
        const decoded = jwt.verify(refreshToken, JWT_SECRET);

        if (decoded.type !== 'refresh') {
            return res.status(401).json({
                error: 'Invalid Token',
                message: 'Token is not a refresh token'
            });
        }

        // Verify token exists in Redis (if available)
        if (req.app.locals.redis) {
            const storedToken = await req.app.locals.redis.get(`refresh:${decoded.userId}`);
            if (storedToken !== refreshToken) {
                return res.status(401).json({
                    error: 'Invalid Token',
                    message: 'Refresh token has been revoked'
                });
            }
        }

        // Find user
        const user = Array.from(users.values()).find(u => u.id === decoded.userId);
        if (!user) {
            return res.status(401).json({
                error: 'User Not Found',
                message: 'User associated with this token no longer exists'
            });
        }

        // Generate new access token
        const accessToken = jwt.sign(
            {
                userId: user.id,
                username: user.username,
                role: user.role,
                email: user.email
            },
            JWT_SECRET,
            { expiresIn: JWT_EXPIRY }
        );

        logger.info(`Access token refreshed for user: ${user.username}`);

        res.json({
            success: true,
            accessToken
        });
    } catch (error) {
        if (error.name === 'JsonWebTokenError') {
            return res.status(401).json({
                error: 'Invalid Token',
                message: 'Refresh token is invalid'
            });
        }
        if (error.name === 'TokenExpiredError') {
            return res.status(401).json({
                error: 'Token Expired',
                message: 'Refresh token has expired'
            });
        }

        logger.error('Token refresh error:', error);
        res.status(500).json({
            error: 'Internal Server Error',
            message: 'An error occurred during token refresh'
        });
    }
});

/**
 * GET /api/auth/verify
 * Verify access token and return user info
 */
router.get('/verify', verifyToken, (req, res) => {
    res.json({
        success: true,
        user: req.user,
        message: 'Token is valid'
    });
});

/**
 * POST /api/auth/logout
 * Logout user and revoke refresh token
 */
router.post('/logout', verifyToken, async (req, res) => {
    try {
        // Remove refresh token from Redis (if available)
        if (req.app.locals.redis) {
            await req.app.locals.redis.del(`refresh:${req.user.userId}`);
        }

        logger.info(`User logged out: ${req.user.username}`);

        res.json({
            success: true,
            message: 'Logged out successfully'
        });
    } catch (error) {
        logger.error('Logout error:', error);
        res.status(500).json({
            error: 'Internal Server Error',
            message: 'An error occurred during logout'
        });
    }
});

/**
 * GET /api/auth/me
 * Get current user information
 */
router.get('/me', verifyToken, (req, res) => {
    const user = Array.from(users.values()).find(u => u.id === req.user.userId);

    if (!user) {
        return res.status(404).json({
            error: 'User Not Found',
            message: 'User information not available'
        });
    }

    res.json({
        success: true,
        user: {
            id: user.id,
            username: user.username,
            email: user.email,
            role: user.role,
            createdAt: user.createdAt
        }
    });
});

export default router;
