import React, { useState } from 'react';
import { ApiService } from '../services/api';
import { KeystoreService } from '../services/keystore';

const Send = ({ walletData, onSuccess }) => {
  const [recipient, setRecipient] = useState('');
  const [amount, setAmount] = useState('');
  const [memo, setMemo] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [showConfirm, setShowConfirm] = useState(false);

  const apiService = new ApiService();
  const keystoreService = new KeystoreService();

  const validateForm = () => {
    if (!recipient.trim()) {
      throw new Error('Recipient address is required');
    }

    if (!recipient.startsWith('paw')) {
      throw new Error('Invalid recipient address (must start with "paw")');
    }

    if (!amount || parseFloat(amount) <= 0) {
      throw new Error('Amount must be greater than 0');
    }

    if (!password) {
      throw new Error('Password is required to sign transaction');
    }
  };

  const handlePreview = () => {
    try {
      setError('');
      validateForm();
      setShowConfirm(true);
    } catch (err) {
      setError(err.message);
    }
  };

  const handleSend = async () => {
    try {
      setLoading(true);
      setError('');
      setSuccess('');

      validateForm();

      // Verify password and get private key
      const wallet = await keystoreService.unlockWallet(password);
      if (!wallet) {
        throw new Error('Invalid password');
      }

      // Convert amount to smallest unit (upaw)
      const amountInUpaw = Math.floor(parseFloat(amount) * 1000000);

      // Send transaction
      const result = await apiService.sendTokens(
        walletData.address,
        recipient,
        amountInUpaw,
        'upaw',
        memo,
        wallet.privateKey
      );

      setSuccess(`Transaction successful! Hash: ${result.transactionHash || result.txhash}`);
      setRecipient('');
      setAmount('');
      setMemo('');
      setPassword('');
      setShowConfirm(false);

      // Refresh wallet data
      if (onSuccess) {
        setTimeout(() => onSuccess(), 2000);
      }
    } catch (err) {
      console.error('Send failed:', err);
      setError(err.message || 'Failed to send transaction');
    } finally {
      setLoading(false);
    }
  };

  if (showConfirm) {
    return (
      <div className="content">
        <div className="card" style={{ maxWidth: '600px', margin: '0 auto' }}>
          <h3 className="card-header">Confirm Transaction</h3>
          <div style={{ marginBottom: '20px' }}>
            <div style={{ padding: '15px', background: 'var(--bg-primary)', borderRadius: '6px', marginBottom: '10px' }}>
              <div style={{ marginBottom: '10px' }}>
                <span className="text-muted">From:</span>
                <div style={{ fontFamily: 'monospace', fontSize: '12px', marginTop: '5px' }}>
                  {walletData.address}
                </div>
              </div>
              <div style={{ marginBottom: '10px' }}>
                <span className="text-muted">To:</span>
                <div style={{ fontFamily: 'monospace', fontSize: '12px', marginTop: '5px' }}>
                  {recipient}
                </div>
              </div>
              <div style={{ marginBottom: '10px' }}>
                <span className="text-muted">Amount:</span>
                <div style={{ fontSize: '20px', fontWeight: '600', color: 'var(--accent)', marginTop: '5px' }}>
                  {amount} PAW
                </div>
              </div>
              {memo && (
                <div>
                  <span className="text-muted">Memo:</span>
                  <div style={{ fontSize: '12px', marginTop: '5px' }}>
                    {memo}
                  </div>
                </div>
              )}
            </div>
            <p className="text-muted" style={{ fontSize: '12px' }}>
              Please verify all details are correct before proceeding.
            </p>
          </div>
          {error && <div className="text-error mb-20">{error}</div>}
          <div className="flex gap-10">
            <button
              className="btn btn-secondary"
              onClick={() => setShowConfirm(false)}
              disabled={loading}
              style={{ flex: 1 }}
            >
              Cancel
            </button>
            <button
              className="btn btn-primary"
              onClick={handleSend}
              disabled={loading}
              style={{ flex: 1 }}
            >
              {loading ? 'Sending...' : 'Confirm & Send'}
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="content">
      <div className="card" style={{ maxWidth: '600px', margin: '0 auto' }}>
        <h3 className="card-header">Send PAW</h3>

        {success && (
          <div style={{
            padding: '15px',
            background: 'rgba(158, 206, 106, 0.1)',
            border: '1px solid var(--success)',
            borderRadius: '6px',
            marginBottom: '20px',
            color: 'var(--success)'
          }}>
            {success}
          </div>
        )}

        <div className="form-group">
          <label className="form-label">Recipient Address</label>
          <input
            type="text"
            className="form-input"
            placeholder="paw1..."
            value={recipient}
            onChange={(e) => setRecipient(e.target.value)}
          />
        </div>

        <div className="form-group">
          <label className="form-label">Amount (PAW)</label>
          <input
            type="number"
            className="form-input"
            placeholder="0.000000"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            step="0.000001"
            min="0"
          />
        </div>

        <div className="form-group">
          <label className="form-label">Memo (Optional)</label>
          <input
            type="text"
            className="form-input"
            placeholder="Transaction note"
            value={memo}
            onChange={(e) => setMemo(e.target.value)}
            maxLength={256}
          />
        </div>

        <div className="form-group">
          <label className="form-label">Password</label>
          <input
            type="password"
            className="form-input"
            placeholder="Enter your password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <div className="text-muted" style={{ fontSize: '11px', marginTop: '5px' }}>
            Required to sign the transaction
          </div>
        </div>

        {error && <div className="text-error mb-20">{error}</div>}

        <button
          className="btn btn-primary"
          onClick={handlePreview}
          disabled={loading}
          style={{ width: '100%' }}
        >
          Preview Transaction
        </button>
      </div>
    </div>
  );
};

export default Send;
