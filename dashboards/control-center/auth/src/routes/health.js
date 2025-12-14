/**
 * Health Check Routes
 * Provides health status for monitoring and load balancers
 */

import express from 'express';
import { logger } from '../utils/logger.js';

const router = express.Router();

// Service start time
const startTime = new Date();

/**
 * GET /health
 * Basic health check endpoint
 */
router.get('/', async (req, res) => {
    const health = {
        status: 'healthy',
        service: 'paw-auth-service',
        version: '1.0.0',
        timestamp: new Date().toISOString(),
        uptime: process.uptime(),
        startTime: startTime.toISOString()
    };

    // Check Redis connection
    if (req.app.locals.redis) {
        try {
            await req.app.locals.redis.ping();
            health.redis = 'connected';
        } catch (error) {
            health.redis = 'disconnected';
            health.status = 'degraded';
            logger.warn('Redis health check failed:', error.message);
        }
    } else {
        health.redis = 'not configured';
    }

    // Memory usage
    const memUsage = process.memoryUsage();
    health.memory = {
        rss: `${Math.round(memUsage.rss / 1024 / 1024)} MB`,
        heapTotal: `${Math.round(memUsage.heapTotal / 1024 / 1024)} MB`,
        heapUsed: `${Math.round(memUsage.heapUsed / 1024 / 1024)} MB`,
        external: `${Math.round(memUsage.external / 1024 / 1024)} MB`
    };

    // CPU usage
    const cpuUsage = process.cpuUsage();
    health.cpu = {
        user: `${Math.round(cpuUsage.user / 1000)} ms`,
        system: `${Math.round(cpuUsage.system / 1000)} ms`
    };

    // Determine overall status
    if (health.redis === 'disconnected') {
        return res.status(503).json(health);
    }

    res.json(health);
});

/**
 * GET /health/ready
 * Readiness probe - checks if service is ready to accept traffic
 */
router.get('/ready', async (req, res) => {
    const ready = {
        ready: true,
        checks: {
            server: 'ok'
        }
    };

    // Check Redis connection
    if (req.app.locals.redis) {
        try {
            await req.app.locals.redis.ping();
            ready.checks.redis = 'ok';
        } catch (error) {
            ready.checks.redis = 'failed';
            ready.ready = false;
        }
    }

    if (!ready.ready) {
        return res.status(503).json(ready);
    }

    res.json(ready);
});

/**
 * GET /health/live
 * Liveness probe - checks if service is alive
 */
router.get('/live', (req, res) => {
    res.json({
        alive: true,
        timestamp: new Date().toISOString()
    });
});

export default router;
