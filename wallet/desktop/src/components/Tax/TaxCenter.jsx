import React, { useEffect, useMemo, useState } from 'react';
import { ApiService } from '../../services/api';

const toCSV = (rows) => {
  const header = ['hash', 'type', 'amount', 'denom', 'fee', 'timestamp'];
  const body = rows
    .map((r) => header.map((h) => `"${(r[h] ?? '').toString().replace(/"/g, '""')}"`).join(','))
    .join('\n');
  return `${header.join(',')}\n${body}`;
};

const inferType = (tx, address) => {
  try {
    const msgs = tx?.body?.messages || tx?.tx?.body?.messages || [];
    if (!msgs.length) return 'other';
    const m = msgs[0];
    if (m['@type']?.includes('MsgSend')) {
      return m.from_address === address ? 'send' : 'receive';
    }
    if (m['@type']?.includes('MsgDelegate')) return 'delegate';
    if (m['@type']?.includes('MsgUndelegate')) return 'undelegate';
    if (m['@type']?.includes('MsgWithdrawDelegatorReward')) return 'reward';
  } catch (e) {
    return 'other';
  }
  return 'other';
};

const extractAmount = (tx) => {
  const msgs = tx?.body?.messages || tx?.tx?.body?.messages || [];
  const coin =
    msgs[0]?.amount ||
    msgs[0]?.token ||
    msgs[0]?.amounts?.[0] ||
    msgs[0]?.tokens?.[0] ||
    msgs[0]?.outputs?.[0]?.coins?.[0];
  if (!coin) return { amount: '0', denom: 'upaw' };
  return { amount: coin.amount || coin, denom: coin.denom || 'upaw' };
};

const TaxCenter = ({ walletData }) => {
  const api = useMemo(() => new ApiService(), []);
  const [rows, setRows] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [exporting, setExporting] = useState(false);

  useEffect(() => {
    if (walletData?.address) {
      loadTxs();
    }
  }, [walletData]);

  const loadTxs = async () => {
    try {
      setLoading(true);
      setError(null);
      const txs = await api.getTransactions(walletData.address, 200);
      const formatted = (txs || []).map((tx) => {
        const hash = tx?.txhash || tx?.hash || 'unknown';
        const fee =
          tx?.auth_info?.fee?.amount?.[0]?.amount ||
          tx?.tx?.auth_info?.fee?.amount?.[0]?.amount ||
          '0';
        const ts =
          tx?.timestamp ||
          tx?.body?.memo ||
          tx?.tx_response?.timestamp ||
          tx?.tx_response?.logs?.timestamp ||
          '';
        const type = inferType(tx, walletData.address);
        const amtInfo = extractAmount(tx);
        return {
          hash,
          type,
          amount: amtInfo.amount,
          denom: amtInfo.denom,
          fee,
          timestamp: ts,
        };
      });
      setRows(formatted);
    } catch (err) {
      setError(err.message || 'Failed to load transactions');
    } finally {
      setLoading(false);
    }
  };

  const exportCSV = () => {
    setExporting(true);
    try {
      const csv = toCSV(rows);
      const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', `paw-tax-${walletData.address}.csv`);
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    } finally {
      setExporting(false);
    }
  };

  if (loading) {
    return (
      <div className="content text-center">
        <div className="loading-spinner" />
        <p className="text-muted">Aggregating transactions...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="card">
        <div className="text-error">{error}</div>
        <button className="btn btn-primary mt-10" onClick={loadTxs}>
          Retry
        </button>
      </div>
    );
  }

  const income = rows.filter((r) => ['receive', 'reward'].includes(r.type));
  const expenses = rows.filter((r) => ['send', 'delegate', 'undelegate'].includes(r.type));
  const incomeTotal = income.reduce((sum, r) => sum + Number(r.amount || 0), 0);
  const expenseTotal = expenses.reduce((sum, r) => sum + Number(r.amount || 0), 0);
  const feesTotal = rows.reduce((sum, r) => sum + Number(r.fee || 0), 0);

  return (
    <div className="content">
      <div className="grid-3">
        <div className="card">
          <div className="text-muted">Income</div>
          <div style={{ fontSize: '26px', fontWeight: 700 }}>{(incomeTotal / 1_000_000).toFixed(3)} PAW</div>
        </div>
        <div className="card">
          <div className="text-muted">Expenses (incl. staking)</div>
          <div style={{ fontSize: '26px', fontWeight: 700 }}>{(expenseTotal / 1_000_000).toFixed(3)} PAW</div>
        </div>
        <div className="card">
          <div className="text-muted">Fees Paid</div>
          <div style={{ fontSize: '26px', fontWeight: 700 }}>{(feesTotal / 1_000_000).toFixed(6)} PAW</div>
        </div>
      </div>

      <div className="flex-between" style={{ marginTop: '16px' }}>
        <h3>Tax-ready CSV</h3>
        <button className="btn btn-primary" onClick={exportCSV} disabled={exporting || rows.length === 0}>
          {exporting ? 'Preparing...' : 'Export CSV'}
        </button>
      </div>

      <div className="table" style={{ marginTop: '12px' }}>
        <div className="table-header">
          <div>Hash</div>
          <div>Type</div>
          <div>Amount</div>
          <div>Fee</div>
          <div>Time</div>
        </div>
        <div className="table-body">
          {rows.slice(0, 100).map((r) => (
            <div className="table-row" key={r.hash}>
              <div className="text-mono" style={{ maxWidth: '240px' }}>{r.hash}</div>
              <div>{r.type}</div>
              <div>{(Number(r.amount || 0) / 1_000_000).toFixed(3)} {r.denom}</div>
              <div>{(Number(r.fee || 0) / 1_000_000).toFixed(6)} upaw</div>
              <div className="text-muted" style={{ fontSize: '12px' }}>{r.timestamp || 'n/a'}</div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default TaxCenter;
