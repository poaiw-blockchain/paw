/**
 * Winston Logger Configuration
 * Provides structured logging with different levels and transports
 */

import winston from 'winston';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Define log format
const logFormat = winston.format.combine(
    winston.format.timestamp({ format: 'YYYY-MM-DD HH:mm:ss' }),
    winston.format.errors({ stack: true }),
    winston.format.splat(),
    winston.format.json()
);

// Console format for development
const consoleFormat = winston.format.combine(
    winston.format.colorize(),
    winston.format.timestamp({ format: 'YYYY-MM-DD HH:mm:ss' }),
    winston.format.printf(({ timestamp, level, message, ...metadata }) => {
        let msg = `${timestamp} [${level}]: ${message}`;
        if (Object.keys(metadata).length > 0) {
            msg += ` ${JSON.stringify(metadata)}`;
        }
        return msg;
    })
);

// Create logger
export const logger = winston.createLogger({
    level: process.env.LOG_LEVEL || 'info',
    format: logFormat,
    defaultMeta: { service: 'paw-auth-service' },
    transports: [
        // Write all logs to console
        new winston.transports.Console({
            format: consoleFormat
        }),
        // Write all logs with level 'error' and below to error.log
        new winston.transports.File({
            filename: path.join(__dirname, '../../logs/error.log'),
            level: 'error',
            maxsize: 5242880, // 5MB
            maxFiles: 5
        }),
        // Write all logs with level 'info' and below to combined.log
        new winston.transports.File({
            filename: path.join(__dirname, '../../logs/combined.log'),
            maxsize: 5242880, // 5MB
            maxFiles: 5
        })
    ],
    exceptionHandlers: [
        new winston.transports.File({
            filename: path.join(__dirname, '../../logs/exceptions.log')
        })
    ],
    rejectionHandlers: [
        new winston.transports.File({
            filename: path.join(__dirname, '../../logs/rejections.log')
        })
    ]
});

// If in production, log to files only (not console)
if (process.env.NODE_ENV === 'production') {
    logger.transports[0].level = 'warn'; // Only show warnings and errors in console
}

export default logger;
