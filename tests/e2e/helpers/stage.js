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

async function resetStage() {
  return stageAdmin('/stage/admin/reset', {
    method: 'POST'
  });
}

async function createRemoteCalendar({ href, displayName }) {
  const state = readState();
  const response = await fetch(stageURL(href), {
    method: 'MKCALENDAR',
    headers: {
      Authorization: `Basic ${Buffer.from(`${state.stageUsername}:${state.stagePassword}`).toString('base64')}`,
      'Content-Type': 'application/xml'
    },
    body: `<?xml version="1.0" encoding="utf-8"?><d:mkcalendar xmlns:d="DAV:"><d:set><d:prop><d:displayname>${xmlEscape(displayName)}</d:displayname></d:prop></d:set></d:mkcalendar>`
  });
  if (!response.ok) {
    throw new Error(`stage calendar create failed: ${href} ${response.status}`);
  }
}

async function createRemoteTask({ calendarHref, uid, title, href, rawVTODO }) {
  return stageAdmin('/stage/admin/tasks', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      calendar_href: calendarHref,
      uid,
      title,
      href,
      raw_vtodo: rawVTODO
    })
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

function xmlEscape(value) {
  return String(value || '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&apos;');
}

module.exports = {
  createRemoteCalendar,
  createRemoteTask,
  deleteRemoteTask,
  resetStage,
  stageState,
  updateRemoteTask
};
