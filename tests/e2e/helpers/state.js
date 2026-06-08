const fs = require('node:fs');
const path = require('node:path');

const repoRoot = path.resolve(__dirname, '../../..');
const runtimeDir = path.join(repoRoot, '.playwright', 'caldo-e2e');
const statePath = path.join(runtimeDir, 'state.json');

function readState() {
  return JSON.parse(fs.readFileSync(statePath, 'utf8'));
}

function writeState(state) {
  fs.mkdirSync(runtimeDir, { recursive: true });
  fs.writeFileSync(statePath, JSON.stringify(state, null, 2));
}

function appURL(pathname) {
  return new URL(pathname, readState().appBaseURL).toString();
}

function stageURL(pathname) {
  return new URL(pathname, readState().stageBaseURL).toString();
}

module.exports = {
  repoRoot,
  runtimeDir,
  statePath,
  readState,
  writeState,
  appURL,
  stageURL
};
