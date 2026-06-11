(function () {
  'use strict';

  var navState = { pendingView: null };
  var tabStateKey = '__caldoTabState';
  var tabState = loadTabState();
  var tabID = ensureTabID(tabState);
  var undoExpiryTimer = 0;
  var focusRefreshState = { inFlight: false, pending: false, lastRun: 0 };

  function generateTabID() {
    if (window.crypto && typeof window.crypto.randomUUID === 'function') {
      return window.crypto.randomUUID();
    }
    return 'tab-' + Date.now().toString(36) + '-' + Math.random().toString(36).slice(2);
  }

  function loadTabState() {
    var historyState = window.history ? window.history.state : null;
    if (!historyState || typeof historyState !== 'object') {
      return {};
    }
    var state = historyState[tabStateKey];
    return state && typeof state === 'object' ? state : {};
  }

  function saveTabState(state) {
    if (!window.history || typeof window.history.replaceState !== 'function') return;
    try {
      var nextHistoryState = window.history.state && typeof window.history.state === 'object' ? Object.assign({}, window.history.state) : {};
      nextHistoryState[tabStateKey] = state || {};
      window.history.replaceState(nextHistoryState, document.title, window.location.href);
    } catch (_) {
      // If history state is unavailable, the in-memory tab ID still works until reload.
    }
  }

  function ensureTabID(state) {
    if (!state.tabID) {
      state.tabID = generateTabID();
      saveTabState(state);
    }
    return state.tabID;
  }

  function isTypingTarget(target) {
    if (!target || !(target instanceof Element)) return false;
    if (target.closest('[contenteditable="true"]')) return true;
    var tag = target.tagName;
    return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || target.isContentEditable;
  }

  function goTo(path) {
    window.location.assign(path);
  }

  function shortcutViewPath(key) {
    switch (key) {
      case 't':
        return '/today';
      case 'u':
        return '/upcoming';
      case 'v':
        return '/favorites';
      case 'p':
        return '/projects';
      case 'l':
        return '/labels';
      case 'f':
        return '/filters';
      case 'c':
        return '/conflicts';
      case 'e':
        return '/settings';
      default:
        return '';
    }
  }

  function closestElement(target, selector) {
    if (!target) return null;
    var element = target instanceof Element ? target : target.parentElement;
    return element ? element.closest(selector) : null;
  }

  function toggleHelpDialog() {
    var dialog = document.querySelector('[data-shortcut-help-dialog]');
    if (!dialog) return;
    if (dialog.hasAttribute('open')) {
      dialog.close();
      return;
    }
    dialog.showModal();
  }

  function openMobileNav() {
    var dialog = document.querySelector('[data-mobile-nav-dialog]');
    if (!dialog || dialog.hasAttribute('open')) return;
    dialog.showModal();
  }

  function closeMobileNav() {
    var dialog = document.querySelector('[data-mobile-nav-dialog]');
    if (!dialog || !dialog.hasAttribute('open')) return;
    dialog.close();
  }

  function inlineCreateError(root) {
    return root ? root.querySelector('[data-inline-task-create-error]') : null;
  }

  function setInlineCreateError(root, message) {
    var error = inlineCreateError(root);
    if (!error) return;
    if (!message) {
      error.textContent = '';
      error.hidden = true;
      return;
    }
    error.textContent = message;
    error.hidden = false;
  }

  function openInlineCreate(root) {
    if (!root) return;
    var trigger = root.querySelector('[data-inline-task-create-trigger]');
    var form = root.querySelector('[data-inline-task-create-form]');
    if (!form) return;
    root.dataset.inlineTaskCreateOpen = 'true';
    if (trigger) {
      trigger.hidden = true;
      trigger.setAttribute('aria-expanded', 'true');
    }
    form.hidden = false;
    setInlineCreateError(root, '');
    var input = form.querySelector('[data-inline-task-create-title]');
    if (input) {
      window.setTimeout(function () {
        input.focus();
      }, 0);
    }
  }

  function closeInlineCreate(root) {
    if (!root) return;
    var trigger = root.querySelector('[data-inline-task-create-trigger]');
    var form = root.querySelector('[data-inline-task-create-form]');
    if (!form) return;
    form.reset();
    form.hidden = true;
    root.dataset.inlineTaskCreateOpen = 'false';
    setInlineCreateError(root, '');
    if (trigger) {
      trigger.hidden = false;
      trigger.setAttribute('aria-expanded', 'false');
      trigger.focus();
    }
  }

  function inlineEditError(root) {
    return root ? root.querySelector('[data-inline-task-edit-error]') : null;
  }

  function setInlineEditError(root, message) {
    var error = inlineEditError(root);
    if (!error) return;
    if (!message) {
      error.textContent = '';
      error.hidden = true;
      return;
    }
    error.textContent = message;
    error.hidden = false;
  }

  function taskRows() {
    return Array.prototype.slice.call(document.querySelectorAll('[data-task-id]'));
  }

  function taskRowID(row) {
    return row ? row.getAttribute('data-task-id') || '' : '';
  }

  function taskRowVersion(row) {
    var raw = row ? row.getAttribute('data-server-version') || '' : '';
    var version = parseInt(raw, 10);
    if (Number.isFinite(version) && version > 0) {
      return version;
    }
    var expectedVersion = row ? row.querySelector('input[name="expected_version"]') : null;
    version = expectedVersion ? parseInt(expectedVersion.value || '', 10) : 0;
    return Number.isFinite(version) ? version : 0;
  }

  function controlIsDirty(control) {
    if (!control || control.disabled || !control.name) return false;
    var type = (control.type || '').toLowerCase();
    if (type === 'checkbox' || type === 'radio') {
      return control.checked !== control.defaultChecked;
    }
    if (control.tagName === 'SELECT') {
      return Array.prototype.some.call(control.options, function (option) {
        return option.selected !== option.defaultSelected;
      });
    }
    return control.value !== control.defaultValue;
  }

  function formIsDirty(form) {
    if (!form || !form.elements) return false;
    return Array.prototype.some.call(form.elements, controlIsDirty);
  }

  function openTaskForms(row) {
    var forms = [];
    if (!row) return forms;

    var inlineEdit = row.querySelector('[data-inline-task-edit-form]');
    if (inlineEdit && !inlineEdit.hidden) {
      forms.push(inlineEdit);
    }

    var subtaskCreate = row.querySelector('[data-subtask-create-form]');
    if (subtaskCreate && !subtaskCreate.hidden) {
      forms.push(subtaskCreate);
    }

    var detailDialog = row.querySelector('[data-task-detail-dialog]');
    if (detailDialog && detailDialog.open) {
      var detailForm = detailDialog.querySelector('[data-task-detail-form]');
      if (detailForm) {
        forms.push(detailForm);
      }
    }

    return forms;
  }

  function rowHasDirtyOpenForm(row) {
    return openTaskForms(row).some(formIsDirty);
  }

  function markStaleLocalChanges(row) {
    if (!row) return;
    var message = 'Aufgabe wurde in einem anderen Tab geändert. Lokale Eingaben bleiben erhalten.';
    row.setAttribute('data-stale-local-changes', 'true');
    setInlineEditError(row, message);
    setInlineCreateError(row.querySelector('[data-subtask-create]'), message);
    setTaskDetailError(row.querySelector('[data-task-detail-dialog]'), message);
    setTaskActionError(row, message);
    setWriteStatus('warning', message);
  }

  function taskRowFromHTML(html, taskID) {
    var template = document.createElement('template');
    template.innerHTML = (html || '').trim();
    var rows = template.content.querySelectorAll('[data-task-id]');
    for (var i = 0; i < rows.length; i += 1) {
      if (taskRowID(rows[i]) === taskID) {
        return rows[i];
      }
    }
    return null;
  }

  function replaceTaskRow(row, html) {
    var taskID = taskRowID(row);
    var nextRow = taskRowFromHTML(html, taskID);
    if (!nextRow) return false;
    row.replaceWith(nextRow);
    if (window.htmx && typeof window.htmx.process === 'function') {
      window.htmx.process(nextRow);
    }
    return true;
  }

  function removeTaskRow(row) {
    if (row) {
      row.remove();
    }
  }

  function fetchTaskFragment(row) {
    var taskID = taskRowID(row);
    if (!taskID) return Promise.resolve();
    return window.fetch('/tasks/' + encodeURIComponent(taskID), {
      method: 'GET',
      credentials: 'same-origin',
      headers: {
        'Accept': 'text/html',
        'X-Tab-ID': tabID
      }
    }).then(function (response) {
      if (response.status === 404) {
        removeTaskRow(row);
        return '';
      }
      if (!response.ok) {
        throw response;
      }
      return response.text();
    }).then(function (html) {
      if (html) {
        replaceTaskRow(row, html);
      }
    }).catch(function () {
      // Focus refresh is opportunistic; normal writes still surface their own errors.
    });
  }

  function fetchCurrentTaskVersions(ids) {
    if (ids.length === 0) {
      return Promise.resolve({});
    }
    return window.fetch('/api/tasks/versions?ids=' + encodeURIComponent(ids.join(',')), {
      method: 'GET',
      credentials: 'same-origin',
      headers: {
        'Accept': 'application/json',
        'X-Tab-ID': tabID
      }
    }).then(function (response) {
      if (!response.ok) {
        throw response;
      }
      return response.json();
    }).then(function (payload) {
      var versions = {};
      (payload.tasks || []).forEach(function (task) {
        if (task && task.task_id) {
          versions[task.task_id] = task.server_version;
        }
      });
      return versions;
    });
  }

  function refreshStaleTaskRows() {
    if (window.location.pathname.indexOf('/setup') === 0 || writeState.pendingRequests > 0) {
      return Promise.resolve();
    }

    var rows = taskRows();
    var knownRows = rows.filter(function (row) {
      return taskRowID(row) !== '' && taskRowVersion(row) > 0;
    });
    if (knownRows.length === 0) {
      return Promise.resolve();
    }

    var ids = knownRows.map(taskRowID);
    return fetchCurrentTaskVersions(ids).then(function (versions) {
      var refreshes = [];
      knownRows.forEach(function (row) {
        if (!document.contains(row)) return;
        var taskID = taskRowID(row);
        var currentVersion = taskRowVersion(row);
        var serverVersion = versions[taskID];
        var stale = typeof serverVersion !== 'number' || serverVersion !== currentVersion;
        if (!stale) return;
        if (rowHasDirtyOpenForm(row)) {
          markStaleLocalChanges(row);
          return;
        }
        if (typeof serverVersion !== 'number') {
          removeTaskRow(row);
          return;
        }
        refreshes.push(fetchTaskFragment(row));
      });
      return Promise.all(refreshes);
    }).catch(function () {
      // Version checks run on focus and should not interrupt the user.
    });
  }

  function scheduleFocusRefresh() {
    if (document.visibilityState && document.visibilityState !== 'visible') return;
    var now = Date.now();
    if (focusRefreshState.inFlight) {
      focusRefreshState.pending = true;
      return;
    }
    if (now - focusRefreshState.lastRun < 500) {
      return;
    }
    focusRefreshState.inFlight = true;
    focusRefreshState.lastRun = now;
    refreshStaleTaskRows().finally(function () {
      focusRefreshState.inFlight = false;
      if (focusRefreshState.pending) {
        focusRefreshState.pending = false;
        scheduleFocusRefresh();
      }
    });
  }

  function openInlineEdit(root) {
    if (!root) return;
    var display = root.querySelector('[data-inline-task-display]');
    var form = root.querySelector('[data-inline-task-edit-form]');
    var trigger = root.querySelector('[data-inline-task-edit-open]');
    if (!form || !display) return;
    root.dataset.inlineTaskEditOpen = 'true';
    display.hidden = true;
    form.hidden = false;
    if (trigger) {
      trigger.setAttribute('aria-expanded', 'true');
    }
    setInlineEditError(root, '');
    var input = form.querySelector('[data-inline-task-edit-title]');
    if (input) {
      window.setTimeout(function () {
        input.focus();
        input.select();
      }, 0);
    }
  }

  function closeInlineEdit(root) {
    if (!root) return;
    var display = root.querySelector('[data-inline-task-display]');
    var form = root.querySelector('[data-inline-task-edit-form]');
    var trigger = root.querySelector('[data-inline-task-edit-open]');
    if (!form || !display) return;
    form.reset();
    form.hidden = true;
    display.hidden = false;
    root.dataset.inlineTaskEditOpen = 'false';
    setInlineEditError(root, '');
    if (trigger) {
      trigger.setAttribute('aria-expanded', 'false');
      trigger.focus();
    }
  }

  function taskDetailError(dialog) {
    return dialog ? dialog.querySelector('[data-task-detail-error]') : null;
  }

  function setTaskDetailError(dialog, message) {
    var error = taskDetailError(dialog);
    if (!error) return;
    if (!message) {
      error.textContent = '';
      error.hidden = true;
      return;
    }
    error.textContent = message;
    error.hidden = false;
  }

  function resetTaskRecurrenceUpdate(form) {
    var marker = form ? form.querySelector('[data-task-recurrence-update]') : null;
    if (marker) {
      marker.disabled = true;
    }
  }

  function markTaskRecurrenceUpdate(control) {
    var form = control ? control.closest('[data-task-detail-form]') : null;
    var marker = form ? form.querySelector('[data-task-recurrence-update]') : null;
    if (marker) {
      marker.disabled = false;
    }
  }

  function handleTaskRecurrenceControlEvent(event) {
    var control = closestElement(event.target, '[data-task-recurrence-control]');
    if (control) {
      markTaskRecurrenceUpdate(control);
    }
  }

  function bindTaskDetail(dialog) {
    if (!dialog || dialog.dataset.taskDetailBound === 'true') return;
    dialog.dataset.taskDetailBound = 'true';
    dialog.addEventListener('close', function () {
      var form = dialog.querySelector('[data-task-detail-form]');
      if (form) {
        form.reset();
        resetTaskRecurrenceUpdate(form);
      }
      var trigger = dialog.__caldoReturnFocus;
      dialog.__caldoReturnFocus = null;
      if (trigger && document.contains(trigger)) {
        trigger.setAttribute('aria-expanded', 'false');
        trigger.focus();
      }
    });
    dialog.addEventListener('click', function (event) {
      if (event.target === dialog) {
        closeTaskDetail(dialog);
      }
    });
  }

  function openTaskDetail(trigger) {
    if (!trigger) return;
    var task = trigger.closest('[data-task-id]');
    var dialog = task ? task.querySelector('[data-task-detail-dialog]') : null;
    if (!dialog) return;
    bindTaskDetail(dialog);
    dialog.__caldoReturnFocus = trigger;
    trigger.setAttribute('aria-expanded', 'true');
    if (typeof dialog.showModal === 'function') {
      if (!dialog.open) {
        dialog.showModal();
      }
    } else {
      dialog.setAttribute('open', '');
    }
    var input = dialog.querySelector('[data-task-detail-title]');
    var close = dialog.querySelector('[data-task-detail-close]');
    window.setTimeout(function () {
      if (input && !input.disabled) {
        input.focus();
        input.select();
        return;
      }
      if (close) close.focus();
    }, 0);
  }

  function closeTaskDetail(dialog) {
    if (!dialog) return;
    if (typeof dialog.close === 'function' && dialog.open) {
      dialog.close();
      return;
    }
    dialog.removeAttribute('open');
    var form = dialog.querySelector('[data-task-detail-form]');
    if (form) {
      form.reset();
      resetTaskRecurrenceUpdate(form);
    }
    var trigger = dialog.__caldoReturnFocus;
    dialog.__caldoReturnFocus = null;
    if (trigger && document.contains(trigger)) {
      trigger.setAttribute('aria-expanded', 'false');
      trigger.focus();
    }
  }

  function taskDetailFailureMessage(status) {
    if (status === 409) {
      return 'Aufgabe hat einen Konflikt. Änderungen wurden nicht als gespeichert markiert.';
    }
    return 'Aufgabe konnte nicht gespeichert werden.';
  }

  function taskActionError(row) {
    return row ? row.querySelector('[data-task-action-error]') : null;
  }

  function setTaskActionError(row, message) {
    var error = taskActionError(row);
    if (!error) return;
    if (!message) {
      error.textContent = '';
      error.hidden = true;
      return;
    }
    error.textContent = message;
    error.hidden = false;
  }

  function taskCompleteError(dialog) {
    return dialog ? dialog.querySelector('[data-task-complete-error]') : null;
  }

  function setTaskCompleteError(dialog, message) {
    var error = taskCompleteError(dialog);
    if (!error) return;
    if (!message) {
      error.textContent = '';
      error.hidden = true;
      return;
    }
    error.textContent = message;
    error.hidden = false;
  }

  function bindTaskComplete(dialog) {
    if (!dialog || dialog.dataset.taskCompleteBound === 'true') return;
    dialog.dataset.taskCompleteBound = 'true';
    dialog.addEventListener('close', function () {
      dialog.querySelectorAll('[data-task-complete-form]').forEach(function (form) {
        form.reset();
      });
      setTaskCompleteError(dialog, '');
      var trigger = dialog.__caldoReturnFocus;
      dialog.__caldoReturnFocus = null;
      if (trigger && document.contains(trigger)) {
        trigger.setAttribute('aria-expanded', 'false');
        trigger.focus();
      }
    });
    dialog.addEventListener('click', function (event) {
      if (event.target === dialog) {
        closeTaskComplete(dialog);
      }
    });
  }

  function openTaskComplete(trigger) {
    if (!trigger) return;
    var task = trigger.closest('[data-task-id]');
    var dialog = task ? task.querySelector('[data-task-complete-dialog]') : null;
    if (!dialog) return;
    bindTaskComplete(dialog);
    dialog.__caldoReturnFocus = trigger;
    trigger.setAttribute('aria-expanded', 'true');
    setTaskCompleteError(dialog, '');
    setTaskActionError(task, '');
    if (typeof dialog.showModal === 'function') {
      if (!dialog.open) {
        dialog.showModal();
      }
    } else {
      dialog.setAttribute('open', '');
    }
    var cancel = dialog.querySelector('[data-task-complete-cancel]');
    window.setTimeout(function () {
      if (cancel) cancel.focus();
    }, 0);
  }

  function closeTaskComplete(dialog) {
    if (!dialog) return;
    if (typeof dialog.close === 'function' && dialog.open) {
      dialog.close();
      return;
    }
    dialog.removeAttribute('open');
    dialog.querySelectorAll('[data-task-complete-form]').forEach(function (form) {
      form.reset();
    });
    setTaskCompleteError(dialog, '');
    var trigger = dialog.__caldoReturnFocus;
    dialog.__caldoReturnFocus = null;
    if (trigger && document.contains(trigger)) {
      trigger.setAttribute('aria-expanded', 'false');
      trigger.focus();
    }
  }

  function taskDeleteError(dialog) {
    return dialog ? dialog.querySelector('[data-task-delete-error]') : null;
  }

  function setTaskDeleteError(dialog, message) {
    var error = taskDeleteError(dialog);
    if (!error) return;
    if (!message) {
      error.textContent = '';
      error.hidden = true;
      return;
    }
    error.textContent = message;
    error.hidden = false;
  }

  function bindTaskDelete(dialog) {
    if (!dialog || dialog.dataset.taskDeleteBound === 'true') return;
    dialog.dataset.taskDeleteBound = 'true';
    dialog.addEventListener('close', function () {
      var form = dialog.querySelector('[data-task-delete-form]');
      if (form) form.reset();
      setTaskDeleteError(dialog, '');
      var trigger = dialog.__caldoReturnFocus;
      dialog.__caldoReturnFocus = null;
      if (trigger && document.contains(trigger)) {
        trigger.setAttribute('aria-expanded', 'false');
        trigger.focus();
      }
    });
    dialog.addEventListener('click', function (event) {
      if (event.target === dialog) {
        closeTaskDelete(dialog);
      }
    });
  }

  function openTaskDelete(trigger) {
    if (!trigger) return;
    var task = trigger.closest('[data-task-id]');
    var dialog = task ? task.querySelector('[data-task-delete-dialog]') : null;
    if (!dialog) return;
    bindTaskDelete(dialog);
    dialog.__caldoReturnFocus = trigger;
    trigger.setAttribute('aria-expanded', 'true');
    setTaskDeleteError(dialog, '');
    setTaskActionError(task, '');
    if (typeof dialog.showModal === 'function') {
      if (!dialog.open) {
        dialog.showModal();
      }
    } else {
      dialog.setAttribute('open', '');
    }
    var cancel = dialog.querySelector('[data-task-delete-cancel]');
    window.setTimeout(function () {
      if (cancel) cancel.focus();
    }, 0);
  }

  function closeTaskDelete(dialog) {
    if (!dialog) return;
    if (typeof dialog.close === 'function' && dialog.open) {
      dialog.close();
      return;
    }
    dialog.removeAttribute('open');
    var form = dialog.querySelector('[data-task-delete-form]');
    if (form) form.reset();
    setTaskDeleteError(dialog, '');
    var trigger = dialog.__caldoReturnFocus;
    dialog.__caldoReturnFocus = null;
    if (trigger && document.contains(trigger)) {
      trigger.setAttribute('aria-expanded', 'false');
      trigger.focus();
    }
  }

  function taskActionFailureMessage(status) {
    if (status === 409) {
      return 'Aktion konnte nicht ausgeführt werden. Aufgabe prüfen.';
    }
    return 'Aktion konnte nicht ausgeführt werden.';
  }

  function taskCompleteFailureMessage(status) {
    if (status === 409) {
      return 'Aufgabe konnte nicht erledigt werden. Aufgabe prüfen.';
    }
    if (status === 502) {
      return 'Aufgabe konnte nicht auf dem CalDAV-Server erledigt werden.';
    }
    return 'Aufgabe konnte nicht erledigt werden.';
  }

  function taskDeleteFailureMessage(status) {
    if (status === 409) {
      return 'Aufgabe konnte nicht gelöscht werden. Aufgabe prüfen.';
    }
    if (status === 502) {
      return 'Aufgabe konnte nicht auf dem CalDAV-Server gelöscht werden.';
    }
    return 'Aufgabe konnte nicht gelöscht werden.';
  }

  function visibleDirectSubtaskRows(parentID) {
    if (!parentID) return [];
    if (window.CSS && typeof window.CSS.escape === 'function') {
      return Array.prototype.slice.call(document.querySelectorAll('[data-parent-task-id="' + window.CSS.escape(parentID) + '"]'));
    }
    return Array.prototype.filter.call(document.querySelectorAll('[data-parent-task-id]'), function (row) {
      return row.getAttribute('data-parent-task-id') === parentID;
    });
  }

  function removeDeletedTaskRows(row) {
    if (!row) return;
    var taskID = row.getAttribute('data-task-id') || '';
    visibleDirectSubtaskRows(taskID).forEach(function (childRow) {
      childRow.remove();
    });
    row.remove();
  }

  function notificationsRoot() {
    return document.getElementById('notifications');
  }

  function clearUndoExpiryTimer() {
    if (undoExpiryTimer) {
      window.clearTimeout(undoExpiryTimer);
      undoExpiryTimer = 0;
    }
  }

  function setNotificationHTML(html) {
    var root = notificationsRoot();
    if (!root) return;
    clearUndoExpiryTimer();
    root.innerHTML = html || '';
    bindUndoCountdown(root.querySelector('[data-undo-toast]'));
  }

  function clearNotifications() {
    setNotificationHTML('');
  }

  function showUndoResult(message) {
    var root = document.getElementById('notifications');
    if (!root) return;
    clearUndoExpiryTimer();
    root.textContent = '';

    var toast = document.createElement('div');
    toast.className = 'caldo-toast caldo-undo-toast caldo-undo-toast-result';
    toast.setAttribute('role', 'status');
    toast.setAttribute('data-undo-toast', '');

    var copy = document.createElement('div');
    copy.className = 'caldo-undo-toast-copy';

    var text = document.createElement('span');
    text.className = 'caldo-undo-status';
    text.setAttribute('data-undo-status', '');
    text.textContent = message;

    copy.appendChild(text);
    toast.appendChild(copy);
    root.appendChild(toast);
  }

  function refreshUndoNotification() {
    if (!undoStatusEnabled()) {
      return Promise.resolve();
    }
    return window.fetch('/tasks/undo/status', {
      method: 'GET',
      credentials: 'same-origin',
      headers: {
        'X-Tab-ID': tabID
      }
    }).then(function (response) {
      if (response.status === 204) {
        clearNotifications();
        return '';
      }
      if (!response.ok) {
        throw response;
      }
      return response.text();
    }).then(function (html) {
      if (html) {
        setNotificationHTML(html);
      }
    }).catch(function () {
      // Undo status is informational; write operations surface their own errors.
    });
  }

  function bindUndoCountdown(toast) {
    if (!toast) return;
    var expiresAt = toast.getAttribute('data-undo-expires-at');
    if (!expiresAt) return;

    function tick() {
      if (!document.contains(toast)) return;
      var remaining = Date.parse(expiresAt) - Date.now();
      if (!Number.isFinite(remaining) || remaining <= 0) {
        markUndoExpired(toast);
        return;
      }
      var seconds = Math.ceil(remaining / 1000);
      var minutes = Math.floor(seconds / 60);
      var rest = seconds % 60;
      var countdown = toast.querySelector('[data-undo-countdown]');
      if (countdown) {
        countdown.textContent = 'Noch ' + minutes + ':' + String(rest).padStart(2, '0') + ' verfügbar.';
      }
      undoExpiryTimer = window.setTimeout(tick, Math.min(1000, remaining));
    }

    tick();
  }

  function markUndoExpired(toast) {
    var status = toast.querySelector('[data-undo-status]');
    var countdown = toast.querySelector('[data-undo-countdown]');
    var button = toast.querySelector('[data-undo-action]');
    if (status) {
      status.textContent = 'Rückgängig abgelaufen.';
    }
    if (countdown) {
      countdown.textContent = 'Der Snapshot ist nicht mehr gültig.';
    }
    if (button) {
      button.disabled = true;
      button.hidden = true;
    }
    toast.classList.add('caldo-undo-toast-expired');
  }

  function setUndoNotificationError(button, message) {
    var toast = button ? button.closest('.caldo-undo-toast') : null;
    var status = toast ? toast.querySelector('[data-undo-status]') : null;
    if (status) {
      status.textContent = message;
    }
    if (toast) {
      toast.classList.add('caldo-undo-toast-error');
    }
  }

  function undoFailureMessage(status) {
    if (status === 404) {
      return 'Rückgängig ist nicht mehr verfügbar.';
    }
    if (status === 409) {
      return 'Rückgängig nicht möglich. Aufgabe prüfen.';
    }
    if (status === 502 || status === 424) {
      return 'Rückgängig konnte nicht auf dem CalDAV-Server gespeichert werden.';
    }
    return 'Rückgängig fehlgeschlagen.';
  }

  function executeUndo(button) {
    if (!button || button.disabled) return;
    button.disabled = true;
    setWriteStatus('pending', 'Speichern ...');
    window.fetch('/tasks/undo', {
      method: 'POST',
      credentials: 'same-origin',
      headers: {
        'X-CSRF-Token': csrfToken(),
        'X-Tab-ID': tabID,
        'Accept': 'text/html'
      }
    }).then(function (response) {
      if (!response.ok) {
        throw response;
      }
      return response.text();
    }).then(function (html) {
      if (html) {
        setNotificationHTML(html);
      } else {
        showUndoResult('Rückgängig ausgeführt.');
      }
      setWriteStatus('saved', 'Rückgängig ausgeführt.');
      window.setTimeout(function () {
        window.location.reload();
      }, 1200);
    }).catch(function (response) {
      var status = response && response.status ? response.status : 0;
      button.disabled = false;
      setWriteStatus('error', 'Rückgängig fehlgeschlagen. Änderungen prüfen.');
      setUndoNotificationError(button, undoFailureMessage(status));
    });
  }

  var writeState = { pendingRequests: 0 };

  function setWriteStatus(kind, message) {
    var el = document.getElementById('write-status');
    if (!el) return;
    if (!message) {
      el.textContent = '';
      el.classList.add('hidden');
      el.classList.remove('text-emerald-700', 'border-emerald-300', 'bg-emerald-50', 'text-amber-700', 'border-amber-300', 'bg-amber-50', 'text-red-700', 'border-red-300', 'bg-red-50');
      return;
    }

    el.textContent = message;
    el.classList.remove('hidden');
    el.classList.remove('text-emerald-700', 'border-emerald-300', 'bg-emerald-50', 'text-amber-700', 'border-amber-300', 'bg-amber-50', 'text-red-700', 'border-red-300', 'bg-red-50');

    if (kind === 'error') {
      el.classList.add('text-red-700', 'border-red-300', 'bg-red-50');
      return;
    }

    if (kind === 'saved') {
      el.classList.add('text-emerald-700', 'border-emerald-300', 'bg-emerald-50');
      return;
    }

    el.classList.add('text-amber-700', 'border-amber-300', 'bg-amber-50');
  }

  function clearSavedStatusSoon() {
    window.setTimeout(function () {
      if (writeState.pendingRequests === 0) {
        setWriteStatus(null, '');
      }
    }, 1200);
  }

  function csrfToken() {
    var meta = document.querySelector('meta[name="csrf-token"]');
    return meta ? meta.getAttribute('content') || '' : '';
  }

  function undoStatusEnabled() {
    return window.location.pathname.indexOf('/setup') !== 0;
  }

  function initializeUndoSurface() {
    if (!undoStatusEnabled()) {
      return;
    }
    refreshUndoNotification();
  }

  document.body.addEventListener('htmx:configRequest', function (event) {
    if (!event.detail || !event.detail.headers) return;
    event.detail.headers['X-Tab-ID'] = tabID;
    var token = csrfToken();
    if (token && !event.detail.headers['X-CSRF-Token']) {
      event.detail.headers['X-CSRF-Token'] = token;
    }
  });

  document.body.addEventListener('htmx:beforeRequest', function (event) {
    var method = ((event.detail && event.detail.requestConfig && event.detail.requestConfig.verb) || '').toUpperCase();
    if (!method || method === 'GET') return;
    writeState.pendingRequests += 1;
    setWriteStatus('pending', 'Speichern ...');
    var inlineCreate = closestElement(event.detail && event.detail.elt, '[data-inline-task-create]');
    if (inlineCreate) {
      setInlineCreateError(inlineCreate, '');
    }
    var inlineEdit = closestElement(event.detail && event.detail.elt, '[data-task-id]');
    if (inlineEdit && inlineEdit.querySelector('[data-inline-task-edit-form]')) {
      setInlineEditError(inlineEdit, '');
    }
    var taskDetail = closestElement(event.detail && event.detail.elt, '[data-task-detail-form]');
    if (taskDetail) {
      setTaskDetailError(taskDetail.closest('[data-task-detail-dialog]'), '');
    }
    var taskAction = closestElement(event.detail && event.detail.elt, '[data-task-action-form]');
    if (taskAction) {
      setTaskActionError(taskAction.closest('[data-task-id]'), '');
    }
    var taskComplete = closestElement(event.detail && event.detail.elt, '[data-task-complete-form]');
    if (taskComplete) {
      var completeDialog = taskComplete.closest('[data-task-complete-dialog]');
      setTaskCompleteError(completeDialog, '');
      setTaskActionError(taskComplete.closest('[data-task-id]'), '');
    }
    var taskDelete = closestElement(event.detail && event.detail.elt, '[data-task-delete-form]');
    if (taskDelete) {
      var deleteDialog = taskDelete.closest('[data-task-delete-dialog]');
      setTaskDeleteError(deleteDialog, '');
      setTaskActionError(taskDelete.closest('[data-task-id]'), '');
    }
  });

  document.body.addEventListener('htmx:afterRequest', function (event) {
    var method = ((event.detail && event.detail.requestConfig && event.detail.requestConfig.verb) || '').toUpperCase();
    if (!method || method === 'GET') return;
    if (writeState.pendingRequests > 0) {
      writeState.pendingRequests -= 1;
    }

    var successful = !!(event.detail && event.detail.successful);
    if (!successful) {
      setWriteStatus('error', 'Speichern fehlgeschlagen. Änderungen prüfen.');
      var failedInlineCreate = closestElement(event.detail && event.detail.elt, '[data-inline-task-create]');
      if (failedInlineCreate) {
        setInlineCreateError(failedInlineCreate, 'Aufgabe konnte nicht gespeichert werden.');
      }
      var failedTaskDetail = closestElement(event.detail && event.detail.elt, '[data-task-detail-form]');
      if (failedTaskDetail) {
        var detailStatus = event.detail && event.detail.xhr ? event.detail.xhr.status : 0;
        setTaskDetailError(failedTaskDetail.closest('[data-task-detail-dialog]'), taskDetailFailureMessage(detailStatus));
        return;
      }
      var failedTaskComplete = closestElement(event.detail && event.detail.elt, '[data-task-complete-form]');
      if (failedTaskComplete) {
        var completeStatus = event.detail && event.detail.xhr ? event.detail.xhr.status : 0;
        var completeMessage = taskCompleteFailureMessage(completeStatus);
        setTaskCompleteError(failedTaskComplete.closest('[data-task-complete-dialog]'), completeMessage);
        setTaskActionError(failedTaskComplete.closest('[data-task-id]'), completeMessage);
        return;
      }
      var failedTaskDelete = closestElement(event.detail && event.detail.elt, '[data-task-delete-form]');
      if (failedTaskDelete) {
        var deleteStatus = event.detail && event.detail.xhr ? event.detail.xhr.status : 0;
        var deleteMessage = taskDeleteFailureMessage(deleteStatus);
        setTaskDeleteError(failedTaskDelete.closest('[data-task-delete-dialog]'), deleteMessage);
        setTaskActionError(failedTaskDelete.closest('[data-task-id]'), deleteMessage);
        return;
      }
      var failedTaskAction = closestElement(event.detail && event.detail.elt, '[data-task-action-form]');
      if (failedTaskAction) {
        var actionStatus = event.detail && event.detail.xhr ? event.detail.xhr.status : 0;
        setTaskActionError(failedTaskAction.closest('[data-task-id]'), taskActionFailureMessage(actionStatus));
        return;
      }
      var failedInlineEdit = closestElement(event.detail && event.detail.elt, '[data-task-id]');
      if (failedInlineEdit && failedInlineEdit.querySelector('[data-inline-task-edit-form]')) {
        setInlineEditError(failedInlineEdit, 'Aufgabe konnte nicht gespeichert werden.');
        var status = event.detail && event.detail.xhr ? event.detail.xhr.status : 0;
        if (status === 409 || status >= 500) {
          window.location.reload();
        }
      }
      return;
    }

    if (writeState.pendingRequests === 0) {
      setWriteStatus('saved', 'Gespeichert');
      clearSavedStatusSoon();
    }

    var successfulTaskComplete = closestElement(event.detail && event.detail.elt, '[data-task-complete-form]');
    if (successfulTaskComplete) {
      closeTaskComplete(successfulTaskComplete.closest('[data-task-complete-dialog]'));
    }

    var successfulTaskDelete = closestElement(event.detail && event.detail.elt, '[data-task-delete-form]');
    if (successfulTaskDelete) {
      var deleteRow = successfulTaskDelete.closest('[data-task-id]');
      closeTaskDelete(successfulTaskDelete.closest('[data-task-delete-dialog]'));
      removeDeletedTaskRows(deleteRow);
      setWriteStatus(null, '');
      refreshUndoNotification();
      return;
    }

    if (closestElement(event.detail && event.detail.elt, '[data-refresh-after-write]')) {
      window.location.reload();
    }
  });

  window.addEventListener('beforeunload', function (event) {
    if (writeState.pendingRequests <= 0) return;
    event.preventDefault();
    event.returnValue = '';
  });

  document.addEventListener('input', handleTaskRecurrenceControlEvent, true);
  document.addEventListener('change', handleTaskRecurrenceControlEvent, true);

  document.addEventListener('click', function (event) {
    var inlineCreateTrigger = closestElement(event.target, '[data-inline-task-create-trigger]');
    if (inlineCreateTrigger) {
      event.preventDefault();
      openInlineCreate(inlineCreateTrigger.closest('[data-inline-task-create]'));
      return;
    }

    var inlineCreateCancel = closestElement(event.target, '[data-inline-task-create-cancel]');
    if (inlineCreateCancel) {
      event.preventDefault();
      closeInlineCreate(inlineCreateCancel.closest('[data-inline-task-create]'));
      return;
    }

    var inlineEditCancel = closestElement(event.target, '[data-inline-task-edit-cancel]');
    if (inlineEditCancel) {
      event.preventDefault();
      closeInlineEdit(inlineEditCancel.closest('[data-task-id]'));
      return;
    }

    var inlineEditOpen = closestElement(event.target, '[data-inline-task-edit-open]');
    if (inlineEditOpen && !closestElement(event.target, '[data-inline-task-edit-form]')) {
      event.preventDefault();
      openInlineEdit(inlineEditOpen.closest('[data-task-id]'));
      return;
    }

    var taskDetailClose = closestElement(event.target, '[data-task-detail-close]');
    if (taskDetailClose) {
      event.preventDefault();
      closeTaskDetail(taskDetailClose.closest('[data-task-detail-dialog]'));
      return;
    }

    var taskDetailOpen = closestElement(event.target, '[data-task-detail-open]');
    if (taskDetailOpen) {
      event.preventDefault();
      openTaskDetail(taskDetailOpen);
      return;
    }

    var taskCompleteClose = closestElement(event.target, '[data-task-complete-close]');
    if (taskCompleteClose) {
      event.preventDefault();
      closeTaskComplete(taskCompleteClose.closest('[data-task-complete-dialog]'));
      return;
    }

    var taskCompleteOpen = closestElement(event.target, '[data-task-complete-open]');
    if (taskCompleteOpen) {
      event.preventDefault();
      openTaskComplete(taskCompleteOpen);
      return;
    }

    var taskDeleteClose = closestElement(event.target, '[data-task-delete-close]');
    if (taskDeleteClose) {
      event.preventDefault();
      closeTaskDelete(taskDeleteClose.closest('[data-task-delete-dialog]'));
      return;
    }

    var taskDeleteOpen = closestElement(event.target, '[data-task-delete-open]');
    if (taskDeleteOpen) {
      event.preventDefault();
      openTaskDelete(taskDeleteOpen);
      return;
    }

    var undoAction = closestElement(event.target, '[data-undo-action]');
    if (undoAction) {
      event.preventDefault();
      executeUndo(undoAction);
      return;
    }

    var mobileNavOpen = closestElement(event.target, '[data-mobile-nav-open]');
    if (mobileNavOpen) {
      event.preventDefault();
      openMobileNav();
      return;
    }

    var mobileNavClose = closestElement(event.target, '[data-mobile-nav-close]');
    if (mobileNavClose) {
      event.preventDefault();
      closeMobileNav();
      return;
    }

    var mobileNavLink = closestElement(event.target, '[data-mobile-nav-dialog] a');
    if (mobileNavLink) {
      closeMobileNav();
      return;
    }

    var closeButton = closestElement(event.target, '[data-shortcut-help-close]');
    if (!closeButton) return;
    var dialog = closeButton.closest('dialog');
    if (dialog) {
      dialog.close();
    }
  }, true);

  window.addEventListener('resize', function () {
    if (window.innerWidth >= 768) {
      closeMobileNav();
    }
  });

  window.addEventListener('focus', scheduleFocusRefresh);
  document.addEventListener('visibilitychange', function () {
    if (document.visibilityState === 'visible') {
      scheduleFocusRefresh();
    }
  });

  document.addEventListener('keydown', function (event) {
    var inlineCreate = closestElement(event.target, '[data-inline-task-create]');
    if (inlineCreate && event.key === 'Escape') {
      event.preventDefault();
      closeInlineCreate(inlineCreate);
      return;
    }

    var inlineEdit = closestElement(event.target, '[data-task-id]');
    if (inlineEdit && inlineEdit.dataset.inlineTaskEditOpen === 'true' && event.key === 'Escape') {
      event.preventDefault();
      closeInlineEdit(inlineEdit);
      return;
    }

    var taskDetailDialog = closestElement(event.target, '[data-task-detail-dialog]');
    if (taskDetailDialog && taskDetailDialog.open && event.key === 'Escape' && typeof taskDetailDialog.close !== 'function') {
      event.preventDefault();
      closeTaskDetail(taskDetailDialog);
      return;
    }

    var taskCompleteDialog = closestElement(event.target, '[data-task-complete-dialog]');
    if (taskCompleteDialog && taskCompleteDialog.open && event.key === 'Escape' && typeof taskCompleteDialog.close !== 'function') {
      event.preventDefault();
      closeTaskComplete(taskCompleteDialog);
      return;
    }

    var taskDeleteDialog = closestElement(event.target, '[data-task-delete-dialog]');
    if (taskDeleteDialog && taskDeleteDialog.open && event.key === 'Escape' && typeof taskDeleteDialog.close !== 'function') {
      event.preventDefault();
      closeTaskDelete(taskDeleteDialog);
      return;
    }

    if (event.defaultPrevented || event.ctrlKey || event.altKey || event.metaKey) return;
    if (isTypingTarget(event.target)) {
      navState.pendingView = null;
      return;
    }

    var key = event.key.toLowerCase();
    var helpShortcut = key === '?' || (event.shiftKey && key === '/');
    if (navState.pendingView) {
      var viewPath = shortcutViewPath(key);
      if (viewPath) {
        event.preventDefault();
        goTo(viewPath);
      }
      navState.pendingView = null;
      return;
    }

    if (key === 'g') {
      navState.pendingView = 'goto';
      return;
    }

    if (key === 'n') {
      event.preventDefault();
      goTo('/quick-add');
      return;
    }

    if (helpShortcut) {
      event.preventDefault();
      toggleHelpDialog();
      return;
    }

    if (key === 's' || key === '/') {
      event.preventDefault();
      goTo('/search');
    }
  });

  initializeUndoSurface();
})();
