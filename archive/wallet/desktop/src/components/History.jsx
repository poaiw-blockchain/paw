import React, { useState, useEffect } from 'react';
import { ApiService } from '../services/api';

const History = ({ walletData }) => {
  const [transactions, setTransactions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const apiService = new ApiService();

  useEffect(() => {
    if (walletData?.address) {
      fetchTransactions();
    }
  }, [walletData]);

  const fetchTransactions = async () => {
    try {
      setLoading(true);
      setError(null);
      const txs = await apiService.getTransactions(walletData.address);
      setTransactions(txs || []);
    } catch (err) {
      console.error('Failed to fetch transactions:', err);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (timestamp) => {
    if (!timestamp) return 'N/A';
    const date = new Date(timestamp);
    return date.toLocaleString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  const formatAmount = (amount, denom) => {
    if (!amount) return '0';
    const value = parseInt(amount) / 1000000;
    return `${value.toFixed(6)} ${(denom || 'upaw').toUpperCase()}`;
  };

  const getTransactionType = (tx) => {
    if (!tx.messages || tx.messages.length === 0) return 'Unknown';
    const msgType = tx.messages[0]['@type'] || tx.messages[0].type;

    if (msgType.includes('MsgSend')) return 'Send';
    if (msgType.includes('MsgDelegate')) return 'Delegate';
    if (msgType.includes('MsgUndelegate')) return 'Undelegate';
    if (msgType.includes('MsgWithdrawDelegatorReward')) return 'Claim Rewards';

    return msgType.split('.').pop() || 'Unknown';
  };

  const getStatusClass = (code) => {
    if (code === 0) return 'status-success';
    if (code === undefined || code === null) return 'status-pending';
    return 'status-failed';
  };

  const getStatusText = (code) => {
    if (code === 0) return 'Success';
    if (code === undefined || code === null) return 'Pending';
    return 'Failed';
  };

  if (loading) {
    return (
      <div className="content text-center">
        <div className="loading-spinner"></div>
        <p className="text-muted">Loading transactions...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="content">
        <div className="card">
          <div className="text-error text-center">
            <p>Failed to load transactions</p>
            <p className="text-muted mt-20">{error}</p>
            <button className="btn btn-primary mt-20" onClick={fetchTransactions}>
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
          <h3 className="card-header" style={{ marginBottom: 0 }}>Transaction History</h3>
          <button className="btn btn-secondary" onClick={fetchTransactions}>
            Refresh
          </button>
        </div>

        {transactions.length > 0 ? (
          <div style={{ overflowX: 'auto' }}>
            <table className="table">
              <thead>
                <tr>
                  <th>Type</th>
                  <th>Hash</th>
                  <th>Amount</th>
                  <th>Status</th>
                  <th>Date</th>
                </tr>
              </thead>
              <tbody>
                {transactions.map((tx, index) => (
                  <tr key={tx.txhash || tx.hash || index}>
                    <td>{getTransactionType(tx)}</td>
                    <td>
                      <span style={{ fontFamily: 'monospace', fontSize: '12px' }}>
                        {(tx.txhash || tx.hash || '').substring(0, 16)}...
                      </span>
                    </td>
                    <td>
                      {tx.messages && tx.messages[0]?.amount?.length > 0
                        ? formatAmount(tx.messages[0].amount[0].amount, tx.messages[0].amount[0].denom)
                        : '-'}
                    </td>
                    <td>
                      <span className={`status-badge ${getStatusClass(tx.code)}`}>
                        {getStatusText(tx.code)}
                      </span>
                    </td>
                    <td>{formatDate(tx.timestamp)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="text-center text-muted">
            <p>No transactions found</p>
            <p style={{ fontSize: '12px', marginTop: '10px' }}>
              Your transaction history will appear here
            </p>
          </div>
        )}
      </div>
    </div>
  );
};

export default History;
