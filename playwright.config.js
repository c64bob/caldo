// @ts-check
const { defineConfig, devices } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './tests/e2e',
  fullyParallel: false,
  workers: 1,
  timeout: 90_000,
  expect: {
    timeout: 10_000
  },
  globalSetup: require.resolve('./tests/e2e/global-setup'),
  globalTeardown: require.resolve('./tests/e2e/global-teardown'),
  outputDir: 'test-results/e2e',
  reporter: process.env.CI
    ? [['dot'], ['html', { open: 'never' }]]
    : [['list'], ['html', { open: 'never' }]],
  use: {
    ...devices['Desktop Chrome'],
    extraHTTPHeaders: {
      'X-Forwarded-User': 'e2e-user'
    },
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure'
  }
});
