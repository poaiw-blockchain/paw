# PAW Blockchain API Documentation

Complete, production-ready API documentation portal for PAW Blockchain.

## Features

- **OpenAPI 3.0 Specification**: Complete API spec with all endpoints
- **Interactive Swagger UI**: Try out API calls directly in browser
- **Redoc Documentation**: Beautiful, responsive API reference
- **Code Examples**: Working examples in cURL, JavaScript, Python, and Go
- **Comprehensive Guides**: Authentication, WebSockets, rate limiting, and error handling
- **Postman Collection**: Import and test all endpoints in Postman
- **Docker Support**: One-command deployment with Docker Compose
- **Search Functionality**: Quick endpoint discovery
- **Dark Theme**: Modern, easy-on-the-eyes design

## Quick Start

### Option 1: Docker (Recommended)

```bash
# Start the documentation server
docker-compose up -d

# Access the portal
open http://localhost:8080
```

The documentation will be available at:
- **Main Portal**: http://localhost:8080
- **Swagger UI**: http://localhost:8080/swagger-ui/
- **Redoc**: http://localhost:8080/redoc/

### Option 2: Local Web Server

```bash
# Using Python
python -m http.server 8080

# Using Node.js
npx http-server -p 8080

# Using PHP
php -S localhost:8080
```

### Option 3: Static File Server

Simply open `index.html` in your web browser.

## Documentation Structure

```
docs/api/
├── index.html                 # Main documentation portal
├── openapi.yaml              # Complete OpenAPI 3.0 specification
├── swagger-ui/
│   └── index.html           # Swagger UI interface
├── redoc/
│   └── index.html           # Redoc interface
├── examples/
│   ├── curl.md              # cURL examples
│   ├── javascript.md        # JavaScript/TypeScript examples
│   ├── python.md            # Python examples
│   └── go.md                # Go examples
├── guides/
│   ├── authentication.md    # Authentication guide
│   ├── websockets.md        # WebSocket guide
│   ├── rate-limiting.md     # Rate limiting guide
│   └── errors.md            # Error codes reference
├── postman/
│   └── PAW-API.postman_collection.json
├── tests/
│   ├── openapi-validation.test.js
│   ├── examples.test.js
│   └── links.test.js
├── docker-compose.yml       # Docker deployment
├── nginx.conf              # Nginx configuration
└── README.md               # This file
```

## API Modules

### DEX Module
- List pools
- Get pool details
- Estimate swaps
- Create pools
- Swap tokens
- Add/remove liquidity

### Oracle Module
- List price feeds
- Get specific prices
- Submit prices (validators)
- Get oracle parameters

### Compute Module
- List compute tasks
- Get task details
- Submit tasks
- List providers

### Cosmos SDK Modules
- **Bank**: Token transfers and balances
- **Staking**: Validators and delegations
- **Governance**: Proposals and voting
- **Auth**: Account information

### Tendermint RPC
- Node status
- Blocks
- Transactions
- Validators

## Usage Examples

### JavaScript

```javascript
const API_URL = 'http://localhost:1317';

// Get all DEX pools
const pools = await fetch(`${API_URL}/paw/dex/v1/pools`)
  .then(r => r.json());

// Get BTC price
const btcPrice = await fetch(`${API_URL}/paw/oracle/v1/prices/BTC%2FUSD`)
  .then(r => r.json());
```

### Python

```python
import requests

API_URL = 'http://localhost:1317'

# Get all DEX pools
pools = requests.get(f'{API_URL}/paw/dex/v1/pools').json()

# Get BTC price
btc_price = requests.get(f'{API_URL}/paw/oracle/v1/prices/BTC%2FUSD').json()
```

### cURL

```bash
# Get all DEX pools
curl http://localhost:1317/paw/dex/v1/pools

# Get BTC price
curl http://localhost:1317/paw/oracle/v1/prices/BTC%2FUSD
```

## Base URLs

- **Mainnet**: https://api.paw.network
- **Testnet**: https://testnet-api.paw.network
- **Local**: http://localhost:1317

## Rate Limits

- **Public endpoints**: 100 requests/minute
- **Authenticated endpoints**: 1000 requests/minute

## Testing

Run the test suite to validate the documentation:

```bash
# Install dependencies
npm install

# Run tests
npm test

# Run specific test suites
npm run test:openapi      # Validate OpenAPI spec
npm run test:examples     # Test code examples
npm run test:links        # Check for broken links
```

## Building for Production

### Update OpenAPI Spec

The OpenAPI specification should be regenerated when API changes:

```bash
# If you have spec generation tools
make generate-openapi

# Or manually update openapi.yaml
```

### Deploy to Production

```bash
# Build Docker image
docker build -t paw-api-docs .

# Deploy to server
docker push paw-api-docs
kubectl apply -f k8s/api-docs-deployment.yaml
```

## Customization

### Update Base URL

Edit the `servers` section in `openapi.yaml`:

```yaml
servers:
  - url: https://your-api.github.com
    description: Production server
```

### Customize Theme

Edit CSS variables in `index.html`:

```css
:root {
  --primary-color: #6366f1;
  --secondary-color: #8b5cf6;
  /* ... */
}
```

## Contributing

When adding new endpoints:

1. Update `openapi.yaml` with new endpoint definitions
2. Add examples to relevant language files in `examples/`
3. Update Postman collection
4. Add tests to `tests/`
5. Update this README if needed

## Troubleshooting

### CORS Issues

If accessing from a different origin, ensure your API server has CORS enabled:

```javascript
// Express.js example
app.use(cors({
  origin: '*',
  methods: ['GET', 'POST']
}));
```

### OpenAPI Validation Errors

Validate your OpenAPI spec:

```bash
npx @redocly/cli lint openapi.yaml
```

### Swagger UI Not Loading

1. Check that `openapi.yaml` is accessible
2. Verify the relative path in `swagger-ui/index.html`
3. Check browser console for errors

## Resources

- [OpenAPI Specification](https://swagger.io/specification/)
- [Swagger UI Documentation](https://swagger.io/tools/swagger-ui/)
- [Redoc Documentation](https://redocly.com/docs/redoc/)
- [Cosmos SDK Docs](https://docs.cosmos.network/)

## Support

- **Documentation**: https://docs.paw.network
- **Discord**: https://discord.gg/paw
- ****: https://github.com/paw-chain/paw

## License

MIT License - see LICENSE file for details

---

**Note**: This documentation portal is automatically generated and maintained. For the most up-to-date API information, always refer to the latest OpenAPI specification.
