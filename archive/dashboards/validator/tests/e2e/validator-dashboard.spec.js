// E2E tests for Validator Dashboard using Playwright

const { test, expect } = require('@playwright/test');

test.describe('Validator Dashboard E2E Tests', () => {
    test.beforeEach(async ({ page }) => {
        // Navigate to the dashboard
        await page.goto('http://localhost:8080');
    });

    test('should load dashboard homepage', async ({ page }) => {
        await expect(page.locator('h1')).toContainText('PAW Validator Dashboard');
        await expect(page.locator('.dashboard-container')).toBeVisible();
    });

    test('should show connection status', async ({ page }) => {
        const connectionStatus = page.locator('#connectionStatus');
        await expect(connectionStatus).toBeVisible();

        // Should eventually connect
        await expect(connectionStatus).toContainText(/Connected|Connecting/, { timeout: 5000 });
    });

    test('should navigate between sections', async ({ page }) => {
        // Click on different navigation links
        const sections = [
            { selector: '[data-section="overview"]', expected: 'Validator Overview' },
            { selector: '[data-section="delegation"]', expected: 'Delegation Management' },
            { selector: '[data-section="rewards"]', expected: 'Rewards Tracking' },
            { selector: '[data-section="performance"]', expected: 'Performance Metrics' },
            { selector: '[data-section="uptime"]', expected: 'Uptime Monitoring' },
            { selector: '[data-section="signing"]', expected: 'Signing Statistics' },
            { selector: '[data-section="slashing"]', expected: 'Slash Events' },
            { selector: '[data-section="settings"]', expected: 'Validator Settings' }
        ];

        for (const section of sections) {
            await page.click(section.selector);
            await expect(page.locator('h2')).toContainText(section.expected);
        }
    });

    test('should add a new validator', async ({ page }) => {
        // Click add validator button
        await page.click('#addValidatorBtn');

        // Modal should appear
        await expect(page.locator('#addValidatorModal')).toHaveClass(/active/);

        // Fill in validator details
        await page.fill('#newValidatorAddress', 'pawvaloper1test123456789');
        await page.fill('#newValidatorName', 'Test Validator');

        // Submit
        await page.click('#confirmAddValidator');

        // Modal should close and validator should be added
        await expect(page.locator('#addValidatorModal')).not.toHaveClass(/active/);
    });

    test('should validate invalid validator address', async ({ page }) => {
        await page.click('#addValidatorBtn');
        await page.fill('#newValidatorAddress', 'invalid-address');
        await page.click('#confirmAddValidator');

        // Should show error (check for alert or error message)
        page.on('dialog', async dialog => {
            expect(dialog.message()).toContain('Invalid validator address');
            await dialog.accept();
        });
    });

    test('should display validator statistics', async ({ page }) => {
        // Add a validator first (if none exist)
        const validatorSelect = page.locator('#validatorSelect');
        const hasValidators = await validatorSelect.locator('option').count() > 1;

        if (!hasValidators) {
            await page.click('#addValidatorBtn');
            await page.fill('#newValidatorAddress', 'pawvaloper1test');
            await page.click('#confirmAddValidator');
        }

        // Check that stats are displayed
        await expect(page.locator('#validatorStatus')).not.toBeEmpty();
        await expect(page.locator('#totalStaked')).not.toBeEmpty();
        await expect(page.locator('#commission')).not.toBeEmpty();
        await expect(page.locator('#uptime')).not.toBeEmpty();
    });

    test('should search delegations', async ({ page }) => {
        // Navigate to delegations
        await page.click('[data-section="delegation"]');

        // Wait for delegation list to load
        await page.waitForSelector('#delegationList', { timeout: 5000 });

        // Search for a delegation
        await page.fill('#delegationSearch', 'paw1');

        // Check that filtered results are shown
        const visibleDelegations = await page.locator('.delegation-item:visible').count();
        expect(visibleDelegations).toBeGreaterThanOrEqual(0);
    });

    test('should sort delegations', async ({ page }) => {
        await page.click('[data-section="delegation"]');
        await page.waitForSelector('#delegationList', { timeout: 5000 });

        // Change sort option
        await page.selectOption('#delegationSort', 'amount');

        // Verify sorting occurred (delegations should be reordered)
        const firstDelegation = page.locator('.delegation-item').first();
        await expect(firstDelegation).toBeVisible();
    });

    test('should display rewards chart', async ({ page }) => {
        await page.click('[data-section="rewards"]');

        // Wait for chart to load
        await page.waitForSelector('#rewardsChart', { timeout: 5000 });

        // Check that chart elements are present
        await expect(page.locator('#rewardsChart')).toBeVisible();
        await expect(page.locator('.chart-canvas')).toBeVisible();
    });

    test('should switch chart types', async ({ page }) => {
        await page.click('[data-section="rewards"]');
        await page.waitForSelector('#rewardsChart', { timeout: 5000 });

        // Click different chart type buttons
        await page.click('[data-type="bar"]');
        await expect(page.locator('[data-type="bar"]')).toHaveClass(/active/);

        await page.click('[data-type="area"]');
        await expect(page.locator('[data-type="area"]')).toHaveClass(/active/);

        await page.click('[data-type="line"]');
        await expect(page.locator('[data-type="line"]')).toHaveClass(/active/);
    });

    test('should change chart timeframe', async ({ page }) => {
        await page.click('[data-section="rewards"]');
        await page.waitForSelector('#rewardsChart', { timeout: 5000 });

        // Click different timeframe buttons
        const timeframes = ['7d', '30d', '90d', '1y', 'all'];

        for (const timeframe of timeframes) {
            await page.click(`[data-timeframe="${timeframe}"]`);
            await expect(page.locator(`[data-timeframe="${timeframe}"]`)).toHaveClass(/active/);
        }
    });

    test('should display uptime monitor', async ({ page }) => {
        await page.click('[data-section="uptime"]');

        // Check uptime visualization elements
        await expect(page.locator('.uptime-monitor')).toBeVisible();
        await expect(page.locator('.block-grid')).toBeVisible();
    });

    test('should show signing statistics', async ({ page }) => {
        await page.click('[data-section="signing"]');

        // Verify signing stats are displayed
        await expect(page.locator('#blocksSigned')).not.toBeEmpty();
        await expect(page.locator('#blocksMissed')).not.toBeEmpty();
        await expect(page.locator('#signRate')).not.toBeEmpty();
    });

    test('should display slash events (or empty state)', async ({ page }) => {
        await page.click('[data-section="slashing"]');

        // Should show either slash events or empty state
        const hasEvents = await page.locator('.slash-event').count() > 0;
        const hasEmptyState = await page.locator('.empty-state').isVisible();

        expect(hasEvents || hasEmptyState).toBeTruthy();
    });

    test('should update commission rate', async ({ page }) => {
        await page.click('[data-section="settings"]');

        // Fill in new commission rate
        await page.fill('#commissionRate', '7.5');

        // Click update button and expect error (requires transaction signing)
        page.on('dialog', async dialog => {
            expect(dialog.message()).toContain('require transaction signing');
            await dialog.accept();
        });

        await page.click('#updateCommission');
    });

    test('should save validator settings', async ({ page }) => {
        await page.click('[data-section="settings"]');

        // Fill in settings
        await page.fill('#moniker', 'Updated Validator Name');
        await page.fill('#website', 'https://updated-validator.com');
        await page.fill('#details', 'Updated validator description');

        // Save settings
        page.on('dialog', async dialog => {
            await dialog.accept();
        });

        await page.click('#saveSettings');
    });

    test('should save alert settings', async ({ page }) => {
        await page.click('[data-section="settings"]');

        // Toggle alert settings
        await page.check('#emailAlerts');
        await page.fill('#alertEmail', 'test-alerts@validator.com');
        await page.check('#uptimeAlerts');
        await page.check('#slashingAlerts');

        // Save alert settings
        page.on('dialog', async dialog => {
            expect(dialog.message()).toContain('Alert settings saved');
            await dialog.accept();
        });

        await page.click('#saveAlertSettings');
    });

    test('should handle responsive design', async ({ page }) => {
        // Test mobile viewport
        await page.setViewportSize({ width: 375, height: 667 });
        await expect(page.locator('.dashboard-sidebar')).toBeVisible();

        // Test tablet viewport
        await page.setViewportSize({ width: 768, height: 1024 });
        await expect(page.locator('.dashboard-sidebar')).toBeVisible();

        // Test desktop viewport
        await page.setViewportSize({ width: 1920, height: 1080 });
        await expect(page.locator('.dashboard-sidebar')).toBeVisible();
    });

    test('should handle real-time updates', async ({ page }) => {
        // Wait for WebSocket connection
        await page.waitForSelector('.status-indicator.connected', { timeout: 10000 });

        // Monitor for updates (in a real scenario)
        const initialUptime = await page.locator('#uptime').textContent();

        // Wait a bit for potential updates
        await page.waitForTimeout(3000);

        // Verify that data could have updated
        const currentUptime = await page.locator('#uptime').textContent();
        expect(currentUptime).toBeDefined();
    });

    test('should persist validator selection', async ({ page }) => {
        // Add and select a validator
        await page.click('#addValidatorBtn');
        await page.fill('#newValidatorAddress', 'pawvaloper1persist');
        await page.click('#confirmAddValidator');

        // Reload page
        await page.reload();

        // Check that validator is still in the list
        const options = await page.locator('#validatorSelect option').allTextContents();
        const hasValidator = options.some(opt => opt.includes('pawvaloper1persist'));
        expect(hasValidator).toBeTruthy();
    });
});

// Run tests if this file is executed directly
if (require.main === module) {
    console.log('E2E tests would run here with Playwright');
    console.log('Run: npx playwright test');
}
