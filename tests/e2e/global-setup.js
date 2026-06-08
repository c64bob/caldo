const fs = require('node:fs');
const net = require('node:net');
const os = require('node:os');
const path = require('node:path');
const { spawn } = require('node:child_process');
const { repoRoot, runtimeDir, writeState } = require('./helpers/state');

const adminToken = 'stage-admin';
const proxyUserHeader = 'X-Forwarded-User';
const encryptionKey = 'MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=';

module.exports = async function globalSetup() {
 fs.rmSync(runtimeDir, { recursive: true, force: true });
 fs.mkdirSync(runtimeDir, { recursive: true });

 const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'caldo-e2e-'));
 const stagePort = await freePort();
 const appPort = await freePort();
 const stageBaseURL = `http://127.0.0.1:${stagePort}`;
 const appBaseURL = `http://127.0.0.1:${appPort}`;
  const children = [];

  try {
    const stage = spawnLogged('stagecaldav', ['run', './cmd/stagecaldav'], {
      STAGE_CALDAV_ADDR: `127.0.0.1:${stagePort}`,
      STAGE_CALDAV_USERNAME: 'stage',
      STAGE_CALDAV_PASSWORD: 'stage',
      STAGE_CALDAV_ADMIN_TOKEN: adminToken
    });
    children.push(stage);

    await waitForHTTP(`${stageBaseURL}/stage/admin/state`, {
      headers: { 'X-Stage-CalDAV-Admin-Token': adminToken }
    }, stage);

    const app = spawnLogged('caldo', ['run', './cmd/caldo'], {
      APP_ENV: 'test',
      BASE_URL: 'https://caldo.e2e.test',
      DB_PATH: path.join(tempDir, 'caldo.db'),
      ENCRYPTION_KEY: encryptionKey,
      LOG_LEVEL: 'error',
      PORT: String(appPort),
      PROXY_USER_HEADER: proxyUserHeader
    });
    children.push(app);

    await waitForHTTP(`${appBaseURL}/health`, {}, app);

    writeState({
      appBaseURL,
      appPID: app.pid,
      appLog: path.join(runtimeDir, 'caldo.log'),
      stageBaseURL,
      stagePID: stage.pid,
      stageLog: path.join(runtimeDir, 'stagecaldav.log'),
      stageAdminToken: adminToken,
      stageUsername: 'stage',
      stagePassword: 'stage',
      proxyUserHeader,
      tempDir
    });
  } catch (error) {
    await Promise.all(children.map((child) => stopChild(child)));
    fs.rmSync(tempDir, { recursive: true, force: true });
    throw error;
  }
};

function spawnLogged(name, args, env) {
  const logPath = path.join(runtimeDir, `${name}.log`);
  const log = fs.openSync(logPath, 'a');
  const child = spawn('go', args, {
    cwd: repoRoot,
    env: { ...process.env, ...env },
    detached: true,
    stdio: ['ignore', log, log]
  });
  fs.closeSync(log);
  return child;
}

async function freePort() {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.on('error', reject);
    server.listen(0, '127.0.0.1', () => {
      const address = server.address();
      server.close(() => resolve(address.port));
    });
  });
}

async function waitForHTTP(url, options, child) {
  const deadline = Date.now() + 45_000;
  let lastError;
  while (Date.now() < deadline) {
    if (child.exitCode !== null) {
      throw new Error(`server exited before readiness check passed: ${url}`);
    }
    try {
      const response = await fetch(url, options);
      if (response.ok) {
        return;
      }
      lastError = new Error(`unexpected status ${response.status}`);
    } catch (error) {
      lastError = error;
    }
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  throw new Error(`timed out waiting for ${url}: ${lastError && lastError.message}`);
}

async function stopChild(child) {
  if (!child || !processGroupAlive(child.pid)) {
    return;
  }
  signalProcessGroup(child.pid, 'SIGTERM');
  const deadline = Date.now() + 5_000;
  while (Date.now() < deadline) {
    if (!processGroupAlive(child.pid)) {
      return;
    }
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  signalProcessGroup(child.pid, 'SIGKILL');
}

function signalProcessGroup(pid, signal) {
  try {
    process.kill(-pid, signal);
    return;
  } catch {
    // Fall back to the direct pid for non-POSIX environments.
  }
  try {
    process.kill(pid, signal);
  } catch {
    // Process already exited.
  }
}

function processGroupAlive(pid) {
  try {
    process.kill(-pid, 0);
    return true;
  } catch {
    // Fall back to the direct pid for non-POSIX environments.
  }
  try {
    process.kill(pid, 0);
    return true;
  } catch {
    return false;
  }
}
