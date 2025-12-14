/**
 * PAW Control Center Authentication Service
 * Provides JWT-based authentication and RBAC authorization
 */

import express from 'express';
import cors from 'cors';
import helmet from 'helmet';
import rateLimit from 'express-rate-limit';
import { createClient } from 'redis';
import dotenv from 'dotenv';
import { logger } from './utils/logger.js';
import authRoutes from './routes/auth.js';
import healthRoutes from './routes/health.js';
import { errorHandler } from './middleware/errorHandler.js';

dotenv.config();

const app = express();
const PORT = process.env.PORT || 3000;

// Redis client for session management
let redisClient;
const initRedis = async () => {
    try {
        redisClient = createClient({
            url: process.env.REDIS_URL || 'redis://localhost:6379',
            password: process.env.REDIS_PASSWORD
        });

        redisClient.on('error', (err) => logger.error('Redis Client Error', err));
        redisClient.on('connect', () => logger.info('Redis client connected'));

        await redisClient.connect();
        return redisClient;
    } catch (error) {
        logger.error('Failed to connect to Redis:', error);
        return null;
    }
};

// Middleware
app.use(helmet({
    contentSecurityPolicy: {
        directives: {
            defaultSrc: ["'self'"],
            styleSrc: ["'self'", "'unsafe-inline'"],
            scriptSrc: ["'self'"],
            imgSrc: ["'self'", 'data:', 'https:'],
        },
    },
    hsts: {
        maxAge: 31536000,
        includeSubDomains: true,
        preload: true
    }
}));

app.use(cors({
    origin: process.env.CORS_ORIGIN || 'http://localhost:11200',
    credentials: true,
    methods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'],
    allowedHeaders: ['Content-Type', 'Authorization']
}));

app.use(express.json({ limit: '10mb' }));
app.use(express.urlencoded({ extended: true, limit: '10mb' }));

// Rate limiting
const limiter = rateLimit({
    windowMs: 15 * 60 * 1000, // 15 minutes
    max: 100, // Limit each IP to 100 requests per windowMs
    message: 'Too many requests from this IP, please try again later.',
    standardHeaders: true,
    legacyHeaders: false,
});
app.use('/api/', limiter);

// Strict rate limiting for auth endpoints
const authLimiter = rateLimit({
    windowMs: 15 * 60 * 1000,
    max: 5, // 5 login attempts per 15 minutes
    message: 'Too many authentication attempts, please try again later.',
    skipSuccessfulRequests: true
});
app.use('/api/auth/login', authLimiter);

// Request logging
app.use((req, res, next) => {
    logger.info(`${req.method} ${req.path}`, {
        ip: req.ip,
        userAgent: req.get('user-agent')
    });
    next();
});

// Make Redis client available to routes
app.locals.redis = null;

// Routes
app.use('/api/auth', authRoutes);
app.use('/health', healthRoutes);

// Root endpoint
app.get('/', (req, res) => {
    res.json({
        service: 'PAW Control Center Authentication Service',
        version: '1.0.0',
        status: 'running',
        endpoints: {
            health: '/health',
            auth: '/api/auth',
            login: '/api/auth/login',
            register: '/api/auth/register',
            refresh: '/api/auth/refresh',
            verify: '/api/auth/verify'
        }
    });
});

// Error handling
app.use(errorHandler);

// 404 handler
app.use((req, res) => {
    res.status(404).json({
        error: 'Not Found',
        message: `Cannot ${req.method} ${req.path}`,
        path: req.path
    });
});

// Start server
const startServer = async () => {
    try {
        // Initialize Redis
        const redis = await initRedis();
        app.locals.redis = redis;

        // Start Express server
        app.listen(PORT, () => {
            logger.info(`Authentication service started on port ${PORT}`);
            logger.info(`Environment: ${process.env.NODE_ENV || 'development'}`);
            logger.info(`CORS origin: ${process.env.CORS_ORIGIN || 'http://localhost:11200'}`);
        });
    } catch (error) {
        logger.error('Failed to start server:', error);
        process.exit(1);
    }
};

// Graceful shutdown
const gracefulShutdown = async () => {
    logger.info('Received shutdown signal, closing server gracefully...');

    if (redisClient) {
        await redisClient.quit();
        logger.info('Redis connection closed');
    }

    process.exit(0);
};

process.on('SIGTERM', gracefulShutdown);
process.on('SIGINT', gracefulShutdown);

// Start the server
startServer();

export default app;
