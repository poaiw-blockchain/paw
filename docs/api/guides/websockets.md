# WebSocket Guide

## Overview

PAW Blockchain supports WebSocket subscriptions for real-time event streaming.

## Connection

```javascript
const ws = new WebSocket('ws://localhost:26657/websocket');

ws.onopen = () => {
  console.log('Connected to PAW WebSocket');
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Received:', data);
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('WebSocket closed');
};
```

## Event Subscriptions

### New Block Events

```javascript
ws.send(JSON.stringify({
  jsonrpc: '2.0',
  method: 'subscribe',
  params: ["tm.event='NewBlock'"],
  id: 1
}));
```

### Transaction Events

```javascript
ws.send(JSON.stringify({
  jsonrpc: '2.0',
  method: 'subscribe',
  params: ["tm.event='Tx'"],
  id: 2
}));
```

### Custom Module Events

```javascript
// DEX swap events
ws.send(JSON.stringify({
  jsonrpc: '2.0',
  method: 'subscribe',
  params: ["dex.swap.pool_id=1"],
  id: 3
}));

// Oracle price updates
ws.send(JSON.stringify({
  jsonrpc: '2.0',
  method: 'subscribe',
  params: ["oracle.price_update.asset='BTC/USD'"],
  id: 4
}));
```

## Event Types

- `tm.event='NewBlock'` - New blocks
- `tm.event='Tx'` - All transactions
- `tm.event='ValidatorSetUpdates'` - Validator changes
- `dex.swap` - DEX swap events
- `oracle.price_update` - Oracle price changes
- `compute.task_completed` - Compute task results

## Unsubscribe

```javascript
ws.send(JSON.stringify({
  jsonrpc: '2.0',
  method: 'unsubscribe',
  params: ["tm.event='NewBlock'"],
  id: 5
}));
```

## Reconnection Logic

```javascript
class ReconnectingWebSocket {
  constructor(url, maxRetries = 5) {
    this.url = url;
    this.maxRetries = maxRetries;
    this.retries = 0;
    this.connect();
  }

  connect() {
    this.ws = new WebSocket(this.url);

    this.ws.onopen = () => {
      console.log('Connected');
      this.retries = 0;
    };

    this.ws.onclose = () => {
      if (this.retries < this.maxRetries) {
        this.retries++;
        setTimeout(() => this.connect(), 1000 * this.retries);
      }
    };
  }
}
```

## See Also

- [Authentication Guide](./authentication.md)
- [Rate Limiting](./rate-limiting.md)
