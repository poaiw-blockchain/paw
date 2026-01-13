import { createSchema, createYoga } from 'graphql-yoga';
import { createServer } from 'node:http';
import fetch from 'node-fetch';

const RPC_URL = process.env.RPC_URL || 'http://127.0.0.1:26657';
const REST_URL = process.env.REST_URL || 'http://127.0.0.1:1317';

async function fetchJson(url) {
    const response = await fetch(url);
    if (!response.ok) {
        throw new Error(`Request failed: ${response.status}`);
    }
    return response.json();
}

const schema = createSchema({
    typeDefs: `
        type Status {
            chainId: String
            latestHeight: Int
            latestTime: String
            catchingUp: Boolean
        }

        type NodeInfo {
            defaultNodeId: String
            network: String
            version: String
        }

        type Query {
            status: Status!
            nodeInfo: NodeInfo!
        }
    `,
    resolvers: {
        Query: {
            status: async () => {
                const data = await fetchJson(`${RPC_URL}/status`);
                const info = data.result || {};
                const syncInfo = info.sync_info || {};
                return {
                    chainId: info.node_info?.network || null,
                    latestHeight: Number(syncInfo.latest_block_height || 0),
                    latestTime: syncInfo.latest_block_time || null,
                    catchingUp: Boolean(syncInfo.catching_up),
                };
            },
            nodeInfo: async () => {
                const data = await fetchJson(`${REST_URL}/cosmos/base/tendermint/v1beta1/node_info`);
                const info = data.default_node_info || {};
                return {
                    defaultNodeId: info.default_node_id || null,
                    network: info.network || null,
                    version: info.version || null,
                };
            },
        },
    },
});

const yoga = createYoga({ schema, graphiql: true });
const server = createServer(yoga);
const port = Number(process.env.PORT || 11100);
server.listen(port, () => {
    console.log(`GraphQL gateway listening on :${port}`);
});
