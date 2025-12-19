import React, { useEffect, useMemo, useState } from 'react';
import { ApiService } from '../../services/api';
import BridgeService from '../../services/bridge';
import LedgerService from '../../services/ledger';

const DEFAULT_CHANNELS = {
  'Cosmos Hub': 'channel-0',
  Osmosis: 'channel-1',
  Neutron: 'channel-2',
  PAW: 'channel-0',
};

const BridgeCenter = ({ walletData }) => {
  const api = useMemo(() => new ApiService(), []);
  const bridgeService = useMemo(() => new BridgeService(), []);
  const ledgerService = useMemo(() => new LedgerService(), []);
  const isLedger = walletData?.type === 'ledger';
  const [form, setForm] = useState({
    sourceChain: 'PAW',
    destChain: 'Cosmos Hub',
    token: 'upaw',
    amount: '',
    destAddress: '',
    password: '',
    sourceChannel: DEFAULT_CHANNELS['Cosmos Hub'],
    sourcePort: 'transfer',
    memo: '',
    timeoutSeconds: 900,
  });
  const [status, setStatus] = useState(null);
  const [txResult, setTxResult] = useState(null);
  const [submitting, setSubmitting] = useState(false);

  const updateField = (field, value) => setForm((f) => ({ ...f, [field]: value }));

  useEffect(() => {
    const nextChannel = DEFAULT_CHANNELS[form.destChain];
    if (nextChannel && form.sourceChannel !== nextChannel) {
      setForm((f) => ({ ...f, sourceChannel: nextChannel }));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [form.destChain]);

  const validate = (requirePassword = true) => {
    if (!form.amount || Number(form.amount) <= 0) return 'Amount must be greater than zero';
    if (!form.destAddress) return 'Destination address is required';
    if (form.sourceChain === form.destChain) return 'Source and destination must differ';
    if (form.destChain === 'PAW' && !form.destAddress.startsWith('paw')) {
      return 'Destination must be a PAW bech32 address';
    }
    if (!isLedger && requirePassword && (!form.password || form.password.length < 8)) {
      return 'Wallet password required to sign';
    }
    return null;
  };

  const simulateQuote = async () => {
    setTxResult(null);
    const err = validate(false);
    if (err) {
      setStatus({ type: 'error', message: err });
      return;
    }
    setStatus({ type: 'info', message: 'Fetching path and fee estimate...' });
    try {
      const endpoint = await api.getEndpoint();
      const rpc = endpoint.includes('1317')
        ? endpoint.replace('1317', '26657').replace(/\/cosmos.*/, '')
        : endpoint;
      setStatus({
        type: 'success',
        message: `Ready: ${form.amount} ${form.token} from ${form.sourceChain} → ${form.destChain} via ${form.sourcePort}/${form.sourceChannel}. RPC: ${rpc}`,
      });
    } catch (e) {
      setStatus({ type: 'error', message: e.message || 'Failed to fetch endpoint' });
    }
  };

  const submitBridge = async () => {
    setTxResult(null);
    const err = validate();
    if (err) {
      setStatus({ type: 'error', message: err });
      return;
    }
    setSubmitting(true);
    try {
      let signer;
      let fromAddress = walletData?.address;
      if (isLedger) {
        signer = await ledgerService.getSigner();
      }

      const res = await bridgeService.bridgeTokens({
        password: form.password,
        amount: form.amount,
        denom: form.token,
        destAddress: form.destAddress,
        sourceChannel: form.sourceChannel,
        sourcePort: form.sourcePort,
        memo: form.memo,
        timeoutSeconds: Number(form.timeoutSeconds) || 900,
        offlineSigner: signer,
        fromAddress,
      });
      setTxResult(res);
      setStatus({
        type: 'success',
        message: `Bridge broadcasted. Tx hash: ${res?.transactionHash || res?.txhash || 'pending'}`,
      });
    } catch (e) {
      setStatus({ type: 'error', message: e.message || 'Bridge submission failed' });
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="content">
      <div className="grid-2">
        <div className="card">
          <h3 className="card-header">Bridge Tokens</h3>
          <div className="form-group">
            <label>From</label>
            <select value={form.sourceChain} onChange={(e) => updateField('sourceChain', e.target.value)}>
              <option>PAW</option>
              <option>Cosmos Hub</option>
              <option>Osmosis</option>
              <option>Neutron</option>
            </select>
          </div>
          <div className="form-group">
            <label>To</label>
            <select value={form.destChain} onChange={(e) => updateField('destChain', e.target.value)}>
              <option>Cosmos Hub</option>
              <option>Osmosis</option>
              <option>Neutron</option>
              <option>PAW</option>
            </select>
          </div>
          <div className="form-group">
            <label>Token</label>
            <select value={form.token} onChange={(e) => updateField('token', e.target.value)}>
              <option value="upaw">PAW (upaw)</option>
              <option value="uusdc">USDC (uusdc)</option>
            </select>
          </div>
          <div className="form-group">
            <label>Amount</label>
            <input
              type="number"
              min="0"
              step="0.0001"
              value={form.amount}
              onChange={(e) => updateField('amount', e.target.value)}
              placeholder="e.g. 10"
            />
          </div>
          <div className="form-group">
            <label>Destination Address</label>
            <input
              type="text"
              value={form.destAddress}
              onChange={(e) => updateField('destAddress', e.target.value)}
              placeholder="bech32 address on destination chain"
            />
          </div>
          <div className="form-group grid-2" style={{ gap: '10px' }}>
            <div>
              <label>IBC Port</label>
              <input
                type="text"
                value={form.sourcePort}
                onChange={(e) => updateField('sourcePort', e.target.value)}
                placeholder="transfer"
              />
            </div>
            <div>
              <label>IBC Channel</label>
              <input
                type="text"
                value={form.sourceChannel}
                onChange={(e) => updateField('sourceChannel', e.target.value)}
                placeholder="channel-0"
              />
            </div>
          </div>
          <div className="form-group grid-2" style={{ gap: '10px' }}>
            <div>
              <label>Timeout (seconds)</label>
              <input
                type="number"
                min="60"
                value={form.timeoutSeconds}
                onChange={(e) => updateField('timeoutSeconds', e.target.value)}
              />
            </div>
            <div>
              <label>Memo (optional)</label>
              <input
                type="text"
                value={form.memo}
                onChange={(e) => updateField('memo', e.target.value)}
                placeholder="IBC memo"
              />
            </div>
          </div>
          {!isLedger && (
            <div className="form-group">
              <label>Wallet Password</label>
              <input
                type="password"
                value={form.password}
                onChange={(e) => updateField('password', e.target.value)}
                placeholder="Required to sign bridge tx"
              />
            </div>
          )}
          {isLedger && (
            <div className="text-muted" style={{ fontSize: '12px', marginBottom: '10px' }}>
              Ledger connected: verify on-device when prompted. No password required.
            </div>
          )}
          <div className="flex-between">
            <button className="btn btn-secondary" onClick={simulateQuote}>Estimate Route</button>
            <button className="btn btn-primary" onClick={submitBridge} disabled={submitting}>
              {submitting ? 'Submitting...' : 'Initiate Bridge'}
            </button>
          </div>
          {status && (
            <div className={`mt-10 ${status.type === 'error' ? 'text-error' : 'text-muted'}`}>
              {status.message}
            </div>
          )}
          {txResult?.transactionHash && (
            <div className="mt-6 text-muted text-mono" style={{ fontSize: '12px' }}>
              Hash: {txResult.transactionHash}
            </div>
          )}
        </div>

        <div className="card">
          <h3 className="card-header">Bridge Checklist</h3>
          <ul style={{ lineHeight: '1.6', paddingLeft: '18px' }}>
            <li>Confirm destination address belongs to the target chain (bech32 prefix).</li>
            <li>Ensure relayer is online and fee account funded.</li>
            <li>Keep app open until the IBC packet is relayed.</li>
            <li>For large transfers, test with a small amount first.</li>
            <li>Cross-check final balance on the destination chain.</li>
          </ul>
          {walletData?.address && (
            <div className="mt-10 text-muted" style={{ fontSize: '12px' }}>
              Source account: <span className="text-mono">{walletData.address}</span>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default BridgeCenter;
