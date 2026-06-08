const { readState, stageURL } = require('./state');

async function stageAdmin(pathname, options = {}) {
  const state = readState();
  const headers = {
    'X-Stage-CalDAV-Admin-Token': state.stageAdminToken,
    ...(options.headers || {})
  };
  const response = await fetch(stageURL(pathname), {
    ...options,
    headers
  });
  if (!response.ok) {
    throw new Error(`stage admin request failed: ${options.method || 'GET'} ${pathname} ${response.status}`);
  }
  if (response.status === 204) {
    return null;
  }
  return response.json();
}

async function stageState() {
  return stageAdmin('/stage/admin/state');
}

async function createRemoteTask({ uid, title, href }) {
  return stageAdmin('/stage/admin/tasks', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ uid, title, href })
  });
}

async function updateRemoteTask({ href, uid, title }) {
  return stageAdmin('/stage/admin/tasks', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ href, uid, title })
  });
}

async function deleteRemoteTask(href) {
  return stageAdmin(`/stage/admin/tasks?href=${encodeURIComponent(href)}`, {
    method: 'DELETE'
  });
}

module.exports = {
  createRemoteTask,
  deleteRemoteTask,
  stageState,
  updateRemoteTask
};
