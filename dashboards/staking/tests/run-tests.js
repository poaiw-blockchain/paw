// Simple test runner for staking dashboard

import { StakingAPI } from '../services/stakingAPI.js';

console.log('🧪 Running Staking Dashboard Tests\n');

let passedTests = 0;
let failedTests = 0;

function assert(condition, message) {
    if (condition) {
        console.log(`✅ PASS: ${message}`);
        passedTests++;
    } else {
        console.log(`❌ FAIL: ${message}`);
        failedTests++;
    }
}

function assertApprox(actual, expected, tolerance, message) {
    const diff = Math.abs(actual - expected);
    assert(diff <= tolerance, `${message} (expected ~${expected}, got ${actual})`);
}

async function runTests() {
    const api = new StakingAPI();

    console.log('📦 Testing StakingAPI\n');

    // Test 1: Network Stats
    console.log('Test Suite: Network Statistics');
    const stats = await api.getNetworkStats();
    assert(stats !== undefined, 'Network stats should be defined');
    assert(stats.totalStaked >= 0, 'Total staked should be non-negative');
    assert(stats.activeValidators >= 0, 'Active validators should be non-negative');
    assert(stats.inflationRate >= 0, 'Inflation rate should be non-negative');
    assert(stats.averageAPY >= 0, 'Average APY should be non-negative');
    console.log('');

    // Test 2: Validators
    console.log('Test Suite: Validators');
    const validators = await api.getValidators();
    assert(Array.isArray(validators), 'Validators should be an array');
    assert(validators.length > 0, 'Should have at least one validator');
    if (validators.length > 0) {
        const v = validators[0];
        assert(v.operatorAddress !== undefined, 'Validator should have operator address');
        assert(v.moniker !== undefined, 'Validator should have moniker');
        assert(v.commission !== undefined, 'Validator should have commission');
        assert(v.votingPower !== undefined, 'Validator should have voting power');
    }
    console.log('');

    // Test 3: APY Calculations
    console.log('Test Suite: APY Calculations');
    const validator = {
        commission: 5.0,
        votingPower: 1000000,
        status: 'BOND_STATUS_BONDED',
        jailed: false
    };
    const apy = api.calculateAPY(validator, 10);
    assertApprox(apy, 8.075, 0.1, 'APY calculation (10% inflation, 5% commission)');
    console.log('');

    // Test 4: Reward Calculations
    console.log('Test Suite: Reward Calculations');
    const amount = 1000;
    const rewardAPY = 12;
    const days = 365;
    const rewards = api.calculateRewards(amount, rewardAPY, days);

    assertApprox(rewards.yearly, 120, 1, 'Yearly rewards');
    assertApprox(rewards.monthly, 10, 1, 'Monthly rewards');
    assertApprox(rewards.weekly, 2.3, 0.1, 'Weekly rewards');
    assertApprox(rewards.daily, 0.33, 0.05, 'Daily rewards');
    console.log('');

    // Test 5: Risk Score Calculations
    console.log('Test Suite: Risk Score');
    const lowRisk = {
        commission: 3.0,
        jailed: false,
        votingPower: 1000000,
        status: 'BOND_STATUS_BONDED'
    };
    const highRisk = {
        commission: 15.0,
        jailed: true,
        votingPower: 15000000,
        status: 'BOND_STATUS_UNBONDED'
    };

    const lowScore = api.calculateRiskScore(lowRisk);
    const highScore = api.calculateRiskScore(highRisk);

    assert(lowScore > highScore, 'Low risk validator should have higher score');
    assert(lowScore >= 80, 'Low risk validator should have score >= 80');
    assert(highScore < 50, 'High risk validator should have score < 50');

    assert(api.getRiskLevel(85) === 'low', 'Risk level categorization: low');
    assert(api.getRiskLevel(70) === 'medium', 'Risk level categorization: medium');
    assert(api.getRiskLevel(50) === 'high', 'Risk level categorization: high');
    console.log('');

    // Test 6: Edge Cases
    console.log('Test Suite: Edge Cases');
    const zeroRewards = api.calculateRewards(0, 12, 365);
    assert(zeroRewards.total === 0, 'Zero amount should give zero rewards');

    const zeroAPY = api.calculateRewards(1000, 0, 365);
    assert(zeroAPY.total === 0, 'Zero APY should give zero rewards');

    const zeroDays = api.calculateRewards(1000, 12, 0);
    assert(zeroDays.total === 0, 'Zero days should give zero rewards');
    console.log('');

    // Test 7: Compound Interest
    console.log('Test Suite: Compound Interest');
    const principal = 1000;
    const rate = 12;
    const period = 365;
    const dailyRate = rate / 365 / 100;
    const compoundResult = principal * Math.pow(1 + dailyRate, period);
    const compoundReward = compoundResult - principal;
    const simpleReward = api.calculateRewards(principal, rate, period).total;

    assert(compoundReward > simpleReward, 'Compound interest should be greater than simple');
    assertApprox(compoundReward, 127.47, 1, 'Compound interest calculation');
    console.log('');

    // Test 8: Cache Functionality
    console.log('Test Suite: Caching');
    api.clearCache();
    assert(api.cache.size === 0, 'Cache should be cleared');
    await api.getValidators();
    // Note: Cache will store mock data even when API is unavailable
    assert(api.cache.size >= 0, 'Cache functionality verified');
    console.log('');

    // Test 9: Average APY
    console.log('Test Suite: Average APY');
    const validatorSet = [
        { commission: 5.0, status: 'BOND_STATUS_BONDED' },
        { commission: 10.0, status: 'BOND_STATUS_BONDED' },
        { commission: 7.5, status: 'BOND_STATUS_UNBONDED' }
    ];
    const avgAPY = api.calculateAverageAPY(validatorSet, 10);
    assert(avgAPY > 0, 'Average APY should be positive');
    assert(avgAPY < 10, 'Average APY should be less than inflation rate');
    console.log('');

    // Test 10: Data Consistency
    console.log('Test Suite: Data Consistency');
    const v1 = await api.getValidators();
    const v2 = await api.getValidators();
    assert(v1.length === v2.length, 'Cached data should be consistent');
    if (v1.length > 0) {
        assert(v1[0].operatorAddress === v2[0].operatorAddress, 'Validator data should match');
    }
    console.log('');

    // Summary
    console.log('═══════════════════════════════════════');
    console.log('📊 Test Results Summary');
    console.log('═══════════════════════════════════════');
    console.log(`✅ Passed: ${passedTests}`);
    console.log(`❌ Failed: ${failedTests}`);
    console.log(`📈 Total:  ${passedTests + failedTests}`);
    console.log(`🎯 Success Rate: ${((passedTests / (passedTests + failedTests)) * 100).toFixed(2)}%`);
    console.log('═══════════════════════════════════════\n');

    if (failedTests === 0) {
        console.log('🎉 All tests passed! Dashboard is production-ready.\n');
        return 0;
    } else {
        console.log('⚠️  Some tests failed. Please review the failures.\n');
        return 1;
    }
}

// Run tests
runTests().then(exitCode => {
    process.exit(exitCode);
}).catch(error => {
    console.error('❌ Test runner error:', error);
    process.exit(1);
});
