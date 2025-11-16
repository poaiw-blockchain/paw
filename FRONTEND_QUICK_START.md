# Frontend Quick Start Guide

## Quick Setup (5 Minutes)

### Exchange Frontend

```bash
# Navigate to directory
cd external/crypto/exchange-frontend

# Install dependencies
npm install

# Copy environment file
cp .env.example .env

# Start development server
npm run dev
```

Open browser to `http://localhost:3000`

### Browser Wallet Extension

```bash
# Navigate to directory
cd external/crypto/browser-wallet-extension

# Install dependencies
npm install

# Build extension
npm run build

# Load in Chrome:
# 1. Open chrome://extensions/
# 2. Enable "Developer mode"
# 3. Click "Load unpacked"
# 4. Select the dist/ directory
```

## Essential Commands

### Exchange Frontend

| Command                  | Description              |
| ------------------------ | ------------------------ |
| `npm run dev`            | Start development server |
| `npm run build`          | Build for production     |
| `npm run preview`        | Preview production build |
| `npm run lint`           | Check code quality       |
| `npm run format`         | Format code              |
| `npm test`               | Run tests                |
| `npm run security:audit` | Security audit           |

### Browser Wallet Extension

| Command           | Description                 |
| ----------------- | --------------------------- |
| `npm run build`   | Build extension             |
| `npm run watch`   | Build with watch mode       |
| `npm run package` | Create ZIP for distribution |
| `npm run lint`    | Check code quality          |
| `npm run format`  | Format code                 |

## Directory Structure

```
external/crypto/
├── exchange-frontend/
│   ├── index-secure.html      # Main HTML (with CSP)
│   ├── app-secure.js          # Main app logic (secure)
│   ├── config.js              # Configuration
│   ├── security.js            # Security utilities
│   ├── api-client.js          # API client
│   ├── websocket-client.js    # WebSocket client
│   ├── vite.config.js         # Build config
│   ├── package.json           # Dependencies
│   ├── .env.example           # Environment template
│   └── README.md              # Full documentation
│
└── browser-wallet-extension/
    ├── popup.html             # Extension UI
    ├── popup.js               # Extension logic
    ├── background.js          # Service worker
    ├── manifest.json          # Extension manifest
    ├── build.js               # Build script
    ├── package.json           # Dependencies
    └── .env.example           # Environment template
```

## Key Security Features

### 1. Content Security Policy (CSP)

- Located in: `index-secure.html` `<meta>` tags
- Prevents XSS attacks
- Restricts script/style sources

### 2. CSRF Protection

- File: `security.js` → `CSRF` object
- Automatic token management
- Cookie-based storage

### 3. Input Sanitization

- File: `security.js` → `Sanitizer` object
- HTML, username, number, URL sanitization
- DOMPurify integration

### 4. Rate Limiting

- File: `security.js` → `RateLimiter` class
- Configurable limits
- Sliding window algorithm

### 5. Secure Storage

- File: `security.js` → `SecureStorage` object
- Expiration timestamps
- Automatic cleanup

### 6. Password Validation

- File: `security.js` → `PasswordValidator` object
- Minimum 8 characters
- Uppercase, lowercase, number, special char required

## Environment Variables

### Required Variables

**Exchange Frontend (.env):**

```bash
VITE_API_BASE_URL=http://localhost:5000/api
VITE_WS_URL=ws://localhost:5000
VITE_ENABLE_CSP=true
VITE_CSRF_ENABLED=true
```

**Browser Wallet Extension (.env):**

```bash
DEFAULT_API_HOST=http://localhost:8545
ENABLE_CSP=true
```

## Common Tasks

### Add New Security Rule

**ESLint (`.eslintrc.json`):**

```json
{
  "rules": {
    "new-security-rule": "error"
  }
}
```

### Add New Dependency

```bash
# Production dependency
npm install --save package-name

# Development dependency
npm install --save-dev package-name

# Security audit after
npm audit
```

### Update Dependencies

```bash
# Check for updates
npm outdated

# Update all
npm update

# Update specific package
npm update package-name

# Audit security
npm audit fix
```

### Configure API Endpoints

**Development:**
Edit `.env.development`:

```bash
VITE_API_BASE_URL=http://localhost:5000/api
VITE_WS_URL=ws://localhost:5000
```

**Production:**
Edit `.env.production`:

```bash
VITE_API_BASE_URL=https://api.aixn.exchange/api
VITE_WS_URL=wss://ws.aixn.exchange
```

## Testing Checklist

### Before Committing

- [ ] Run `npm run lint` - No errors
- [ ] Run `npm run format` - Code formatted
- [ ] Run `npm test` - Tests passing
- [ ] Run `npm run build` - Build succeeds
- [ ] Run `npm run security:audit` - No high vulnerabilities

### Before Deploying

- [ ] Update version in `package.json`
- [ ] Update `CHANGELOG.md`
- [ ] Run full test suite
- [ ] Build for production
- [ ] Test production build locally
- [ ] Review security scan results
- [ ] Backup current production
- [ ] Deploy to staging first
- [ ] Test on staging
- [ ] Deploy to production
- [ ] Verify production deployment
- [ ] Monitor for errors

## Troubleshooting

### Build Fails

```bash
# Clear cache and reinstall
rm -rf node_modules package-lock.json
npm install

# Clear build directory
npm run clean
npm run build
```

### ESLint Errors

```bash
# Auto-fix issues
npm run lint -- --fix

# Check specific file
npx eslint path/to/file.js
```

### Port Already in Use

```bash
# Find process using port 3000
lsof -i :3000

# Kill process
kill -9 <PID>

# Or use different port
npm run dev -- --port 3001
```

### Module Not Found

```bash
# Reinstall dependencies
npm install

# Clear npm cache
npm cache clean --force
npm install
```

## Performance Tips

### 1. Optimize Imports

```javascript
// Bad - imports entire library
import _ from 'lodash';

// Good - imports specific function
import debounce from 'lodash/debounce';
```

### 2. Lazy Loading

```javascript
// Dynamic import for code splitting
const module = await import('./heavy-module.js');
```

### 3. Minimize Bundle Size

```bash
# Analyze bundle
npm run build
# Check dist/stats.html for bundle analysis
```

### 4. Use Production Build

```bash
# Always use production build for deployment
NODE_ENV=production npm run build
```

## Security Checklist

### Code Review

- [ ] No hardcoded secrets/API keys
- [ ] All user inputs sanitized
- [ ] CSRF tokens on forms
- [ ] Authentication required for sensitive actions
- [ ] Error messages don't leak sensitive info
- [ ] HTTPS/WSS in production
- [ ] CSP headers configured
- [ ] Rate limiting enabled
- [ ] Session timeout configured
- [ ] Strong password requirements

### Deployment

- [ ] Environment variables set
- [ ] HTTPS enabled
- [ ] CSP headers active
- [ ] CORS configured correctly
- [ ] Rate limiting on server
- [ ] Monitoring enabled
- [ ] Backups configured
- [ ] DDoS protection active
- [ ] Security headers set
- [ ] Error tracking enabled

## Getting Help

### Documentation

- Full docs: `external/crypto/exchange-frontend/README.md`
- Security summary: `FRONTEND_BUILD_SECURITY_SUMMARY.md`
- This guide: `FRONTEND_QUICK_START.md`

### Common Issues

- Check browser console for errors
- Check network tab for API failures
- Verify backend is running
- Check environment variables
- Review GitHub Actions logs

### Support

- GitHub Issues: Report bugs
- Pull Requests: Contribute fixes
- Security: Use private disclosure

## Next Steps

1. **Read Full Documentation**: Check README.md files
2. **Review Security**: Read FRONTEND_BUILD_SECURITY_SUMMARY.md
3. **Configure Environment**: Set up .env files
4. **Run Tests**: Verify everything works
5. **Start Development**: Make changes
6. **Submit PR**: Contribute back

---

**Quick Links:**

- [Full Security Summary](./FRONTEND_BUILD_SECURITY_SUMMARY.md)
- [Exchange Frontend README](./external/crypto/exchange-frontend/README.md)
- [Wallet Extension README](./external/crypto/browser-wallet-extension/README.md)
- [GitHub Actions](./.github/workflows/frontend-ci.yml)
