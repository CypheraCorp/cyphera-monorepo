import { test, expect } from '@playwright/test';

// Test configuration
const TEST_USER = {
  email: process.env.TEST_USER_EMAIL || 'test@example.com',
  password: process.env.TEST_USER_PASSWORD || 'testpassword123',
};

const API_BASE_URL = process.env.CYPHERA_API_BASE_URL || 'http://localhost:8000';
const APP_URL = process.env.APP_URL || 'http://localhost:3000';

test.describe('API Key Management', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to signin page
    await page.goto(`${APP_URL}/merchants/signin`);
    
    // Wait for page to load
    await page.waitForLoadState('networkidle');
  });

  test('should create, view, and delete API key', async ({ page }) => {
    // Step 1: Sign in
    await test.step('Sign in to merchant account', async () => {
      // Click Web3Auth button
      await page.click('button:has-text("Continue with Web3Auth")');
      
      // Handle Web3Auth popup (this will vary based on your Web3Auth setup)
      // For demo purposes, assuming email/password login
      const popup = await page.waitForEvent('popup');
      await popup.waitForLoadState();
      
      // Fill in credentials in popup
      await popup.fill('input[type="email"]', TEST_USER.email);
      await popup.fill('input[type="password"]', TEST_USER.password);
      await popup.click('button[type="submit"]');
      
      // Wait for redirect back to app
      await page.waitForURL('**/merchants/dashboard');
    });

    // Step 2: Navigate to settings
    await test.step('Navigate to settings page', async () => {
      // Click settings in sidebar
      await page.click('a[href="/merchants/settings"]');
      
      // Wait for settings page to load
      await page.waitForURL('**/merchants/settings');
      await expect(page.locator('h1')).toContainText('Settings');
    });

    // Step 3: Switch to API Keys tab
    await test.step('Switch to API Keys tab', async () => {
      await page.click('button[role="tab"]:has-text("API Keys")');
      
      // Verify tab is active
      await expect(page.locator('[role="tabpanel"][data-state="active"]')).toBeVisible();
    });

    // Step 4: Create new API key
    let apiKeyValue: string | null = null;
    const apiKeyName = `Test Key ${Date.now()}`;
    
    await test.step('Create new API key', async () => {
      // Click create button
      await page.click('button:has-text("Create API Key")');
      
      // Fill in the form
      await page.fill('input[id="name"]', apiKeyName);
      await page.click('button[role="combobox"]');
      await page.click('[role="option"]:has-text("Read & Write")');
      
      // Create the key
      await page.click('button:has-text("Create Key")');
      
      // Wait for key to be displayed
      await page.waitForSelector('code', { state: 'visible' });
      
      // Copy the API key value
      const codeElement = page.locator('code').first();
      apiKeyValue = await codeElement.textContent();
      expect(apiKeyValue).toBeTruthy();
      
      // Verify warning message is shown
      await expect(page.locator('text=Make sure to copy your API key now')).toBeVisible();
      
      // Click done
      await page.click('button:has-text("Done")');
    });

    // Step 5: Verify API key appears in list
    await test.step('Verify API key in list', async () => {
      // Check that the table contains our new key
      await expect(page.locator('table')).toContainText(apiKeyName);
      await expect(page.locator('table')).toContainText('cyk_');
      await expect(page.locator('table')).toContainText('WRITE');
    });

    // Step 6: Test API key functionality
    await test.step('Test API key works', async () => {
      // Make a test API call with the key
      const response = await page.request.get(`${API_BASE_URL}/api/v1/api-keys`, {
        headers: {
          'X-API-Key': apiKeyValue!,
        },
      });
      
      expect(response.ok()).toBeTruthy();
      const data = await response.json();
      expect(data.data).toBeInstanceOf(Array);
    });

    // Step 7: Delete the API key
    await test.step('Delete API key', async () => {
      // Find the row with our key and click delete
      const row = page.locator('tr', { hasText: apiKeyName });
      await row.locator('button[aria-label*="delete"], button:has(svg)').last().click();
      
      // Confirm deletion
      await page.waitForSelector('text=Delete API Key');
      await expect(page.locator('text=Are you sure you want to delete')).toBeVisible();
      await page.click('button:has-text("Delete")');
      
      // Wait for deletion to complete
      await page.waitForResponse(response => 
        response.url().includes('/api/api-keys/') && 
        response.request().method() === 'DELETE'
      );
      
      // Verify key is removed from list
      await expect(page.locator('table')).not.toContainText(apiKeyName);
    });

    // Step 8: Verify deleted key no longer works
    await test.step('Verify deleted key no longer works', async () => {
      const response = await page.request.get(`${API_BASE_URL}/api/v1/api-keys`, {
        headers: {
          'X-API-Key': apiKeyValue!,
        },
        failOnStatusCode: false,
      });
      
      expect(response.status()).toBe(401);
    });
  });

  test('should show proper validation errors', async ({ page }) => {
    // Assume already logged in (you can reuse the login step)
    // Navigate directly to settings
    await page.goto(`${APP_URL}/merchants/settings`);
    
    // Switch to API Keys tab
    await page.click('button[role="tab"]:has-text("API Keys")');
    
    // Try to create key without name
    await test.step('Validate empty name', async () => {
      await page.click('button:has-text("Create API Key")');
      await page.click('button:has-text("Create Key")');
      
      // Should show validation error
      await expect(page.locator('text=Please enter a name for the API key')).toBeVisible();
    });
  });

  test('should handle rate limiting', async ({ page }) => {
    // This test would verify that the UI properly handles rate limit errors
    // You'd need to trigger rate limiting somehow, perhaps by making many requests
  });
});

// Helper function to handle Web3Auth login
async function loginWithWeb3Auth(page: any, email: string, password: string) {
  // This would contain the actual Web3Auth login flow
  // The implementation depends on your Web3Auth configuration
}