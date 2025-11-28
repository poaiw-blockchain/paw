import React, { useState } from 'react';
import { KeystoreService } from '../services/keystore';

const Setup = ({ onWalletCreated }) => {
  const [mode, setMode] = useState('create');
  const [mnemonic, setMnemonic] = useState('');
  const [generatedMnemonic, setGeneratedMnemonic] = useState('');
  const [confirmed, setConfirmed] = useState(false);
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const keystoreService = new KeystoreService();

  const handleCreateWallet = async () => {
    try {
      setLoading(true);
      setError('');

      if (password !== confirmPassword) {
        throw new Error('Passwords do not match');
      }

      if (password.length < 8) {
        throw new Error('Password must be at least 8 characters');
      }

      const newMnemonic = await keystoreService.generateMnemonic();
      setGeneratedMnemonic(newMnemonic);
      setMode('confirm');
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleConfirmWallet = async () => {
    try {
      setLoading(true);
      setError('');

      if (!confirmed) {
        throw new Error('Please confirm you have written down your mnemonic');
      }

      const wallet = await keystoreService.createWallet(generatedMnemonic, password);
      onWalletCreated(wallet);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleImportWallet = async () => {
    try {
      setLoading(true);
      setError('');

      if (password !== confirmPassword) {
        throw new Error('Passwords do not match');
      }

      if (password.length < 8) {
        throw new Error('Password must be at least 8 characters');
      }

      if (!mnemonic.trim()) {
        throw new Error('Please enter your mnemonic phrase');
      }

      const wallet = await keystoreService.createWallet(mnemonic.trim(), password);
      onWalletCreated(wallet);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  if (mode === 'confirm') {
    return (
      <div className="content">
        <div className="card" style={{ maxWidth: '600px', margin: '0 auto' }}>
          <h2 className="card-header">Backup Your Mnemonic</h2>
          <p className="text-muted mb-20">
            Write down these 24 words in order and store them safely. You will need them to recover your wallet.
          </p>
          <div style={{
            background: 'var(--bg-primary)',
            padding: '20px',
            borderRadius: '6px',
            marginBottom: '20px',
            fontFamily: 'monospace',
            fontSize: '14px',
            lineHeight: '1.8',
            userSelect: 'all'
          }}>
            {generatedMnemonic}
          </div>
          <div className="form-group">
            <label style={{ display: 'flex', alignItems: 'center', gap: '10px', cursor: 'pointer' }}>
              <input
                type="checkbox"
                checked={confirmed}
                onChange={(e) => setConfirmed(e.target.checked)}
              />
              <span>I have written down my mnemonic phrase</span>
            </label>
          </div>
          {error && <div className="text-error mb-20">{error}</div>}
          <div className="flex gap-10">
            <button
              className="btn btn-secondary"
              onClick={() => setMode('create')}
              disabled={loading}
            >
              Back
            </button>
            <button
              className="btn btn-primary"
              onClick={handleConfirmWallet}
              disabled={!confirmed || loading}
            >
              {loading ? 'Creating...' : 'Create Wallet'}
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="content">
      <div className="card" style={{ maxWidth: '500px', margin: '0 auto' }}>
        <h2 className="card-header">Welcome to PAW Wallet</h2>
        <p className="text-muted mb-20">
          Create a new wallet or import an existing one using your mnemonic phrase.
        </p>

        <div className="flex gap-10 mb-20">
          <button
            className={`btn ${mode === 'create' ? 'btn-primary' : 'btn-secondary'}`}
            onClick={() => setMode('create')}
            style={{ flex: 1 }}
          >
            Create New Wallet
          </button>
          <button
            className={`btn ${mode === 'import' ? 'btn-primary' : 'btn-secondary'}`}
            onClick={() => setMode('import')}
            style={{ flex: 1 }}
          >
            Import Wallet
          </button>
        </div>

        {mode === 'create' ? (
          <>
            <div className="form-group">
              <label className="form-label">Password</label>
              <input
                type="password"
                className="form-input"
                placeholder="Enter password (min 8 characters)"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label className="form-label">Confirm Password</label>
              <input
                type="password"
                className="form-input"
                placeholder="Confirm password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
              />
            </div>
            {error && <div className="text-error mb-20">{error}</div>}
            <button
              className="btn btn-primary"
              onClick={handleCreateWallet}
              disabled={loading || !password || !confirmPassword}
              style={{ width: '100%' }}
            >
              {loading ? 'Creating...' : 'Generate Mnemonic'}
            </button>
          </>
        ) : (
          <>
            <div className="form-group">
              <label className="form-label">Mnemonic Phrase</label>
              <textarea
                className="form-textarea"
                placeholder="Enter your 24-word mnemonic phrase"
                value={mnemonic}
                onChange={(e) => setMnemonic(e.target.value)}
                rows={4}
              />
            </div>
            <div className="form-group">
              <label className="form-label">Password</label>
              <input
                type="password"
                className="form-input"
                placeholder="Enter password (min 8 characters)"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>
            <div className="form-group">
              <label className="form-label">Confirm Password</label>
              <input
                type="password"
                className="form-input"
                placeholder="Confirm password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
              />
            </div>
            {error && <div className="text-error mb-20">{error}</div>}
            <button
              className="btn btn-primary"
              onClick={handleImportWallet}
              disabled={loading || !mnemonic || !password || !confirmPassword}
              style={{ width: '100%' }}
            >
              {loading ? 'Importing...' : 'Import Wallet'}
            </button>
          </>
        )}
      </div>
    </div>
  );
};

export default Setup;
