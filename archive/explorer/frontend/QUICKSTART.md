# PAW Chain Explorer Frontend - Quick Start Guide

## Prerequisites

Ensure you have the following installed:
- **Node.js**: v18.17.0 or higher
- **npm**: v9.0.0 or higher

Check your versions:
```bash
node --version
npm --version
```

## Installation

### 1. Navigate to the Frontend Directory
```bash
cd /home/decri/blockchain-projects/paw/explorer/frontend
```

### 2. Install Dependencies
```bash
npm install
```

This will install all required dependencies including:
- Next.js 14
- React 18
- TypeScript
- Tailwind CSS
- TanStack Query
- Recharts
- Radix UI components
- And many more...

### 3. Configure Environment Variables

Create a `.env.local` file:
```bash
cp .env.example .env.local
```

Edit `.env.local` with your configuration:
```env
# API Configuration
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
NEXT_PUBLIC_WS_URL=ws://localhost:8080/ws
NEXT_PUBLIC_GRAPHQL_URL=http://localhost:8080/graphql

# Application Configuration
NEXT_PUBLIC_APP_NAME=PAW Chain Explorer
NEXT_PUBLIC_CHAIN_ID=paw-1
NEXT_PUBLIC_CHAIN_NAME=PAW Chain
```

## Running the Development Server

### Start the Server
```bash
npm run dev
```

The application will be available at:
- **URL**: http://localhost:3000
- **Hot Reload**: Enabled (changes auto-refresh)

### Expected Console Output
```
▲ Next.js 14.2.0
- Local:        http://localhost:3000
- Environments: .env.local

✓ Ready in 2.5s
```

## Verify Installation

### 1. Open Your Browser
Navigate to: http://localhost:3000

### 2. Check These Pages
- **Home**: http://localhost:3000 (Dashboard with network stats)
- **Blocks**: http://localhost:3000/blocks (Block list)
- **Transactions**: http://localhost:3000/transactions (Transaction list)

### 3. Test Search
- Use the search bar in the header
- Try searching for:
  - Block number (e.g., "1000")
  - Transaction hash
  - Account address

### 4. Test Dark Mode
- Click the moon/sun icon in the header
- Theme should toggle between light and dark

## Building for Production

### 1. Build the Application
```bash
npm run build
```

This creates an optimized production build in `.next/` directory.

Expected output:
```
Route (app)                              Size     First Load JS
┌ ○ /                                    5.2 kB        120 kB
├ ○ /account/[address]                   8.4 kB        125 kB
├ ○ /block/[height]                      7.1 kB        123 kB
├ ○ /blocks                              6.5 kB        122 kB
├ ○ /transactions                        7.8 kB        124 kB
└ ○ /tx/[hash]                           8.9 kB        126 kB

○  (Static)  automatically rendered as static HTML
```

### 2. Start Production Server
```bash
npm start
```

Production server runs on: http://localhost:3000

### 3. Optional: Custom Port
```bash
PORT=8080 npm start
```

## Development Commands

### Type Checking
```bash
npm run type-check
```
Runs TypeScript compiler to check for type errors.

### Linting
```bash
npm run lint
```
Runs ESLint to check code quality.

### Code Formatting
```bash
npm run format
```
Formats code using Prettier.

### Run All Checks
```bash
npm run type-check && npm run lint
```

## Troubleshooting

### Port Already in Use
If port 3000 is already in use:
```bash
# Kill the process using port 3000 (Linux/Mac)
lsof -ti:3000 | xargs kill -9

# Or use a different port
PORT=3001 npm run dev
```

### Module Not Found Errors
```bash
# Clear node_modules and reinstall
rm -rf node_modules package-lock.json
npm install
```

### Build Errors
```bash
# Clear Next.js cache
rm -rf .next
npm run build
```

### TypeScript Errors
```bash
# Regenerate TypeScript types
npm run type-check
```

## API Backend Requirement

The frontend requires the PAW Chain Explorer API backend to be running:

### Check Backend Status
```bash
# The backend should be running on:
curl http://localhost:8080/api/v1/stats/network
```

### If Backend is Not Running
1. Navigate to the backend directory
2. Start the indexer and API server
3. Verify it's accessible on port 8080

## Testing the Application

### Manual Testing Checklist

#### Home Page (/)
- [ ] Network statistics display
- [ ] Latest blocks section shows blocks
- [ ] Latest transactions section shows transactions
- [ ] Charts render without errors
- [ ] WebSocket connection indicator shows status
- [ ] Search bar is functional

#### Block Detail Page (/block/[height])
- [ ] Block information displays correctly
- [ ] Transaction list shows all transactions
- [ ] Previous/Next navigation works
- [ ] Copy hash button works
- [ ] Links to proposer work

#### Transaction Detail Page (/tx/[hash])
- [ ] Transaction status displays
- [ ] Messages tab shows transaction messages
- [ ] Events tab shows transaction events
- [ ] Raw log tab shows raw data
- [ ] Copy hash button works

#### Account Page (/account/[address])
- [ ] Account statistics display
- [ ] Transactions tab shows history
- [ ] Balances tab shows balances
- [ ] Tokens tab shows tokens
- [ ] Pagination works

#### Blocks List (/blocks)
- [ ] Block list displays
- [ ] Search functionality works
- [ ] Sorting works
- [ ] Pagination works
- [ ] Auto-refresh updates data

#### Transactions List (/transactions)
- [ ] Transaction list displays
- [ ] Filter by status works
- [ ] Search functionality works
- [ ] Pagination works
- [ ] Auto-refresh updates data

#### Global Features
- [ ] Dark mode toggle works
- [ ] Mobile responsive design
- [ ] Search autocomplete works
- [ ] Loading states show
- [ ] Error pages display properly

## Project Structure Overview

```
frontend/
├── app/                    # Next.js App Router pages
│   ├── account/           # Account detail page
│   ├── block/             # Block detail page
│   ├── blocks/            # Blocks list page
│   ├── tx/                # Transaction detail page
│   ├── transactions/      # Transactions list page
│   ├── layout.tsx         # Root layout
│   ├── page.tsx           # Home page
│   ├── error.tsx          # Error boundary
│   └── loading.tsx        # Loading state
├── components/
│   ├── ui/                # Reusable UI components
│   ├── charts/            # Chart components
│   ├── layout/            # Header, Footer
│   ├── search/            # Search components
│   └── blocks/            # Block components
├── hooks/                 # Custom React hooks
├── lib/
│   ├── api.ts            # API client
│   └── utils.ts          # Utilities
├── public/               # Static assets
├── next.config.js        # Next.js config
├── tailwind.config.ts    # Tailwind config
└── package.json          # Dependencies
```

## Next Steps

1. **Explore the Codebase**
   - Check `lib/api.ts` for API integration
   - Review `lib/utils.ts` for utility functions
   - Examine components in `components/ui/`

2. **Customize Styling**
   - Edit `app/globals.css` for theme colors
   - Modify `tailwind.config.ts` for custom styles

3. **Add Features**
   - Validators page
   - DEX analytics
   - Oracle price feeds
   - Governance dashboard

4. **Deploy**
   - Build production version
   - Deploy to Vercel, Netlify, or your hosting provider
   - Configure environment variables

## Getting Help

### Documentation
- **Next.js**: https://nextjs.org/docs
- **React**: https://react.dev
- **Tailwind CSS**: https://tailwindcss.com/docs
- **TanStack Query**: https://tanstack.com/query/latest/docs/react/overview

### Common Issues
- Check the console for error messages
- Verify API backend is running
- Ensure environment variables are set correctly
- Clear browser cache and cookies

### Support
-  Issues: Report bugs and request features
- Discord: Join the PAW Chain community
- Documentation: Read the full README.md

## Success Indicators

Your installation is successful if:
- ✅ Development server starts without errors
- ✅ Home page loads and displays data
- ✅ Dark mode toggle works
- ✅ Search functionality works
- ✅ All pages are accessible
- ✅ No console errors (except API errors if backend is down)

## Development Tips

1. **Hot Reload**: Save any file to see instant changes
2. **TypeScript**: VS Code provides excellent IntelliSense
3. **Tailwind IntelliSense**: Install the VS Code extension
4. **React DevTools**: Install browser extension for debugging
5. **Component Development**: Use the existing components as templates

Happy developing! 🚀
