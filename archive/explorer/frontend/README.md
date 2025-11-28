# PAW Chain Explorer Frontend

Production-ready blockchain explorer frontend for the PAW Chain network, built with Next.js 14, React, and TypeScript.

## Features

- **Real-time Updates**: WebSocket integration for live blockchain data
- **Responsive Design**: Mobile-first approach with Tailwind CSS
- **Advanced Search**: Fast search with autocomplete for blocks, transactions, and addresses
- **Dark Mode**: Built-in theme switching with next-themes
- **Data Visualization**: Interactive charts using Recharts
- **Type-Safe**: Full TypeScript support with strict type checking
- **Performance Optimized**: Server-side rendering, code splitting, and caching
- **SEO Friendly**: Proper meta tags and structured data

## Tech Stack

- **Framework**: Next.js 14 (App Router)
- **Language**: TypeScript
- **Styling**: Tailwind CSS
- **UI Components**: Radix UI primitives
- **State Management**: TanStack Query (React Query)
- **Charts**: Recharts
- **Forms**: React Hook Form + Zod
- **Theme**: next-themes
- **Icons**: Lucide React

## Getting Started

### Prerequisites

- Node.js 18.17.0 or higher
- npm 9.0.0 or higher

### Installation

1. Clone the repository and navigate to the frontend directory:
   ```bash
   cd explorer/frontend
   ```

2. Install dependencies:
   ```bash
   npm install
   ```

3. Create environment file:
   ```bash
   cp .env.example .env.local
   ```

4. Configure your environment variables in `.env.local`:
   ```env
   NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
   NEXT_PUBLIC_WS_URL=ws://localhost:8080/ws
   NEXT_PUBLIC_GRAPHQL_URL=http://localhost:8080/graphql
   ```

### Development

Run the development server:

```bash
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) in your browser.

### Building for Production

```bash
npm run build
npm start
```

### Testing

```bash
npm run test
npm run test:watch
npm run test:coverage
```

### Linting

```bash
npm run lint
npm run type-check
```

## Project Structure

```
frontend/
├── app/                      # Next.js app directory (routes)
│   ├── account/[address]/   # Account detail page
│   ├── block/[height]/      # Block detail page
│   ├── blocks/              # Blocks list page
│   ├── tx/[hash]/           # Transaction detail page
│   ├── transactions/        # Transactions list page
│   ├── layout.tsx           # Root layout
│   ├── page.tsx             # Home page
│   ├── error.tsx            # Error boundary
│   ├── loading.tsx          # Loading state
│   └── globals.css          # Global styles
├── components/
│   ├── blocks/              # Block-related components
│   ├── charts/              # Chart components (Recharts)
│   ├── layout/              # Layout components (Header, Footer)
│   ├── search/              # Search components
│   └── ui/                  # Reusable UI components
├── hooks/                   # Custom React hooks
│   ├── use-search.ts       # Search functionality
│   ├── use-websocket.ts    # WebSocket connection
│   └── use-copy-to-clipboard.ts
├── lib/
│   ├── api.ts              # API client and types
│   └── utils.ts            # Utility functions
├── public/                  # Static assets
├── next.config.js          # Next.js configuration
├── tailwind.config.ts      # Tailwind CSS configuration
├── tsconfig.json           # TypeScript configuration
└── package.json            # Dependencies

```

## Key Pages

### Home Page (`/`)
- Network statistics dashboard
- Latest blocks and transactions
- Real-time updates via WebSocket
- Quick links to DEX, Oracle, and Compute modules
- Network activity charts

### Block Detail Page (`/block/[height]`)
- Block information and metadata
- List of transactions in the block
- Proposer information
- Navigation to previous/next blocks

### Transaction Detail Page (`/tx/[hash]`)
- Transaction status and details
- Messages and events breakdown
- Gas usage information
- Raw log data

### Account Page (`/account/[address]`)
- Account balances and tokens
- Transaction history with pagination
- Total sent/received amounts
- Account timeline

### Blocks List (`/blocks`)
- Paginated list of all blocks
- Search and filter functionality
- Real-time updates
- Block time statistics

### Transactions List (`/transactions`)
- Paginated list of all transactions
- Filter by status (success/failed)
- Search functionality
- Real-time updates

## Components

### UI Components (`components/ui/`)
All UI components are built using Radix UI primitives and styled with Tailwind CSS:
- Button, Card, Input, Badge, Skeleton
- Table, Tabs, Select, Tooltip
- Fully accessible and keyboard navigable

### Chart Components (`components/charts/`)
- **NetworkStatsChart**: Transaction and block count over time
- **TransactionVolumeChart**: Transaction volume visualization
- **PriceChart**: Asset price charts with area visualization

### Search Components (`components/search/`)
- **SearchBar**: Main search input with autocomplete
- **SearchResults**: Dropdown with search results

## Hooks

### `use-search`
Handles search functionality with debouncing and navigation:
```tsx
const { search, results, isSearching, error } = useSearch()
```

### `use-websocket`
Manages WebSocket connections for real-time updates:
```tsx
const { isConnected, lastMessage, sendMessage } = useWebSocket()
```

### `use-copy-to-clipboard`
Provides clipboard copy functionality:
```tsx
const { copy, isCopied } = useCopyToClipboard()
```

## API Integration

The frontend integrates with the PAW Chain Explorer API through a type-safe client (`lib/api.ts`):

- REST API endpoints for all blockchain data
- WebSocket support for real-time updates
- GraphQL endpoint support
- Automatic error handling and retries
- Request/response interceptors

## Styling

### Tailwind CSS
- Custom color palette with CSS variables
- Dark mode support via `next-themes`
- Responsive breakpoints
- Custom animations

### Theme Variables
Located in `app/globals.css`:
- Light and dark mode color schemes
- Consistent design tokens
- Accessible color contrasts

## Performance Optimizations

1. **Server-Side Rendering (SSR)**: Initial page load optimization
2. **Code Splitting**: Automatic route-based splitting
3. **Image Optimization**: Next.js Image component
4. **Caching**: React Query for intelligent data caching
5. **Debouncing**: Search input debouncing
6. **Lazy Loading**: Components and routes

## Browser Support

- Chrome (latest)
- Firefox (latest)
- Safari (latest)
- Edge (latest)

## Contributing

1. Follow the existing code style
2. Write TypeScript with strict mode
3. Add proper type definitions
4. Test your changes
5. Update documentation as needed

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `NEXT_PUBLIC_API_URL` | API endpoint | `http://localhost:8080/api/v1` |
| `NEXT_PUBLIC_WS_URL` | WebSocket endpoint | `ws://localhost:8080/ws` |
| `NEXT_PUBLIC_GRAPHQL_URL` | GraphQL endpoint | `http://localhost:8080/graphql` |

## Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm start` - Start production server
- `npm run lint` - Run ESLint
- `npm run type-check` - Run TypeScript compiler check
- `npm run format` - Format code with Prettier
- `npm test` - Run tests
- `npm run test:watch` - Run tests in watch mode
- `npm run test:coverage` - Generate coverage report

## License

Copyright (c) 2024 PAW Chain. All rights reserved.

## Support

For issues and questions:
-  Issues: https://github.com/pawchain/explorer/issues
- Documentation: https://docs.pawchain.io
- Discord: https://discord.gg/pawchain
