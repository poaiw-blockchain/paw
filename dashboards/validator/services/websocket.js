// ValidatorWebSocket Service - Real-time updates via WebSocket

class ValidatorWebSocket {
    constructor() {
        this.ws = null;
        this.reconnectInterval = 5000;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 10;
        this.listeners = {};
        this.subscribedValidators = new Set();
        this.wsURL = window.PAW_WS_URL || 'ws://localhost:26657/websocket'; // Tendermint WebSocket endpoint
    }

    connect() {
        try {
            this.ws = new WebSocket(this.wsURL);

            this.ws.onopen = () => {
                console.log('WebSocket connected');
                this.reconnectAttempts = 0;
                this.emit('connected');
                this.subscribeToEvents();
            };

            this.ws.onmessage = (event) => {
                this.handleMessage(event);
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.emit('error', error);
            };

            this.ws.onclose = () => {
                console.log('WebSocket disconnected');
                this.emit('disconnected');
                this.attemptReconnect();
            };
        } catch (error) {
            console.error('Failed to create WebSocket:', error);
            this.emit('error', error);
            this.attemptReconnect();
        }
    }

    disconnect() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }

    attemptReconnect() {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);

            setTimeout(() => {
                this.connect();
            }, this.reconnectInterval);
        } else {
            console.error('Max reconnection attempts reached');
            this.emit('maxReconnectAttemptsReached');
        }
    }

    subscribeToEvents() {
        // Subscribe to new blocks
        this.subscribe('tm.event', 'NewBlock');

        // Subscribe to validator updates
        this.subscribe('tm.event', 'ValidatorSetUpdates');

        // Subscribe to transactions
        this.subscribe('tm.event', 'Tx');
    }

    subscribe(key, value) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            console.warn('WebSocket not connected, cannot subscribe');
            return;
        }

        const subscribeMsg = {
            jsonrpc: '2.0',
            method: 'subscribe',
            id: `subscribe_${key}_${value}`,
            params: {
                query: `${key}='${value}'`
            }
        };

        this.send(subscribeMsg);
    }

    subscribeToValidator(validatorAddress) {
        this.subscribedValidators.add(validatorAddress);

        // Subscribe to validator-specific events
        // This would be enhanced based on the blockchain's event system
        console.log(`Subscribed to validator: ${validatorAddress}`);
    }

    unsubscribeFromValidator(validatorAddress) {
        this.subscribedValidators.delete(validatorAddress);
        console.log(`Unsubscribed from validator: ${validatorAddress}`);
    }

    handleMessage(event) {
        try {
            const data = JSON.parse(event.data);

            if (data.error) {
                console.error('WebSocket message error:', data.error);
                this.emit('error', data.error);
                return;
            }

            // Handle subscription responses
            if (data.id && data.id.startsWith('subscribe_')) {
                console.log('Subscription confirmed:', data.id);
                return;
            }

            // Handle event notifications
            if (data.result && data.result.data) {
                this.handleEvent(data.result.data);
            }
        } catch (error) {
            console.error('Error parsing WebSocket message:', error);
        }
    }

    handleEvent(eventData) {
        const eventType = eventData.type;

        switch (eventType) {
            case 'tendermint/event/NewBlock':
                this.handleNewBlock(eventData.value);
                break;

            case 'tendermint/event/ValidatorSetUpdates':
                this.handleValidatorSetUpdate(eventData.value);
                break;

            case 'tendermint/event/Tx':
                this.handleTransaction(eventData.value);
                break;

            default:
                console.log('Unhandled event type:', eventType);
        }
    }

    handleNewBlock(blockData) {
        const block = {
            height: blockData.block?.header?.height,
            time: blockData.block?.header?.time,
            proposer: blockData.block?.header?.proposer_address,
            txCount: blockData.block?.data?.txs?.length || 0
        };

        this.emit('newBlock', block);

        // Check if any subscribed validators are involved
        this.subscribedValidators.forEach(validatorAddress => {
            // In production, check if validator signed this block
            this.emit('validatorUpdate', {
                address: validatorAddress,
                block: block
            });
        });
    }

    handleValidatorSetUpdate(updateData) {
        this.emit('validatorSetUpdate', updateData);

        // Check for updates to subscribed validators
        updateData.validator_updates?.forEach(update => {
            const validatorAddress = this.pubKeyToAddress(update.pub_key);
            if (this.subscribedValidators.has(validatorAddress)) {
                this.emit('validatorUpdate', {
                    address: validatorAddress,
                    votingPower: update.power
                });
            }
        });
    }

    handleTransaction(txData) {
        // Parse transaction events for validator-related activities
        const events = txData.result?.events || [];

        events.forEach(event => {
            // Check for delegation, unbonding, reward events, etc.
            if (this.isValidatorEvent(event)) {
                this.emit('validatorTransaction', {
                    type: event.type,
                    attributes: event.attributes,
                    txHash: txData.result?.hash
                });
            }
        });
    }

    isValidatorEvent(event) {
        const validatorEventTypes = [
            'delegate',
            'unbond',
            'redelegate',
            'withdraw_rewards',
            'withdraw_commission',
            'edit_validator'
        ];

        return validatorEventTypes.includes(event.type);
    }

    pubKeyToAddress(pubKey) {
        // Simplified conversion - in production, use proper cryptographic conversion
        // This would involve proper Tendermint public key to address conversion
        return 'pawvaloper' + Buffer.from(pubKey.value, 'base64').toString('hex').substring(0, 38);
    }

    send(data) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            console.warn('WebSocket not connected, cannot send message');
            return false;
        }

        try {
            this.ws.send(JSON.stringify(data));
            return true;
        } catch (error) {
            console.error('Error sending WebSocket message:', error);
            return false;
        }
    }

    on(event, callback) {
        if (!this.listeners[event]) {
            this.listeners[event] = [];
        }
        this.listeners[event].push(callback);
    }

    off(event, callback) {
        if (!this.listeners[event]) return;

        if (callback) {
            this.listeners[event] = this.listeners[event].filter(cb => cb !== callback);
        } else {
            delete this.listeners[event];
        }
    }

    emit(event, data) {
        if (!this.listeners[event]) return;

        this.listeners[event].forEach(callback => {
            try {
                callback(data);
            } catch (error) {
                console.error(`Error in event listener for ${event}:`, error);
            }
        });
    }

    getConnectionState() {
        if (!this.ws) return 'disconnected';

        switch (this.ws.readyState) {
            case WebSocket.CONNECTING:
                return 'connecting';
            case WebSocket.OPEN:
                return 'connected';
            case WebSocket.CLOSING:
                return 'closing';
            case WebSocket.CLOSED:
                return 'disconnected';
            default:
                return 'unknown';
        }
    }

    isConnected() {
        return this.ws && this.ws.readyState === WebSocket.OPEN;
    }
}

// Mock WebSocket for development/testing
class MockValidatorWebSocket extends ValidatorWebSocket {
    constructor() {
        super();
        this.mockInterval = null;
    }

    connect() {
        console.log('Using mock WebSocket connection');

        setTimeout(() => {
            this.emit('connected');
            this.startMockEvents();
        }, 100);
    }

    disconnect() {
        if (this.mockInterval) {
            clearInterval(this.mockInterval);
            this.mockInterval = null;
        }
        this.emit('disconnected');
    }

    startMockEvents() {
        // Emit mock events every few seconds
        this.mockInterval = setInterval(() => {
            // Mock new block
            if (Math.random() > 0.3) {
                this.emit('newBlock', {
                    height: Math.floor(1000000 + Math.random() * 1000),
                    time: new Date().toISOString(),
                    proposer: 'mock_proposer',
                    txCount: Math.floor(Math.random() * 50)
                });
            }

            // Mock validator update
            if (Math.random() > 0.8) {
                this.subscribedValidators.forEach(address => {
                    this.emit('validatorUpdate', {
                        address: address,
                        uptime: 0.99 + Math.random() * 0.01,
                        tokens: 1000000 + Math.random() * 100000
                    });
                });
            }
        }, 6000); // Every 6 seconds (simulating block time)
    }

    subscribeToValidator(validatorAddress) {
        this.subscribedValidators.add(validatorAddress);
        console.log(`Mock subscribed to validator: ${validatorAddress}`);
    }

    isConnected() {
        return this.mockInterval !== null;
    }

    getConnectionState() {
        return this.mockInterval ? 'connected' : 'disconnected';
    }
}

// Auto-detect and use mock WebSocket in development
function createWebSocketConnection() {
    // Check if we're in development mode or if WebSocket is unavailable
    if (typeof WebSocket === 'undefined' || window.location.protocol === 'file:') {
        console.log('Using mock WebSocket for development');
        return new MockValidatorWebSocket();
    }

    // Try real WebSocket, fall back to mock on error
    try {
        return new ValidatorWebSocket();
    } catch (error) {
        console.warn('Real WebSocket failed, using mock:', error);
        return new MockValidatorWebSocket();
    }
}

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        ValidatorWebSocket,
        MockValidatorWebSocket,
        createWebSocketConnection
    };
}
