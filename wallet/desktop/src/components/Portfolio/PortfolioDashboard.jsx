import React, { useEffect, useMemo, useState } from 'react';
import { ApiService } from '../../services/api';
import './PortfolioDashboard.css';

const formatAmount = (amount) =>
  (amount / 1_000_000).toLocaleString('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });

const meterStyle = (pct) => ({
  width: `${Math.min(100, Math.max(0, pct))}%`,
  background: 'linear-gradient(90deg, var(--accent) 0%, #7ce7ac 100%)',
  height: '8px',
  borderRadius: '8px',
});

const SummaryCard = ({ label, value, sublabel }) => (
  <div className="card" style={{ minHeight: '120px' }}>
    <div className="text-muted" style={{ fontSize: '13px', marginBottom: '6px' }}>
      {label}
    </div>
    <div style={{ fontSize: '28px', fontWeight: 700 }}>{value}</div>
    {sublabel && (
      <div className="text-muted" style={{ fontSize: '12px', marginTop: '6px' }}>
        {sublabel}
      </div>
    )}
  </div>
);

const PortfolioDashboard = ({ walletData }) => {
  const api = useMemo(() => new ApiService(), []);
  const [balances, setBalances] = useState([]);
  const [delegations, setDelegations] = useState([]);
  const [rewards, setRewards] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    if (walletData?.address) {
      loadPortfolio();
    }
  }, [walletData]);

  const loadPortfolio = async () => {
    try {
      setLoading(true);
      setError(null);

      const [balanceResp, delegationsResp, rewardsResp] = await Promise.all([
        api.getBalance(walletData.address),
        api.getDelegations(walletData.address),
        api.getRewards(walletData.address),
      ]);

      setBalances(balanceResp?.balances || []);
      setDelegations(delegationsResp || []);
      const rewardTotal = rewardsResp?.total?.reduce(
        (sum, r) => sum + Number(r.amount || 0),
        0,
      );
      setRewards(rewardTotal || 0);
    } catch (err) {
      console.error('Portfolio load failed', err);
      setError(err.message || 'Failed to load portfolio');
    } finally {
      setLoading(false);
    }
  };

  const stakedTotal = useMemo(
    () =>
      delegations.reduce((sum, d) => {
        const amt = d?.balance?.amount || d?.delegation?.shares || '0';
        return sum + Number(amt);
      }, 0),
    [delegations],
  );

  const availableTotal = useMemo(
    () =>
      balances.reduce((sum, b) => (b.denom === 'upaw' ? sum + Number(b.amount || 0) : sum), 0),
    [balances],
  );

  const totalPortfolio = availableTotal + stakedTotal + rewards;
  const stakedPct = totalPortfolio ? (stakedTotal / totalPortfolio) * 100 : 0;
  const rewardsPct = totalPortfolio ? (rewards / totalPortfolio) * 100 : 0;

  const allocation = [
    { label: 'Available', amount: availableTotal, pct: totalPortfolio ? (availableTotal / totalPortfolio) * 100 : 0 },
    { label: 'Staked', amount: stakedTotal, pct: stakedPct },
    { label: 'Rewards', amount: rewards, pct: rewardsPct },
  ].filter((a) => a.amount > 0);

  if (loading) {
    return (
      <div className="content text-center">
        <div className="loading-spinner" />
        <p className="text-muted">Loading portfolio...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="card">
        <div className="text-error">{error}</div>
        <button className="btn btn-primary mt-10" onClick={loadPortfolio}>
          Retry
        </button>
      </div>
    );
  }

  return (
    <div className="content">
      <div className="grid-3">
        <SummaryCard label="Total Portfolio" value={`${formatAmount(totalPortfolio)} PAW`} />
        <SummaryCard label="Staked" value={`${formatAmount(stakedTotal)} PAW`} sublabel={`${stakedPct.toFixed(1)}% of portfolio`} />
        <SummaryCard label="Unclaimed Rewards" value={`${formatAmount(rewards)} PAW`} sublabel={`${rewardsPct.toFixed(1)}% of portfolio`} />
      </div>

      <div className="card" style={{ marginTop: '16px' }}>
        <div className="flex-between">
          <h3 className="card-header">Allocation</h3>
          <button className="btn btn-secondary" onClick={loadPortfolio}>
            Refresh
          </button>
        </div>
        {allocation.length === 0 ? (
          <div className="text-muted">No assets yet.</div>
        ) : (
          <div style={{ display: 'grid', gap: '12px', marginTop: '12px' }}>
            {allocation.map((item) => (
              <div key={item.label}>
                <div className="flex-between">
                  <div>{item.label}</div>
                  <div className="text-muted">
                    {formatAmount(item.amount)} PAW • {item.pct.toFixed(1)}%
                  </div>
                </div>
                <div style={{ background: 'var(--bg-primary)', borderRadius: '8px', padding: '6px' }}>
                  <div style={meterStyle(item.pct)} />
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="card" style={{ marginTop: '16px' }}>
        <h3 className="card-header">Positions</h3>
        <div className="table">
          <div className="table-header">
            <div>Type</div>
            <div>Amount (PAW)</div>
            <div>Details</div>
          </div>
          <div className="table-body">
            {balances.map((b, idx) => (
              <div className="table-row" key={`bal-${idx}`}>
                <div>Available</div>
                <div>{formatAmount(Number(b.amount || 0))}</div>
                <div>{b.denom}</div>
              </div>
            ))}
            {delegations.map((d, idx) => (
              <div className="table-row" key={`del-${idx}`}>
                <div>Delegated</div>
                <div>{formatAmount(Number(d?.balance?.amount || d?.delegation?.shares || 0))}</div>
                <div>{d?.delegation?.validator_address || 'validator'}</div>
              </div>
            ))}
            {rewards > 0 && (
              <div className="table-row">
                <div>Rewards</div>
                <div>{formatAmount(rewards)}</div>
                <div>Claimable</div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default PortfolioDashboard;
