/**
 * Role-Based Access Control (RBAC) Middleware
 */

import { logger } from '../utils/logger.js';

// Define role permissions
const ROLE_PERMISSIONS = {
    admin: ['read', 'write', 'delete', 'manage_users', 'network_control'],
    operator: ['read', 'write', 'network_control'],
    viewer: ['read']
};

/**
 * Check if user has required permission
 * @param {string} permission - Required permission
 * @returns {Function} Express middleware
 */
export const checkPermission = (permission) => {
    return (req, res, next) => {
        if (!req.user) {
            return res.status(401).json({
                error: 'Unauthorized',
                message: 'Authentication required'
            });
        }

        const userRole = req.user.role;
        const permissions = ROLE_PERMISSIONS[userRole] || [];

        if (!permissions.includes(permission)) {
            logger.warn(`Access denied for user ${req.user.username} (role: ${userRole}) - required permission: ${permission}`);
            return res.status(403).json({
                error: 'Forbidden',
                message: `Your role (${userRole}) does not have permission: ${permission}`,
                required: permission,
                available: permissions
            });
        }

        logger.debug(`Permission granted for user ${req.user.username} - permission: ${permission}`);
        next();
    };
};

/**
 * Check if user has required role
 * @param {...string} roles - Allowed roles
 * @returns {Function} Express middleware
 */
export const checkRole = (...roles) => {
    return (req, res, next) => {
        if (!req.user) {
            return res.status(401).json({
                error: 'Unauthorized',
                message: 'Authentication required'
            });
        }

        const userRole = req.user.role;

        if (!roles.includes(userRole)) {
            logger.warn(`Access denied for user ${req.user.username} (role: ${userRole}) - required roles: ${roles.join(', ')}`);
            return res.status(403).json({
                error: 'Forbidden',
                message: `Your role (${userRole}) is not authorized for this action`,
                required: roles,
                current: userRole
            });
        }

        logger.debug(`Role check passed for user ${req.user.username} - role: ${userRole}`);
        next();
    };
};

/**
 * Check if user can access resource
 * Admin can access all resources, others can only access their own
 * @param {Function} getResourceOwnerId - Function to extract owner ID from request
 * @returns {Function} Express middleware
 */
export const checkResourceAccess = (getResourceOwnerId) => {
    return (req, res, next) => {
        if (!req.user) {
            return res.status(401).json({
                error: 'Unauthorized',
                message: 'Authentication required'
            });
        }

        // Admin has access to everything
        if (req.user.role === 'admin') {
            return next();
        }

        // Get resource owner ID
        const resourceOwnerId = getResourceOwnerId(req);

        // Check if user owns the resource
        if (req.user.userId !== resourceOwnerId) {
            logger.warn(`Resource access denied for user ${req.user.username} - resource owner: ${resourceOwnerId}`);
            return res.status(403).json({
                error: 'Forbidden',
                message: 'You do not have permission to access this resource'
            });
        }

        next();
    };
};

/**
 * Get user permissions
 * @param {string} role - User role
 * @returns {string[]} Array of permissions
 */
export const getPermissions = (role) => {
    return ROLE_PERMISSIONS[role] || [];
};

/**
 * Check if role has permission
 * @param {string} role - User role
 * @param {string} permission - Permission to check
 * @returns {boolean} True if role has permission
 */
export const hasPermission = (role, permission) => {
    const permissions = ROLE_PERMISSIONS[role] || [];
    return permissions.includes(permission);
};
