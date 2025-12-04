import React, { useState, useEffect } from 'react';
import { ApiService } from '../services/api';

const Wallet = ({ walletData, onRefresh }) => {
  const [balance, setBalance] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [refreshing, setRefreshing] = useState(false);

  const apiService = new ApiService();

  useEffect(() => {
    if (walletData?.address) {
      fetchBalance();
    }
  }, [walletData]);

  const fetchBalance = async () => {
    try {
      setLoading(true);
      setError(null);
      const balanceData = await apiService.getBalance(walletData.address);
      setBalance(balanceData);
    } catch (err) {
      console.error('Failed to fetch balance:', err);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleRefresh = async () => {
    setRefreshing(true);
    await fetchBalance();
    if (onRefresh) {
      await onRefresh();
    }
    setRefreshing(false);
  };

  const formatAmount = (amount, denom) => {
    if (!amount) return '0';
    // Convert from smallest unit (upaw) to PAW
    const value = parseInt(amount) / 1000000;
    return value.toLocaleString('en-US', { minimumFractionDigits: 6, maximumFractionDigits: 6 });
  };

  if (loading) {
    return (
      <div className="content text-center">
        <div className="loading-spinner"></div>
        <p className="text-muted">Loading balance...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="content">
        <div className="card">
          <div className="text-error text-center">
            <p>Failed to load balance</p>
            <p className="text-muted mt-20">{error}</p>
            <button className="btn btn-primary mt-20" onClick={fetchBalance}>
              Retry
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="content">
      <div className="card">
        <div className="flex-between mb-20">
          <h3 className="card-header" style={{ marginBottom: 0 }}>Balance</h3>
          <button
            className="btn btn-secondary"
            onClick={handleRefresh}
            disabled={refreshing}
          >
            {refreshing ? 'Refreshing...' : 'Refresh'}
          </button>
        </div>

        {balance && balance.balances && balance.balances.length > 0 ? (
          <div>
            {balance.balances.map((coin, index) => (
              <div key={index} style={{ marginBottom: '15px' }}>
                <div style={{ fontSize: '32px', fontWeight: '600', color: 'var(--accent)' }}>
                  {formatAmount(coin.amount, coin.denom)} {coin.denom.toUpperCase()}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center text-muted">
            <p>No balance found</p>
            <p style={{ fontSize: '12px', marginTop: '10px' }}>
              Your wallet is empty. Get some PAW tokens to get started.
            </p>
          </div>
        )}
      </div>

      <div className="card">
        <h3 className="card-header">Wallet Information</h3>
        <div style={{ display: 'grid', gap: '15px' }}>
          <div>
            <div className="text-muted" style={{ fontSize: '12px', marginBottom: '5px' }}>
              Address
            </div>
            <div style={{
              fontFamily: 'monospace',
              fontSize: '13px',
              wordBreak: 'break-all',
              background: 'var(--bg-primary)',
              padding: '10px',
              borderRadius: '4px'
            }}>
              {walletData.address}
            </div>
          </div>

          {walletData.publicKey && (
            <div>
              <div className="text-muted" style={{ fontSize: '12px', marginBottom: '5px' }}>
                Public Key
              </div>
              <div style={{
                fontFamily: 'monospace',
                fontSize: '13px',
                wordBreak: 'break-all',
                background: 'var(--bg-primary)',
                padding: '10px',
                borderRadius: '4px'
              }}>
                {walletData.publicKey}
              </div>
            </div>
          )}
        </div>
      </div>

      <div className="card">
        <h3 className="card-header">Quick Actions</h3>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: '10px' }}>
          <button className="btn btn-primary">Send Tokens</button>
          <button className="btn btn-secondary">View History</button>
          <button className="btn btn-secondary">Backup Wallet</button>
        </div>
      </div>
    </div>
  );
};

export default Wallet;
