const fs = require('node:fs');
const { readState, runtimeDir } = require('./helpers/state');

module.exports = async function globalTeardown() {
  let state;
  try {
    state = readState();
  } catch {
    return;
  }

  await Promise.all([
    stopProcess(state.appPID),
    stopProcess(state.stagePID)
  ]);

  if (process.env.CALDO_E2E_KEEP_ARTIFACTS !== '1') {
    fs.rmSync(state.tempDir, { recursive: true, force: true });
  }
};

async function stopProcess(pid) {
  if (!pid) {
    return;
  }
  if (!processGroupAlive(pid)) {
    return;
  }
  signalProcessGroup(pid, 'SIGTERM');

  const deadline = Date.now() + 8_000;
  while (Date.now() < deadline) {
    if (!processGroupAlive(pid)) {
      return;
    }
    await new Promise((resolve) => setTimeout(resolve, 150));
  }

  signalProcessGroup(pid, 'SIGKILL');
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
