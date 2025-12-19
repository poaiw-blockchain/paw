import React, { useState, useEffect } from 'react';
import { makeMultisigThresholdPubkey, pubkeyToAddress } from '@cosmjs/amino';
import { KeystoreService } from '../services/keystore';
import LedgerService from '../services/ledger';

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
  const [cosigners, setCosigners] = useState([
    { name: 'Signer 1', pubkey: '', transport: 'hardware', path: "m/44'/118'/0'/0/0" },
    { name: 'Signer 2', pubkey: '', transport: 'hardware', path: "m/44'/118'/1'/0/0" },
  ]);
  const [threshold, setThreshold] = useState(2);
  const [multisigResult, setMultisigResult] = useState(null);
  const [multisigError, setMultisigError] = useState('');

  const keystoreService = new KeystoreService();
  const ledgerService = new LedgerService();

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
          await ledgerService.clearSavedWallet();
          if (onWalletReset) {
            onWalletReset();
          }
        } catch (err) {
          setError('Failed to reset wallet');
        }
      }
    }
  };

  const handleCosignerChange = (index, field, value) => {
    const next = [...cosigners];
    next[index] = { ...next[index], [field]: value };
    setCosigners(next);
  };

  const addCosigner = () => {
    setCosigners([
      ...cosigners,
      { name: `Signer ${cosigners.length + 1}`, pubkey: '', transport: 'hardware', path: "m/44'/118'/0'/0/0" },
    ]);
  };

  const removeCosigner = (index) => {
    if (cosigners.length <= 2) return;
    setCosigners(cosigners.filter((_, i) => i !== index));
  };

  const generateMultisigBundle = () => {
    try {
      setMultisigError('');
      if (threshold < 1 || threshold > cosigners.length) {
        throw new Error('Threshold must be between 1 and the number of cosigners');
      }
      const prefix = 'paw';
      const pubkeys = cosigners.map((c) => {
        if (!c.pubkey?.trim()) {
          throw new Error('All cosigners must include a base64 secp256k1 pubkey');
        }
        return { type: 'tendermint/PubKeySecp256k1', value: c.pubkey.trim() };
      });
      const multisig = makeMultisigThresholdPubkey(pubkeys, threshold);
      const address = pubkeyToAddress(multisig, prefix);
      const bundle = {
        threshold,
        prefix,
        cosigners,
        address,
        note: 'Standard PAW multisig bundle; share with cosigners for recovery or signing.',
      };
      setMultisigResult(JSON.stringify(bundle, null, 2));
    } catch (err) {
      setMultisigError(err.message);
      setMultisigResult(null);
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
        <h3 className="card-header">Multisig / Recovery</h3>
        <p className="text-muted mb-10">
          Build a hardware-first multisig bundle (2-5 signers). Provide secp256k1 pubkeys (base64), optional transport/path hints, and choose a threshold.
        </p>
        <div className="form-group">
          <label className="form-label">Threshold</label>
          <input
            type="number"
            className="form-input"
            value={threshold}
            min={1}
            max={cosigners.length}
            onChange={(e) => setThreshold(Number(e.target.value))}
          />
        </div>
        {cosigners.map((cosigner, idx) => (
          <div key={idx} className="form-group" style={{ border: '1px solid var(--border)', padding: '10px', borderRadius: '6px', marginBottom: '10px' }}>
            <div className="flex-between" style={{ gap: '10px', alignItems: 'center' }}>
              <input
                type="text"
                className="form-input"
                placeholder="Cosigner label"
                value={cosigner.name}
                onChange={(e) => handleCosignerChange(idx, 'name', e.target.value)}
              />
              {cosigners.length > 2 && (
                <button className="btn btn-secondary" onClick={() => removeCosigner(idx)} style={{ minWidth: '80px' }}>
                  Remove
                </button>
              )}
            </div>
            <input
              type="text"
              className="form-input"
              placeholder="Pubkey (base64 secp256k1)"
              value={cosigner.pubkey}
              onChange={(e) => handleCosignerChange(idx, 'pubkey', e.target.value)}
              style={{ marginTop: '8px' }}
            />
            <div className="flex-between" style={{ gap: '10px', marginTop: '8px' }}>
              <select
                className="form-input"
                value={cosigner.transport}
                onChange={(e) => handleCosignerChange(idx, 'transport', e.target.value)}
              >
                <option value="hardware">Hardware (Ledger/Trezor)</option>
                <option value="software">Software</option>
              </select>
              <input
                type="text"
                className="form-input"
                value={cosigner.path}
                onChange={(e) => handleCosignerChange(idx, 'path', e.target.value)}
                placeholder="Derivation path m/44'/118'/0'/0/0"
              />
            </div>
          </div>
        ))}
        <div className="flex-between" style={{ marginBottom: '10px' }}>
          <button className="btn btn-secondary" onClick={addCosigner}>
            Add Cosigner
          </button>
          <button className="btn btn-primary" onClick={generateMultisigBundle}>
            Generate Bundle
          </button>
        </div>
        {multisigError && <div className="text-error mb-10">{multisigError}</div>}
        {multisigResult && (
          <textarea
            className="form-textarea"
            style={{ fontFamily: 'monospace', height: '160px' }}
            readOnly
            value={multisigResult}
          />
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
