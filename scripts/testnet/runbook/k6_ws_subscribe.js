import ws from 'k6/ws';
import { check } from 'k6';

const rpc = __ENV.RPC_URL;
const wsUrl = __ENV.RPC_WS_URL || (rpc ? rpc.replace('http', 'ws') + '/websocket' : null);
if (!wsUrl) {
  throw new Error('RPC_URL or RPC_WS_URL is required');
}

const profile = __ENV.PROFILE || 'baseline';
const isPeak = profile === 'peak';

export const options = {
  vus: __ENV.VUS ? parseInt(__ENV.VUS, 10) : isPeak ? 10 : 3,
  duration: __ENV.DURATION || (isPeak ? '3m' : '1m'),
  thresholds: {
    checks: ['rate>0.99'],
  },
};

export default function () {
  const res = ws.connect(wsUrl, {}, function (socket) {
    socket.on('open', () => {
      socket.send(
        JSON.stringify({
          jsonrpc: '2.0',
          method: 'subscribe',
          id: 1,
          params: { query: "tm.event='NewBlock'" },
        })
      );
    });

    socket.on('message', () => {
      socket.close();
    });

    socket.setTimeout(() => {
      socket.close();
    }, 5000);
  });

  check(res, { 'ws connected': (r) => r && r.status === 101 });
}
