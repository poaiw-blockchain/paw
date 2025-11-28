# PAW Documentation Portal

Comprehensive documentation for PAW Blockchain.

## Development

### Installation

```bash
npm install
```

### Local Development

```bash
npm run dev
```

Visit http://localhost:5173 (local dev server)

### Build

```bash
npm run build
```

### Preview Build

```bash
npm run preview
```

### Testing

```bash
# Run all tests
npm test

# Run specific tests
npm run test:links
npm run test:build
npm run test:search
```

## Structure

```
docs/portal/
├── .vitepress/          # VitePress configuration
│   ├── config.js        # Site configuration
│   └── theme/           # Custom theme
├── guide/               # User guides
├── developer/           # Developer documentation
├── validator/           # Validator guides
├── reference/           # Technical reference
├── tests/               # Test suites
├── meta/                # Portal implementation documentation
│   ├── IMPLEMENTATION_COMPLETE.md
│   ├── IMPLEMENTATION_SUMMARY.md
│   ├── DOCUMENTATION_PORTAL_SUMMARY.md
│   ├── FILES_CREATED.md
│   ├── TEST_RESULTS.md
│   └── QUICK_REFERENCE.md
├── index.md             # Homepage
├── faq.md              # FAQ
├── glossary.md         # Glossary
└── package.json        # Dependencies
```

## Contributing

1. Edit markdown files in respective directories
2. Test locally with `npm run dev`
3. Run tests with `npm test`
4. Submit pull request

## License

MIT
