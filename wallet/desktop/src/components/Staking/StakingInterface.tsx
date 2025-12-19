import React, { useEffect, useMemo, useState } from 'react';
import {
  StakingPortfolio,
  StakingService,
  ValidatorMetrics,
} from '../../services/staking';
import LedgerService from '../../services/ledger';

interface WalletData {
  address: string;
  type?: string;
}

type ActionType = 'delegate' | 'undelegate' | 'redelegate' | 'withdraw';

interface Props {
  walletData: WalletData | null;
  service?: StakingService;
}

const DEFAULT_FORM = {
  validator: '',
  dstValidator: '',
  amount: '',
  password: '',
  memo: '',
};

const StakingInterface: React.FC<Props> = ({ walletData, service }) => {
  const stakingService = useMemo(() => service ?? new StakingService(), [service]);
  const ledgerService = useMemo(() => new LedgerService(), []);
  const [portfolio, setPortfolio] = useState<StakingPortfolio | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [action, setAction] = useState<ActionType>('delegate');
  const [form, setForm] = useState(DEFAULT_FORM);
  const [statusMessage, setStatusMessage] = useState('');
  const [error, setError] = useState('');
  const [lastTxHash, setLastTxHash] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const isLedger = walletData?.type === 'ledger';

  useEffect(() => {
    if (walletData?.address) {
      refreshPortfolio();
    }
  }, [walletData?.address, stakingService]);

  useEffect(() => {
    if (portfolio) {
      const defaultValidator = portfolio.validators[0]?.validatorAddress || '';
      const defaultDelegate = getDelegatedValidators(portfolio)[0]?.validatorAddress || '';
      setForm((prev) => ({
        ...prev,
        validator: prev.validator || defaultValidator || defaultDelegate,
        dstValidator: prev.dstValidator || defaultDelegate,
      }));
    }
  }, [portfolio]);

  const refreshPortfolio = async () => {
    if (!walletData?.address) {
      return;
    }

    setIsLoading(true);
    setError('');
    try {
      const data = await stakingService.getPortfolio(walletData.address);
      setPortfolio(data);
    } catch (err: any) {
      setError(err?.message || 'Failed to load staking data');
    } finally {
      setIsLoading(false);
    }
  };

  const handleAction = async () => {
    if (!portfolio) {
      return;
    }
    setIsSubmitting(true);
    setStatusMessage('');
    setError('');
    try {
      let result;
      let offlineSigner;
      if (isLedger) {
        offlineSigner = await ledgerService.getSigner();
      }
      if (action === 'delegate') {
        result = await stakingService.delegate({
          validatorAddress: form.validator,
          amount: form.amount,
          password: isLedger ? undefined : form.password,
          offlineSigner,
          fromAddress: walletData?.address,
          memo: form.memo.trim() || undefined,
        });
      } else if (action === 'undelegate') {
        result = await stakingService.undelegate({
          validatorAddress: form.validator,
          amount: form.amount,
          password: isLedger ? undefined : form.password,
          offlineSigner,
          fromAddress: walletData?.address,
          memo: form.memo.trim() || undefined,
        });
      } else if (action === 'redelegate') {
        result = await stakingService.redelegate({
          srcValidatorAddress: form.validator,
          dstValidatorAddress: form.dstValidator,
          amount: form.amount,
          password: isLedger ? undefined : form.password,
          offlineSigner,
          fromAddress: walletData?.address,
          memo: form.memo.trim() || undefined,
        });
      } else {
        result = await stakingService.withdraw({
          validatorAddress: form.validator,
          password: isLedger ? undefined : form.password,
          offlineSigner,
          fromAddress: walletData?.address,
          memo: form.memo.trim() || undefined,
        });
      }

      setStatusMessage(`Transaction broadcast successfully (tx ${result.transactionHash})`);
      setLastTxHash(result.transactionHash);
      setForm((prev) => ({
        ...prev,
        amount: action === 'withdraw' ? prev.amount : '',
        password: '',
        memo: '',
      }));
      await refreshPortfolio();
    } catch (err: any) {
      setError(err?.message || 'Failed to execute staking transaction');
    } finally {
      setIsSubmitting(false);
    }
  };

  const validators = portfolio?.validators || [];
  const delegatedValidators = useMemo(
    () => getDelegatedValidators(portfolio),
    [portfolio]
  );
  const rewardPositions = useMemo(
    () => getRewardingValidators(portfolio),
    [portfolio]
  );

  const actionValidators = (() => {
    switch (action) {
      case 'delegate':
        return validators;
      case 'redelegate':
        return delegatedValidators;
      default:
        return delegatedValidators;
    }
  })();

  const canSubmit = useMemo(() => {
    if (!isLedger && (!form.password || form.password.length < 8)) {
      return false;
    }
    if (action === 'withdraw') {
      return Boolean(form.validator);
    }
    if (!form.amount || Number(form.amount) <= 0) {
      return false;
    }
    if (!form.validator) {
      return false;
    }
    if (action === 'redelegate' && !form.dstValidator) {
      return false;
    }
    return true;
  }, [action, form]);

  const renderValidatorRow = (validator: ValidatorMetrics) => (
    <tr key={validator.validatorAddress}>
      <td>
        <div className="staking-validator-name">
          <strong>{validator.moniker}</strong>
          <small>{validator.validatorAddress}</small>
        </div>
      </td>
      <td>{validator.votingPowerPercent.toFixed(2)}%</td>
      <td>{validator.aprEstimate.toFixed(2)}%</td>
      <td>{validator.commissionFormatted}</td>
      <td>{validator.myDelegationDisplay}</td>
      <td>
        <span
          className={`status-badge ${
            validator.jailed
              ? 'status-failed'
              : validator.status === 'BOND_STATUS_BONDED'
              ? 'status-success'
              : 'status-pending'
          }`}
        >
          {validator.jailed ? 'Jailed' : formatStatus(validator.status)}
        </span>
      </td>
    </tr>
  );

  return (
    <div className="content staking-content">
      {isLoading && (
        <div className="loading-overlay">
          <div className="loading-spinner" />
        </div>
      )}

      {portfolio && (
        <>
          <div className="staking-summary-grid">
            <div className="staking-card">
              <p className="staking-card-label">Total Delegated</p>
              <h3>{portfolio.summary.totalDelegatedDisplay}</h3>
            </div>
            <div className="staking-card">
              <p className="staking-card-label">Unclaimed Rewards</p>
              <h3>{portfolio.summary.totalRewardsDisplay}</h3>
            </div>
            <div className="staking-card">
              <p className="staking-card-label">Active Validators</p>
              <h3>{portfolio.summary.activeValidators}</h3>
              <small>{`Network APR ~${portfolio.summary.averageApr.toFixed(2)}%`}</small>
            </div>
            <div className="staking-card">
              <p className="staking-card-label">Last Sync</p>
              <h3>{new Date(portfolio.updatedAt).toLocaleTimeString()}</h3>
            </div>
          </div>

          <div className="staking-grid">
            <section className="staking-panel">
              <header className="staking-panel-header">
                <div>
                  <h3>Validator Set</h3>
                  <p className="text-muted">
                    Monitor commission, voting power, and personal exposure.
                  </p>
                </div>
                <button className="btn btn-secondary" onClick={refreshPortfolio}>
                  Refresh
                </button>
              </header>
              <div className="staking-table-wrapper">
                <table className="table staking-table">
                  <thead>
                    <tr>
                      <th>Validator</th>
                      <th>Voting Power</th>
                      <th>APR</th>
                      <th>Commission</th>
                      <th>Your Stake</th>
                      <th>Status</th>
                    </tr>
                  </thead>
                  <tbody>{validators.map(renderValidatorRow)}</tbody>
                </table>
              </div>
            </section>

            <section className="staking-panel staking-action-panel">
              <header className="staking-panel-header">
                <h3>Staking Actions</h3>
                <p className="text-muted">
                  Delegate, redelegate, or withdraw rewards with built-in safety checks.
                </p>
              </header>
              <div className="form-group">
                <label className="form-label">Action</label>
                <select
                  className="form-input"
                  value={action}
                  onChange={(e) => setAction(e.target.value as ActionType)}
                >
                  <option value="delegate">Delegate</option>
                  <option value="undelegate">Undelegate</option>
                  <option value="redelegate">Redelegate</option>
                  <option value="withdraw">Withdraw Rewards</option>
                </select>
              </div>

              <div className="form-group">
                <label className="form-label">Validator</label>
                <select
                  className="form-input"
                  value={form.validator}
                  onChange={(e) => setForm((prev) => ({ ...prev, validator: e.target.value }))}
                >
                  <option value="">Select validator</option>
                  {actionValidators.map((validator) => (
                    <option key={validator.validatorAddress} value={validator.validatorAddress}>
                      {validator.moniker}
                    </option>
                  ))}
                </select>
              </div>

              {action === 'redelegate' && (
                <div className="form-group">
                  <label className="form-label">Destination Validator</label>
                  <select
                    className="form-input"
                    value={form.dstValidator}
                    onChange={(e) => setForm((prev) => ({ ...prev, dstValidator: e.target.value }))}
                  >
                    <option value="">Select destination</option>
                    {validators
                      .filter((validator) => validator.validatorAddress !== form.validator)
                      .map((validator) => (
                        <option key={validator.validatorAddress} value={validator.validatorAddress}>
                          {validator.moniker}
                        </option>
                      ))}
                  </select>
                </div>
              )}

              {action !== 'withdraw' && (
                <div className="form-group">
                  <label className="form-label">Amount ({portfolio.summary.symbol})</label>
                  <input
                    type="number"
                    min="0"
                    className="form-input"
                    value={form.amount}
                    onChange={(e) => setForm((prev) => ({ ...prev, amount: e.target.value }))}
                    placeholder="0.0"
                  />
                </div>
              )}

              <div className="form-group">
                <label className="form-label">Wallet Authentication</label>
                {!isLedger ? (
                  <input
                    type="password"
                    className="form-input"
                    value={form.password}
                    onChange={(e) => setForm((prev) => ({ ...prev, password: e.target.value }))}
                    placeholder="Decrypt keystore to sign"
                  />
                ) : (
                  <div className="text-muted" style={{ fontSize: '12px' }}>
                    Ledger connected: confirm staking transaction on your device. No password required.
                  </div>
                )}
              </div>

              <div className="form-group">
                <label className="form-label">Memo (optional)</label>
                <input
                  className="form-input"
                  value={form.memo}
                  maxLength={128}
                  onChange={(e) => setForm((prev) => ({ ...prev, memo: e.target.value }))}
                />
              </div>

              {statusMessage && <div className="form-success">{statusMessage}</div>}
              {error && <div className="form-error">{error}</div>}
              {lastTxHash && (
                <div className="staking-tx">
                  Latest tx hash: <code>{lastTxHash}</code>
                </div>
              )}

              <button
                type="button"
                className="btn btn-primary staking-submit"
                disabled={!canSubmit || isSubmitting}
                onClick={handleAction}
              >
                {isSubmitting ? 'Submitting…' : 'Execute'}
              </button>
            </section>
          </div>

          <div className="staking-grid">
            <section className="staking-panel">
              <header className="staking-panel-header">
                <h3>Delegations</h3>
                <p className="text-muted">Track current stake allocations.</p>
              </header>
              <div className="staking-table-wrapper">
                <table className="table staking-table">
                  <thead>
                    <tr>
                      <th>Validator</th>
                      <th>Delegated</th>
                      <th>Rewards</th>
                      <th>Status</th>
                      <th>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {delegatedValidators.map((delegation) => (
                      <tr key={delegation.validatorAddress}>
                        <td>
                          <div className="staking-validator-name">
                            <strong>{delegation.validatorMoniker}</strong>
                            <small>{delegation.validatorAddress}</small>
                          </div>
                        </td>
                        <td>{delegation.amountDisplay}</td>
                        <td>{delegation.rewardsDisplay}</td>
                        <td>{formatStatus(delegation.status)}</td>
                        <td>
                          <div className="staking-row-actions">
                            <button
                              className="btn btn-secondary"
                              onClick={() => {
                                setAction('undelegate');
                                setForm((prev) => ({ ...prev, validator: delegation.validatorAddress }));
                              }}
                            >
                              Undelegate
                            </button>
                            <button
                              className="btn btn-secondary"
                              onClick={() => {
                                setAction('withdraw');
                                setForm((prev) => ({ ...prev, validator: delegation.validatorAddress }));
                              }}
                            >
                              Withdraw
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>

            <section className="staking-panel">
              <header className="staking-panel-header">
                <h3>Reward Opportunities</h3>
                <p className="text-muted">
                  Validators accruing rewards above dust thresholds.
                </p>
              </header>
              <div className="staking-table-wrapper">
                <table className="table staking-table">
                  <thead>
                    <tr>
                      <th>Validator</th>
                      <th>Pending Rewards</th>
                      <th>Quick Action</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rewardPositions.map((position) => (
                      <tr key={position.validatorAddress}>
                        <td>
                          <div className="staking-validator-name">
                            <strong>{position.validatorMoniker}</strong>
                            <small>{position.validatorAddress}</small>
                          </div>
                        </td>
                        <td>{position.rewardsDisplay}</td>
                        <td>
                          <button
                            className="btn btn-primary"
                            onClick={() => {
                              setAction('withdraw');
                              setForm((prev) => ({ ...prev, validator: position.validatorAddress }));
                            }}
                          >
                            Prepare Withdraw
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          </div>
        </>
      )}
    </div>
  );
};

function getDelegatedValidators(portfolio: StakingPortfolio | null) {
  if (!portfolio) {
    return [];
  }
  return portfolio.delegations.filter((delegation) => {
    const numeric = parseFloat(delegation.amountBase || '0');
    return Number.isFinite(numeric) && numeric > 0;
  });
}

function getRewardingValidators(portfolio: StakingPortfolio | null) {
  if (!portfolio) {
    return [];
  }
  return portfolio.rewards.filter((delegation) => {
    const numeric = parseFloat(delegation.rewardsBase || '0');
    return Number.isFinite(numeric) && numeric > 0;
  });
}

function formatStatus(status: string) {
  switch (status) {
    case 'BOND_STATUS_BONDED':
    case 'bonded':
      return 'Bonded';
    case 'BOND_STATUS_UNBONDING':
      return 'Unbonding';
    case 'BOND_STATUS_UNBONDED':
      return 'Unbonded';
    default:
      return status || 'Unknown';
  }
}

export default StakingInterface;
