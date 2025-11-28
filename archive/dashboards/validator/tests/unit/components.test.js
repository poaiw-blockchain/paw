// Unit tests for Dashboard Components

const ValidatorCard = require('../../components/ValidatorCard');
const DelegationList = require('../../components/DelegationList');
const RewardsChart = require('../../components/RewardsChart');
const UptimeMonitor = require('../../components/UptimeMonitor');

describe('ValidatorCard', () => {
    const mockValidatorData = {
        address: 'pawvaloper1test',
        moniker: 'Test Validator',
        website: 'https://test.com',
        details: 'Test validator details',
        identity: '',
        tokens: '1000000000000',
        delegatorShares: '1000000000000',
        commission: {
            rate: 0.05,
            maxRate: 0.20,
            maxChangeRate: 0.01
        },
        status: 'BOND_STATUS_BONDED',
        jailed: false
    };

    it('should render validator card with correct data', () => {
        const card = new ValidatorCard(mockValidatorData);
        const html = card.render();

        expect(html).toContain('Test Validator');
        expect(html).toContain('pawvaloper1test');
        expect(html).toContain('https://test.com');
    });

    it('should show jailed warning when validator is jailed', () => {
        const jailedData = { ...mockValidatorData, jailed: true };
        const card = new ValidatorCard(jailedData);
        const html = card.render();

        expect(html).toContain('Jailed');
        expect(html).toContain('jailed-warning');
    });

    it('should format commission rates correctly', () => {
        const card = new ValidatorCard(mockValidatorData);

        expect(card.formatCommission(0.05)).toBe('5.00%');
        expect(card.formatCommission(0.1234)).toBe('12.34%');
        expect(card.formatCommission(0)).toBe('0.00%');
    });

    it('should format tokens correctly', () => {
        const card = new ValidatorCard(mockValidatorData);

        expect(card.formatTokens('1000000')).toContain('PAW');
        expect(card.formatTokens(null)).toBe('0 PAW');
    });

    it('should escape HTML in user-provided content', () => {
        const maliciousData = {
            ...mockValidatorData,
            moniker: '<script>alert("xss")</script>',
            details: '<img src=x onerror=alert(1)>'
        };

        const card = new ValidatorCard(maliciousData);
        const html = card.render();

        expect(html).not.toContain('<script>');
        expect(html).not.toContain('onerror=');
    });
});

describe('DelegationList', () => {
    const mockDelegations = [
        {
            delegatorAddress: 'paw1delegator1',
            shares: '5000000000',
            pendingRewards: '10000',
            timestamp: '2024-01-01T00:00:00Z'
        },
        {
            delegatorAddress: 'paw1delegator2',
            shares: '3000000000',
            pendingRewards: '6000',
            timestamp: '2024-01-02T00:00:00Z'
        }
    ];

    it('should render delegation list with correct data', () => {
        const list = new DelegationList(mockDelegations);
        const html = list.render();

        expect(html).toContain('paw1delegator1');
        expect(html).toContain('paw1delegator2');
        expect(html).toContain('Total Delegations');
    });

    it('should show empty state when no delegations', () => {
        const list = new DelegationList([]);
        const html = list.render();

        expect(html).toContain('No delegations found');
    });

    it('should calculate total amount correctly', () => {
        const list = new DelegationList(mockDelegations);
        const total = list.formatTotalAmount();

        expect(total).toContain('PAW');
    });

    it('should sort delegations by amount', () => {
        const list = new DelegationList(mockDelegations);
        list.sortBy = 'amount';
        list.sortOrder = 'desc';

        const sorted = list.sortDelegations(mockDelegations);

        expect(parseFloat(sorted[0].shares)).toBeGreaterThanOrEqual(
            parseFloat(sorted[1].shares)
        );
    });

    it('should sort delegations by date', () => {
        const list = new DelegationList(mockDelegations);
        list.sortBy = 'timestamp';
        list.sortOrder = 'desc';

        const sorted = list.sortDelegations(mockDelegations);

        expect(new Date(sorted[0].timestamp).getTime()).toBeGreaterThanOrEqual(
            new Date(sorted[1].timestamp).getTime()
        );
    });

    it('should filter delegations by search term', () => {
        const list = new DelegationList(mockDelegations);
        list.filter('delegator1');

        expect(list.filteredDelegations).toHaveLength(1);
        expect(list.filteredDelegations[0].delegatorAddress).toBe('paw1delegator1');
    });
});

describe('RewardsChart', () => {
    const mockRewardsData = Array.from({ length: 30 }, (_, i) => ({
        timestamp: new Date(Date.now() - i * 24 * 60 * 60 * 1000).toISOString(),
        amount: 10000 + Math.random() * 5000,
        commission: 500 + Math.random() * 250
    }));

    it('should process rewards data correctly', () => {
        const chart = new RewardsChart(mockRewardsData);
        const processed = chart.processData();

        expect(processed.total).toBeGreaterThan(0);
        expect(processed.averageDaily).toBeGreaterThan(0);
        expect(processed.dataPoints).toHaveLength(30);
    });

    it('should filter data by timeframe', () => {
        const chart = new RewardsChart(mockRewardsData);
        chart.timeframe = '7d';

        const filtered = chart.filterDataByTimeframe();

        expect(filtered.length).toBeLessThanOrEqual(7);
    });

    it('should calculate trend correctly', () => {
        const increasingData = Array.from({ length: 30 }, (_, i) => ({
            timestamp: new Date(Date.now() - (29 - i) * 24 * 60 * 60 * 1000).toISOString(),
            amount: 10000 + i * 100
        }));

        const chart = new RewardsChart(increasingData);
        const trend = chart.calculateTrend(increasingData);

        expect(trend).toBeGreaterThan(0); // Positive trend for increasing data
    });

    it('should handle empty rewards data', () => {
        const chart = new RewardsChart([]);
        const processed = chart.processData();

        expect(processed.total).toBe(0);
        expect(processed.averageDaily).toBe(0);
    });

    it('should format amounts correctly', () => {
        const chart = new RewardsChart([]);

        expect(chart.formatAmount(1234567)).toContain('1.23M');
        expect(chart.formatAmount(12345)).toContain('12.35K');
        expect(chart.formatAmount(123)).toContain('123.00');
    });
});

describe('UptimeMonitor', () => {
    const mockUptimeData = {
        uptimePercentage: 99.5,
        totalBlocks: 10000,
        missedBlocks: 50,
        blocks: Array.from({ length: 100 }, (_, i) => ({
            height: 1000000 - i,
            signed: Math.random() > 0.02,
            proposed: Math.random() > 0.99
        })),
        uptime7d: 99.5,
        uptime30d: 99.2,
        consecutiveMisses: 2,
        longestStreak: 850,
        alerts: []
    };

    it('should render uptime monitor with correct data', () => {
        const monitor = new UptimeMonitor(mockUptimeData);
        const html = monitor.render();

        expect(html).toContain('99.50%');
        expect(html).toContain('10,000');
        expect(html).toContain('Block Signing History');
    });

    it('should classify uptime correctly', () => {
        const monitor = new UptimeMonitor(mockUptimeData);

        const excellentMonitor = new UptimeMonitor({ ...mockUptimeData, uptimePercentage: 99.5 });
        expect(excellentMonitor.getUptimeClass()).toBe('excellent');

        const goodMonitor = new UptimeMonitor({ ...mockUptimeData, uptimePercentage: 96 });
        expect(goodMonitor.getUptimeClass()).toBe('good');

        const warningMonitor = new UptimeMonitor({ ...mockUptimeData, uptimePercentage: 92 });
        expect(warningMonitor.getUptimeClass()).toBe('warning');

        const criticalMonitor = new UptimeMonitor({ ...mockUptimeData, uptimePercentage: 85 });
        expect(criticalMonitor.getUptimeClass()).toBe('critical');
    });

    it('should calculate recent uptime correctly', () => {
        const blocks = Array.from({ length: 100 }, (_, i) => ({
            signed: i < 98 // 98 out of 100 signed
        }));

        const monitor = new UptimeMonitor({ ...mockUptimeData, blocks });
        const recentUptime = monitor.calculateRecentUptime(100);

        expect(parseFloat(recentUptime)).toBe(98);
    });

    it('should render block grid', () => {
        const monitor = new UptimeMonitor(mockUptimeData);
        const html = monitor.renderBlockGrid();

        expect(html).toContain('block-cell');
        expect(html).toContain('signed');
    });

    it('should calculate time to slash correctly', () => {
        const safeMonitor = new UptimeMonitor({ ...mockUptimeData, missedBlocks: 50 });
        expect(safeMonitor.calculateTimeToSlash()).toBe('Safe');

        const criticalMonitor = new UptimeMonitor({ ...mockUptimeData, missedBlocks: 480 });
        expect(criticalMonitor.calculateTimeToSlash()).toContain('blocks');

        const slashedMonitor = new UptimeMonitor({ ...mockUptimeData, missedBlocks: 500 });
        expect(slashedMonitor.calculateTimeToSlash()).toBe('Slashed');
    });
});

// Run tests if this file is executed directly
if (require.main === module) {
    console.log('Component tests would run here with Jest');
}

module.exports = {};
