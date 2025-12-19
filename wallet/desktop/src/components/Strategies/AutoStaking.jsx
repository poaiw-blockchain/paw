import React, { useEffect, useMemo, useRef, useState } from 'react';
import Decimal from 'decimal.js';
import { StakingService } from '../../services/staking';

const STRATEGIES = [
  {
    id: 'conservative',
    name: 'Conservative Auto-Compound',
    description: 'Claim rewards daily and redelegate to top-performing validators.',
    cadence: 'Daily',
    risk: 'Low',
    targetAPR: '9-11%',
  },
  {
    id: 'balanced',
    name: 'Balanced Rotation',
    description: 'Rotate rewards weekly across diversified validator set; auto-undelegate from underperformers.',
    cadence: 'Weekly',
    risk: 'Medium',
    targetAPR: '11-13%',
  },
  {
    id: 'aggressive',
    name: 'Aggressive Yield Hunt',
    description: 'Daily compounding with dynamic validator scoring; higher churn tolerance.',
    cadence: 'Daily',
    risk: 'High',
    targetAPR: '13-15%',
  },
];

const AutoStaking = ({ walletData }) => {
  const stakingService = useMemo(() => new StakingService(), []);
  const isLedger = walletData?.type === 'ledger';
  const [selected, setSelected] = useState('balanced');
  const [paused, setPaused] = useState(false);
  const [automationEnabled, setAutomationEnabled] = useState(false);
  const [running, setRunning] = useState(false);
  const [nextRunTs, setNextRunTs] = useState(null);
  const [password, setPassword] = useState('');
  const [minRewards, setMinRewards] = useState('0.25');
  const [statusLog, setStatusLog] = useState([]);
  const timerRef = useRef(null);

  const STRATEGY_RULES = {
    conservative: { cadenceMs: 24 * 60 * 60 * 1000, fanout: 2 },
    balanced: { cadenceMs: 6 * 60 * 60 * 1000, fanout: 3 },
    aggressive: { cadenceMs: 3 * 60 * 60 * 1000, fanout: 4 },
  };

  const appendLog = (message) => {
    setStatusLog((log) => [{ ts: Date.now(), message }, ...log].slice(0, 4));
  };

  const clearTimer = () => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  };

  const scheduleNext = (delayMs) => {
    clearTimer();
    const ts = Date.now() + delayMs;
    setNextRunTs(ts);
    timerRef.current = setTimeout(() => runStrategy('scheduled'), delayMs);
  };

  useEffect(() => () => clearTimer(), []);

  useEffect(() => {
    if (!automationEnabled || paused) {
      clearTimer();
      return;
    }
    scheduleNext(STRATEGY_RULES[selected]?.cadenceMs || 6 * 60 * 60 * 1000);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [automationEnabled, paused, selected]);

  const formatNextRun = () => {
    if (!automationEnabled) return 'Disabled';
    if (!nextRunTs) return 'Scheduling...';
    const diff = nextRunTs - Date.now();
    if (diff <= 0) return 'Running shortly';
    const mins = Math.round(diff / 60000);
    if (mins >= 60) {
      const hours = (mins / 60).toFixed(1);
      return `in ~${hours}h`;
    }
    return `in ${mins}m`;
  };

  const selectTargets = (validators, fanout) => {
    const available = validators
      .filter((v) => !v.jailed && (v.status || '').toLowerCase().includes('bond'))
      .sort((a, b) => b.aprEstimate - a.aprEstimate);
    return available.slice(0, Math.max(1, fanout || 2));
  };

  const runStrategy = async (trigger) => {
    if (paused) {
      appendLog('Automation paused; skipping cycle');
      return;
    }
    if (running) {
      appendLog('Cycle already running; skipping duplicate trigger');
      return;
    }
    if (!walletData?.address) {
      appendLog('Connect a wallet to run automation');
      return;
    }
    if (isLedger) {
      appendLog('Auto-staking is unavailable for Ledger wallets (requires automated signing).');
      return;
    }
    if (!password || password.length < 8) {
      appendLog('Enter wallet password (min 8 chars) to sign auto-stake txs');
      return;
    }

    setRunning(true);
    try {
      appendLog(`Starting ${selected} cycle (${trigger})`);
      const portfolio = await stakingService.getPortfolio(walletData.address);
      const rewards = portfolio.delegations.filter(
        (d) => new Decimal(d.rewardsBase || '0').gt(0)
      );
      const totalRewardsBase = rewards.reduce(
        (acc, r) => acc.plus(new Decimal(r.rewardsBase || '0')),
        new Decimal(0)
      );
      const totalRewardsDisplay = totalRewardsBase.dividedBy(1_000_000);
      let threshold = new Decimal(0);
      try {
        threshold = new Decimal(minRewards || '0');
      } catch {
        threshold = new Decimal(0);
      }
      if (totalRewardsDisplay.lt(threshold)) {
        appendLog(
          `Skip: rewards ${totalRewardsDisplay.toFixed(4)} below threshold ${threshold.toFixed(2)}`
        );
        return;
      }

      const { fanout } = STRATEGY_RULES[selected] || { fanout: 2 };
      const targets = selectTargets(portfolio.validators, fanout);
      if (!targets.length) {
        appendLog('No active validators available for compounding');
        return;
      }

      for (const reward of rewards) {
        await stakingService.withdraw({
          validatorAddress: reward.validatorAddress,
          password,
          memo: `auto-withdraw:${selected}`,
        });
      }

      const perTarget = totalRewardsDisplay.dividedBy(targets.length);
      if (perTarget.lte(0)) {
        appendLog('Skip: reward amount too small after fanout');
        return;
      }

      for (const target of targets) {
        await stakingService.delegate({
          validatorAddress: target.validatorAddress,
          amount: perTarget.toFixed(6),
          password,
          memo: `auto-compound:${selected}`,
        });
      }

      appendLog(
        `Compounded ${totalRewardsDisplay.toFixed(4)} ${portfolio.summary.symbol} across ${
          targets.length
        } validators`
      );
    } catch (e) {
      appendLog(`Auto-stake failed: ${e.message || 'Unknown error'}`);
    } finally {
      setRunning(false);
      if (automationEnabled && !paused) {
        scheduleNext(STRATEGY_RULES[selected]?.cadenceMs || 6 * 60 * 60 * 1000);
      }
    }
  };

  const handleApply = () => {
    if (!walletData?.address) {
      appendLog('Wallet not initialized; complete setup first');
      return;
    }
    if (isLedger) {
      appendLog('Auto-staking is unavailable for Ledger wallets (requires automated signing).');
      return;
    }
    if (!password || password.length < 8) {
      appendLog('Enter wallet password (min 8 chars) to start automation');
      return;
    }
    setAutomationEnabled(true);
    setPaused(false);
    runStrategy('manual');
  };

  return (
    <div className="content">
      <div className="card">
        <div className="flex-between">
          <h3 className="card-header">Automated Staking Strategies</h3>
          <div className="flex-center" style={{ gap: '8px' }}>
            <span className="text-muted">Status:</span>
            <span className={paused ? 'text-error' : 'text-success'}>
              {paused ? 'Paused' : 'Active'}
            </span>
            <button className="btn btn-secondary" onClick={() => setPaused((p) => !p)}>
              {paused ? 'Resume' : 'Pause'}
            </button>
          </div>
        </div>
        <p className="text-muted" style={{ marginTop: '6px' }}>
          Configure auto-compounding and validator rotation rules. Strategies run locally; keep the app online or schedule via your automation runner.
        </p>
        <div className="grid-3" style={{ gap: '10px', marginTop: '10px' }}>
          <div>
            <label className="text-muted">Wallet Password</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Required to sign automation txs"
            />
          </div>
          <div>
            <label className="text-muted">Min Rewards to Compound (PAW)</label>
            <input
              type="number"
              min="0"
              step="0.01"
              value={minRewards}
              onChange={(e) => setMinRewards(e.target.value)}
            />
          </div>
          <div>
            <label className="text-muted">Next Run</label>
            <div style={{ marginTop: '8px' }}>{formatNextRun()}</div>
          </div>
        </div>
      </div>

      <div className="grid-3" style={{ marginTop: '12px' }}>
        {STRATEGIES.map((s) => (
          <div
            key={s.id}
            className={`card ${selected === s.id ? 'card-selected' : ''}`}
            style={{ cursor: 'pointer' }}
            onClick={() => setSelected(s.id)}
          >
            <div className="flex-between">
              <h4 style={{ margin: 0 }}>{s.name}</h4>
              <span className="badge">{s.risk}</span>
            </div>
            <div className="text-muted" style={{ fontSize: '12px', marginTop: '4px' }}>
              {s.description}
            </div>
            <div className="text-muted" style={{ marginTop: '10px', fontSize: '13px' }}>
              Cadence: {s.cadence} · Target APR: {s.targetAPR}
            </div>
          </div>
        ))}
      </div>

      <div className="card" style={{ marginTop: '16px' }}>
        <h4 className="card-header">Execution Plan</h4>
        <ul style={{ lineHeight: 1.7, paddingLeft: '18px' }}>
          <li>Claim distribution rewards at the configured cadence.</li>
          <li>Evaluate validator performance (uptime, commission, slashing history).</li>
          <li>Redelegate rewards to the top-scoring set (exclude jailed/low uptime).</li>
          <li>Track gas spend; skip auto-actions if fee > 1% of rewards.</li>
        </ul>
        <div className="flex-between mt-10">
          <div className="text-muted">Next run: {formatNextRun()}</div>
          <div className="flex-center" style={{ gap: '10px' }}>
            <button className="btn btn-secondary" onClick={() => runStrategy('manual')}>
              Run once now
            </button>
            <button className="btn btn-primary" onClick={handleApply} disabled={running}>
              {running ? 'Executing...' : automationEnabled ? 'Reschedule' : 'Apply Strategy'}
            </button>
          </div>
        </div>
        {walletData?.address && (
          <div className="text-muted mt-6" style={{ fontSize: '12px' }}>
            Delegator: <span className="text-mono">{walletData.address}</span>
          </div>
        )}
        {statusLog.length > 0 && (
          <div className="mt-10">
            <div className="text-muted" style={{ fontSize: '12px', marginBottom: '4px' }}>
              Recent cycles
            </div>
            <ul style={{ paddingLeft: '18px', lineHeight: 1.6 }}>
              {statusLog.map((log) => (
                <li key={log.ts}>{log.message}</li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </div>
  );
};

export default AutoStaking;
