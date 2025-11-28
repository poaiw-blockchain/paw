import React, { useState, useEffect } from 'react';
import { KeystoreService } from '../services/keystore';

const Settings = ({ onWalletReset }) => {
  const [apiEndpoint, setApiEndpoint] = useState('http://localhost:1317');
  const [wsEndpoint, setWsEndpoint] = useState('ws://localhost:26657');
  const [autoUpdate, setAutoUpdate] = useState(true);
  const [showMnemonic, setShowMnemonic] = useState(false);
  const [password, setPassword] = useState('');
  const [mnemonic, setMnemonic] = useState('');
  const [version, setVersion] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const keystoreService = new KeystoreService();

  useEffect(() => {
    loadSettings();
    loadVersion();
  }, []);

  const loadSettings = async () => {
    try {
      if (window.electron?.store) {
        const savedApi = await window.electron.store.get('apiEndpoint');
        const savedWs = await window.electron.store.get('wsEndpoint');
        const savedAutoUpdate = await window.electron.store.get('autoUpdate');

        if (savedApi) setApiEndpoint(savedApi);
        if (savedWs) setWsEndpoint(savedWs);
        if (savedAutoUpdate !== undefined) setAutoUpdate(savedAutoUpdate);
      }
    } catch (err) {
      console.error('Failed to load settings:', err);
    }
  };

  const loadVersion = async () => {
    try {
      if (window.electron?.app) {
        const ver = await window.electron.app.getVersion();
        setVersion(ver);
      }
    } catch (err) {
      console.error('Failed to load version:', err);
    }
  };

  const handleSaveSettings = async () => {
    try {
      setError('');
      setSuccess('');

      if (window.electron?.store) {
        await window.electron.store.set('apiEndpoint', apiEndpoint);
        await window.electron.store.set('wsEndpoint', wsEndpoint);
        await window.electron.store.set('autoUpdate', autoUpdate);
      }

      setSuccess('Settings saved successfully');
      setTimeout(() => setSuccess(''), 3000);
    } catch (err) {
      setError('Failed to save settings');
    }
  };

  const handleShowMnemonic = async () => {
    try {
      setError('');
      if (!password) {
        setError('Password is required');
        return;
      }

      const wallet = await keystoreService.unlockWallet(password);
      if (!wallet) {
        setError('Invalid password');
        return;
      }

      const savedMnemonic = await keystoreService.getMnemonic();
      setMnemonic(savedMnemonic);
      setShowMnemonic(true);
      setPassword('');
    } catch (err) {
      setError(err.message);
    }
  };

  const handleResetWallet = async () => {
    if (window.electron?.dialog) {
      const result = await window.electron.dialog.showMessageBox({
        type: 'warning',
        title: 'Reset Wallet',
        message: 'This will delete your current wallet. Make sure you have backed up your mnemonic phrase.',
        detail: 'This action cannot be undone.',
        buttons: ['Cancel', 'Reset Wallet'],
        defaultId: 0,
        cancelId: 0
      });

      if (result.response === 1) {
        try {
          await keystoreService.clearWallet();
          if (onWalletReset) {
            onWalletReset();
          }
        } catch (err) {
          setError('Failed to reset wallet');
        }
      }
    }
  };

  return (
    <div className="content">
      <div className="card">
        <h3 className="card-header">Network Settings</h3>

        {success && (
          <div style={{
            padding: '10px',
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
          <label className="form-label">API Endpoint</label>
          <input
            type="text"
            className="form-input"
            value={apiEndpoint}
            onChange={(e) => setApiEndpoint(e.target.value)}
            placeholder="http://localhost:1317"
          />
        </div>

        <div className="form-group">
          <label className="form-label">WebSocket Endpoint</label>
          <input
            type="text"
            className="form-input"
            value={wsEndpoint}
            onChange={(e) => setWsEndpoint(e.target.value)}
            placeholder="ws://localhost:26657"
          />
        </div>

        <div className="form-group">
          <label style={{ display: 'flex', alignItems: 'center', gap: '10px', cursor: 'pointer' }}>
            <input
              type="checkbox"
              checked={autoUpdate}
              onChange={(e) => setAutoUpdate(e.target.checked)}
            />
            <span>Enable automatic updates</span>
          </label>
        </div>

        <button className="btn btn-primary" onClick={handleSaveSettings}>
          Save Settings
        </button>
      </div>

      <div className="card">
        <h3 className="card-header">Wallet Backup</h3>

        {!showMnemonic ? (
          <>
            <p className="text-muted mb-20">
              Enter your password to view your mnemonic phrase
            </p>
            <div className="form-group">
              <label className="form-label">Password</label>
              <input
                type="password"
                className="form-input"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Enter password"
              />
            </div>
            {error && <div className="text-error mb-20">{error}</div>}
            <button className="btn btn-primary" onClick={handleShowMnemonic}>
              Show Mnemonic
            </button>
          </>
        ) : (
          <>
            <p className="text-warning mb-20">
              Keep your mnemonic phrase safe and never share it with anyone!
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
              {mnemonic}
            </div>
            <button className="btn btn-secondary" onClick={() => {
              setShowMnemonic(false);
              setMnemonic('');
            }}>
              Hide Mnemonic
            </button>
          </>
        )}
      </div>

      <div className="card">
        <h3 className="card-header">Danger Zone</h3>
        <p className="text-muted mb-20">
          Reset your wallet and delete all data. Make sure you have backed up your mnemonic phrase.
        </p>
        <button className="btn btn-danger" onClick={handleResetWallet}>
          Reset Wallet
        </button>
      </div>

      <div className="card">
        <h3 className="card-header">About</h3>
        <div style={{ display: 'grid', gap: '10px', fontSize: '14px' }}>
          <div className="flex-between">
            <span className="text-muted">Version</span>
            <span>{version || 'Unknown'}</span>
          </div>
          <div className="flex-between">
            <span className="text-muted">Platform</span>
            <span>{navigator.platform}</span>
          </div>
          <div className="flex-between">
            <span className="text-muted">User Agent</span>
            <span style={{ fontSize: '11px', fontFamily: 'monospace' }}>
              {navigator.userAgent.substring(0, 50)}...
            </span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Settings;
