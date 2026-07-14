(function () {
  'use strict';

  var navState = { pendingView: null };
  var tabStateKey = '__caldoTabState';
  var tabState = loadTabState();
  var tabID = ensureTabID(tabState);
  var undoExpiryTimer = 0;
  var focusRefreshState = { inFlight: false, pending: false, lastRun: 0 };
  var taskMoveDragState = null;
  var editableControlDragRow = null;
  var explicitQuickAddPreviewRequests = new WeakSet();

  /* ── Toast notification system ───────────────────────────── */

  var toastDefaults = { duration: 3500, maxVisible: 4 };
  var toastTimers = [];

  function toastContainer() {
    var el = document.querySelector('[data-toast-container]');
    if (!el) {
      el = document.createElement('div');
      el.setAttribute('data-toast-container', '');
      el.setAttribute('aria-live', 'polite');
      el.setAttribute('aria-atomic', 'false');
      document.body.appendChild(el);
    }
    return el;
  }

  function dismissToast(item) {
    if (!item || !document.contains(item)) return;
    item.classList.add('caldo-toast-dismissing');
    setTimeout(function () { item.remove(); }, 200);
  }

  function showToast(message, kind, duration) {
    var container = toastContainer();
    kind = kind || 'info';
    duration = typeof duration === 'number' ? duration : toastDefaults.duration;

    var item = document.createElement('div');
    item.className = 'caldo-toast-item caldo-toast-' + kind;
    item.setAttribute('role', 'status');

    var text = document.createElement('span');
    text.textContent = message;
    item.appendChild(text);

    var dismiss = document.createElement('button');
    dismiss.type = 'button';
    dismiss.className = 'caldo-toast-dismiss';
    dismiss.setAttribute('aria-label', 'Schließen');
    dismiss.textContent = '×';
    dismiss.addEventListener('click', function () {
      clearTimeout(item.__caldoTimer);
      dismissToast(item);
    });
    item.appendChild(dismiss);

    container.appendChild(item);

    // Enforce maxVisible: remove oldest excess
    while (container.children.length > toastDefaults.maxVisible) {
      dismissToast(container.children[0]);
    }

    if (duration > 0) {
      item.__caldoTimer = setTimeout(function () { dismissToast(item); }, duration);
    }
    return item;
  }

  /* ── Bulk selection ──────────────────────────────────────── */

  var bulkSelection = { ids: [], lastClicked: null, running: false };

  function bulkSelectedSet() {
    var set = {};
    bulkSelection.ids.forEach(function (id) { set[id] = true; });
    return set;
  }

  function updateBulkSelectionVisuals() {
    var selected = bulkSelectedSet();
    taskRows().forEach(function (row) {
      var id = taskRowID(row);
      var isSelected = !!selected[id];
      row.classList.toggle('caldo-task-row-selected', isSelected);
      var control = row.querySelector('[data-bulk-select-control]');
      if (control) {
        control.checked = isSelected;
        control.disabled = bulkSelection.running;
      }
    });
    renderBulkBar();
  }

  function renderBulkBar() {
    var existing = document.querySelector('.caldo-bulk-bar');
    if (bulkSelection.ids.length === 0) {
      if (existing) existing.remove();
      return;
    }
    if (!existing) {
      existing = document.createElement('div');
      existing.className = 'caldo-bulk-bar';
      existing.setAttribute('role', 'region');
      existing.setAttribute('aria-label', 'Mehrfachbearbeitung');
      document.body.appendChild(existing);
    }
    var count = bulkSelection.ids.length;
    var disabled = bulkSelection.running ? ' disabled' : '';
    existing.innerHTML =
      '<span class="caldo-bulk-bar-count">' + count + ' Aufgabe' + (count === 1 ? '' : 'n') + ' ausgewählt</span>' +
      '<div class="caldo-bulk-bar-actions">' +
      '<button type="button" class="caldo-button caldo-button-ghost" data-bulk-select-all' + disabled + '>Alle abwählen</button>' +
      '<button type="button" class="caldo-button caldo-button-primary" data-bulk-complete' + disabled + '>' + (bulkSelection.running ? 'Speichern ...' : 'Erledigen') + '</button>' +
      '</div>';
  }

  function toggleBulkSelect(taskID) {
    var idx = bulkSelection.ids.indexOf(taskID);
    if (idx === -1) {
      bulkSelection.ids.push(taskID);
    } else {
      bulkSelection.ids.splice(idx, 1);
    }
    updateBulkSelectionVisuals();
  }

  function bulkSelectRange(fromID, toID) {
    var rows = taskRows().filter(function (row) {
      return !!row.querySelector('[data-bulk-select-control]');
    });
    var ids = rows.map(taskRowID);
    var startIdx = ids.indexOf(fromID);
    var endIdx = ids.indexOf(toID);
    if (startIdx === -1 || endIdx === -1) return;
    if (startIdx > endIdx) { var tmp = startIdx; startIdx = endIdx; endIdx = tmp; }
    for (var i = startIdx; i <= endIdx; i++) {
      if (bulkSelection.ids.indexOf(ids[i]) === -1) {
        bulkSelection.ids.push(ids[i]);
      }
    }
    updateBulkSelectionVisuals();
  }

  function clearBulkSelection() {
    if (bulkSelection.running) return;
    bulkSelection.ids = [];
    bulkSelection.lastClicked = null;
    updateBulkSelectionVisuals();
  }

  function bulkCompleteTasks() {
    if (bulkSelection.ids.length === 0 || bulkSelection.running) return;
    var ids = bulkSelection.ids.slice();
    bulkSelection.running = true;
    updateBulkSelectionVisuals();
    var requests = ids.map(function (taskID) {
      var row = document.querySelector('[data-task-id="' + taskID + '"]');
      if (!row) return Promise.resolve({ taskID: taskID, ok: false });
      var form = row.querySelector('form[data-task-action-form]');
      if (!form) return Promise.resolve({ taskID: taskID, ok: false });
      var versionInput = form.querySelector('input[name="expected_version"]');
      var params = new URLSearchParams();
      params.set('expected_version', versionInput ? versionInput.value : '');
      return fetch(form.getAttribute('action'), {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded', 'Accept': 'text/html', 'X-CSRF-Token': csrfToken(), 'X-Tab-ID': tabID },
        body: params.toString()
      }).then(function (response) {
        return { taskID: taskID, ok: response.ok, row: row };
      }).catch(function () {
        return { taskID: taskID, ok: false, row: row };
      });
    });
    Promise.all(requests).then(function (results) {
      var succeeded = results.filter(function (result) { return result.ok; });
      var failed = results.filter(function (result) { return !result.ok; });
      succeeded.forEach(function (result) {
        if (result.row && document.contains(result.row)) result.row.remove();
      });
      bulkSelection.running = false;
      bulkSelection.ids = failed.map(function (result) { return result.taskID; });
      bulkSelection.lastClicked = bulkSelection.ids.length ? bulkSelection.ids[bulkSelection.ids.length - 1] : null;
      updateBulkSelectionVisuals();

      if (succeeded.length > 0) refreshUndoNotification();
      if (failed.length > 0) {
        var failedMessage = failed.length === 1 ? '1 Aufgabe konnte nicht erledigt werden.' : failed.length + ' Aufgaben konnten nicht erledigt werden.';
        if (succeeded.length > 0) {
          failedMessage = succeeded.length + ' Aufgabe' + (succeeded.length === 1 ? '' : 'n') + ' erledigt, ' + failedMessage;
        }
        showToast(failedMessage, 'error');
        return;
      }
      showToast(succeeded.length + ' Aufgabe' + (succeeded.length === 1 ? '' : 'n') + ' erledigt.', 'success');
    });
  }

  function activateBulkSelection(row, extendRange) {
    var taskID = taskRowID(row);
    if (!taskID || !row.querySelector('[data-bulk-select-control]') || bulkSelection.running) return false;
    if (extendRange && bulkSelection.lastClicked) {
      bulkSelectRange(bulkSelection.lastClicked, taskID);
    } else {
      toggleBulkSelect(taskID);
    }
    bulkSelection.lastClicked = taskID;
    return true;
  }

  function handleBulkSelectionClick(event) {
    var clearButton = closestElement(event.target, '[data-bulk-select-all]');
    if (clearButton) {
      event.preventDefault();
      clearBulkSelection();
      return true;
    }
    var completeButton = closestElement(event.target, '[data-bulk-complete]');
    if (completeButton) {
      event.preventDefault();
      bulkCompleteTasks();
      return true;
    }

    var row = closestElement(event.target, '[data-task-id]');
    if (!row) return false;
    var explicitControl = closestElement(event.target, '[data-bulk-select-control]');
    var modifierSelection = event.shiftKey || event.ctrlKey || event.metaKey;
    if (explicitControl && row.querySelector('[data-bulk-select-control]')) {
      activateBulkSelection(row, event.shiftKey);
      return true;
    }
    if (modifierSelection && row.querySelector('[data-bulk-select-control]')) {
      event.preventDefault();
      activateBulkSelection(row, event.shiftKey);
      return true;
    }
    if (bulkSelection.ids.length > 0) clearBulkSelection();
    return false;
  }

  function handleBulkSelectionKeydown(event) {
    var control = closestElement(event.target, '[data-bulk-select-control]');
    if (!control || (event.key !== ' ' && event.key !== 'Enter')) return;
    event.preventDefault();
    event.stopPropagation();
    activateBulkSelection(control.closest('[data-task-id]'), event.shiftKey);
  }

  /* ── Task hover preview ──────────────────────────────────── */

  var hoverPreviewState = { el: null, hideTimer: null, fetchTimer: null };

  function showTaskHoverPreview(row, event) {
    if (!row || hoverPreviewState.el) return;
    var taskID = taskRowID(row);
    if (!taskID) return;

    // First try inline description
    var desc = row.querySelector('.caldo-task-description');
    var text = desc ? (desc.textContent || '').trim() : '';

    // If no inline description, fetch via AJAX
    if (!text) {
      if (hoverPreviewState.fetchTimer) clearTimeout(hoverPreviewState.fetchTimer);
      hoverPreviewState.fetchTimer = setTimeout(function () {
        fetchTaskPreview(taskID, row);
      }, 600);
      return;
    }

    renderHoverPreview(row, 'Beschreibung', desc);
  }

  function fetchTaskPreview(taskID, row) {
    fetch('/tasks/' + encodeURIComponent(taskID), {
      method: 'GET',
      credentials: 'same-origin',
      headers: { 'Accept': 'text/html', 'X-Tab-ID': tabID }
    }).then(function (r) { return r.ok ? r.text() : ''; }).then(function (html) {
      if (!html) return;
      var tpl = document.createElement('template');
      tpl.innerHTML = html.trim();
      var desc = tpl.content.querySelector('.caldo-task-description');
      var text = desc ? (desc.textContent || '').trim() : '';
      if (!text || !document.contains(row)) return;
      renderHoverPreview(row, 'Beschreibung', desc);
    }).catch(function () {});
  }

  function cloneRenderedDescription(target, source) {
    if (!target || !source) return false;
    target.replaceChildren();
    Array.prototype.forEach.call(source.childNodes, function (node) {
      target.appendChild(node.cloneNode(true));
    });
    return true;
  }

  function renderHoverPreview(row, label, description) {
    if (hoverPreviewState.el) return;
    var preview = document.createElement('div');
    preview.className = 'caldo-task-hover-preview';
    var title = document.createElement('div');
    title.className = 'caldo-task-hover-preview-title';
    title.textContent = label;
    var body = document.createElement('div');
    body.className = 'caldo-task-hover-preview-description';
    cloneRenderedDescription(body, description);
    preview.appendChild(title);
    preview.appendChild(body);
    row.classList.add('caldo-task-row-relative');
    row.appendChild(preview);
    hoverPreviewState.el = preview;
  }

  function hideTaskHoverPreview() {
    if (hoverPreviewState.hideTimer) {
      clearTimeout(hoverPreviewState.hideTimer);
      hoverPreviewState.hideTimer = null;
    }
    if (hoverPreviewState.fetchTimer) {
      clearTimeout(hoverPreviewState.fetchTimer);
      hoverPreviewState.fetchTimer = null;
    }
    if (hoverPreviewState.el && document.contains(hoverPreviewState.el)) {
      hoverPreviewState.el.remove();
    }
    hoverPreviewState.el = null;
  }

  /* ── Skeleton loader for navigation ──────────────────────── */

  function showNavigationSkeleton() {
    var main = document.querySelector('.caldo-content');
    if (!main) return;
    main.setAttribute('data-nav-skeleton', 'true');
    main.innerHTML =
      '<section class="caldo-state">' +
      '<div class="caldo-loading-lines" aria-hidden="true">' +
      '<span></span><span></span><span></span><span></span><span></span>' +
      '</div></section>';
  }

  /* ── Task row insertion animation ────────────────────────── */

  function animateNewTaskRow(row) {
    if (!row) return;
    row.classList.add('caldo-task-row-enter');
    setTimeout(function () { row.classList.remove('caldo-task-row-enter'); }, 250);
  }

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

  var focusableSelector = 'button, [href], input, select, textarea, summary, [tabindex]:not([tabindex="-1"])';
  var formErrorIDSequence = 0;

  function elementIsFocusable(element) {
    if (!(element instanceof Element)) return false;
    if (element.hidden || element.disabled || element.getAttribute('aria-hidden') === 'true') return false;
    var style = window.getComputedStyle ? window.getComputedStyle(element) : null;
    if (style && (style.display === 'none' || style.visibility === 'hidden')) return false;
    return element.getClientRects().length > 0 || element === document.activeElement;
  }

  function focusableControls(root) {
    if (!root || !root.querySelectorAll) return [];
    return Array.prototype.slice.call(root.querySelectorAll(focusableSelector)).filter(elementIsFocusable);
  }

  function focusFirstDialogControl(dialog, preferredSelector) {
    if (!dialog) return;
    var preferred = preferredSelector ? dialog.querySelector(preferredSelector) : null;
    var target = elementIsFocusable(preferred) ? preferred : focusableControls(dialog)[0];
    if (!target || typeof target.focus !== 'function') return;
    window.setTimeout(function () {
      if (document.contains(target)) {
        target.focus();
      }
    }, 0);
  }

  function topOpenDialog() {
    var dialogs = document.querySelectorAll('dialog[open]');
    return dialogs.length > 0 ? dialogs[dialogs.length - 1] : null;
  }

  function trapFocusInDialog(dialog, event) {
    if (!dialog || event.key !== 'Tab') return false;
    var controls = focusableControls(dialog);
    if (controls.length === 0) {
      event.preventDefault();
      return true;
    }

    var first = controls[0];
    var last = controls[controls.length - 1];
    if (!dialog.contains(document.activeElement)) {
      event.preventDefault();
      first.focus();
      return true;
    }

    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
      return true;
    }

    if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
      return true;
    }

    return false;
  }

  function showDialog(dialog) {
    if (!dialog) return;
    if (typeof dialog.showModal === 'function') {
      if (!dialog.open) {
        dialog.showModal();
      }
      return;
    }
    dialog.setAttribute('open', '');
  }

  function closeDialog(dialog) {
    if (!dialog) return;
    if (typeof dialog.close === 'function' && dialog.open) {
      dialog.close();
      return;
    }
    dialog.removeAttribute('open');
    returnDialogFocus(dialog);
  }

  function returnDialogFocus(dialog) {
    if (!dialog) return;
    var trigger = dialog.__caldoReturnFocus;
    dialog.__caldoReturnFocus = null;
    if (trigger && document.contains(trigger) && typeof trigger.focus === 'function' && elementIsFocusable(trigger)) {
      trigger.focus();
    }
  }

  function tokenList(value) {
    return String(value || '').split(/\s+/).filter(Boolean);
  }

  function addTokenAttribute(element, name, token) {
    if (!element || !token) return;
    var tokens = tokenList(element.getAttribute(name));
    if (tokens.indexOf(token) === -1) {
      tokens.push(token);
      element.setAttribute(name, tokens.join(' '));
    }
  }

  function ensureElementID(element, prefix) {
    if (!element) return '';
    if (element.id) return element.id;
    formErrorIDSequence += 1;
    element.id = (prefix || 'caldo-error') + '-' + formErrorIDSequence;
    return element.id;
  }

  function firstFormControl(form) {
    if (!form || !form.querySelectorAll) return null;
    var controls = Array.prototype.slice.call(form.querySelectorAll('input:not([type="hidden"]), select, textarea'));
    for (var i = 0; i < controls.length; i += 1) {
      if (elementIsFocusable(controls[i]) || (!controls[i].hidden && !controls[i].disabled)) {
        return controls[i];
      }
    }
    return null;
  }

  function formForError(error) {
    if (!error) return null;
    var form = error.closest('form');
    if (form) return form;
    if (error.matches && error.matches('[data-quick-add-overlay-error]')) {
      var overlay = error.closest('[data-quick-add-overlay]');
      if (overlay) {
        return overlay.querySelector('[data-quick-add-save-form]') || overlay.querySelector('[data-quick-add-overlay-form]');
      }
    }
    var dialog = error.closest('[data-task-detail-dialog], [data-task-complete-dialog], [data-task-delete-dialog]');
    if (dialog) {
      return dialog.querySelector('form');
    }
    var sibling = error.previousElementSibling;
    while (sibling) {
      if (sibling.matches && sibling.matches('form')) return sibling;
      sibling = sibling.previousElementSibling;
    }
    var container = error.closest('[data-quick-add-overlay], [data-conflict-resolution], [data-setup-import], [data-task-id], .caldo-card, .caldo-page');
    return container ? container.querySelector('form') : null;
  }

  function exposeFormError(error) {
    if (!error) return;
    error.setAttribute('role', 'alert');
    if (!error.hasAttribute('aria-live')) {
      error.setAttribute('aria-live', 'polite');
    }
    var errorID = ensureElementID(error, 'caldo-form-error');
    var form = formForError(error);
    if (!form) return;
    addTokenAttribute(form, 'aria-describedby', errorID);
    var control = firstFormControl(form);
    if (control) {
      addTokenAttribute(control, 'aria-describedby', errorID);
      if (!error.hidden && String(error.textContent || '').trim() !== '') {
        control.setAttribute('aria-invalid', 'true');
      } else if (control.getAttribute('aria-invalid') === 'true') {
        control.setAttribute('aria-invalid', 'false');
      }
    }
  }

  function setErrorMessage(error, message) {
    if (!error) return;
    exposeFormError(error);
    if (!message) {
      error.textContent = '';
      error.hidden = true;
      var form = formForError(error);
      var control = firstFormControl(form);
      if (control && control.getAttribute('aria-invalid') === 'true') {
        control.setAttribute('aria-invalid', 'false');
      }
      return;
    }
    error.textContent = message;
    error.hidden = false;
    exposeFormError(error);
  }

  function initializeFormErrors(root) {
    var scope = root && root.querySelectorAll ? root : document;
    var errors = [];
    if (scope instanceof Element && scope.matches('.caldo-alert-error')) {
      errors.push(scope);
    }
    Array.prototype.push.apply(errors, scope.querySelectorAll('.caldo-alert-error'));
    errors.forEach(exposeFormError);
  }

  function isSyncRequestElement(target) {
    return !!closestElement(target, '[data-sync-request]');
  }

  function submitForm(form) {
    if (!form) return;
    if (typeof form.requestSubmit === 'function') {
      form.requestSubmit();
      return;
    }
    form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
  }

  function normalizeThemeMode(mode) {
    return mode === 'light' || mode === 'dark' || mode === 'system' ? mode : 'system';
  }

  function systemThemeMode() {
    if (!window.matchMedia) return 'light';
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }

  function effectiveThemeMode(mode) {
    mode = normalizeThemeMode(mode);
    return mode === 'system' ? systemThemeMode() : mode;
  }

  function applyThemeMode(mode) {
    var root = document.documentElement;
    if (!root || !root.hasAttribute('data-theme-root')) return 'system';
    mode = normalizeThemeMode(mode);
    root.setAttribute('data-theme-mode', mode);
    root.classList.remove('light', 'dark');
    if (mode === 'light' || mode === 'dark') {
      root.classList.add(mode);
    }
    root.setAttribute('data-theme-effective', effectiveThemeMode(mode));
    refreshThemeToggles(mode);
    return mode;
  }

  function themeModeLabel(button, mode) {
    var key = 'data-theme-' + normalizeThemeMode(mode) + '-label';
    return button.getAttribute(key) || normalizeThemeMode(mode);
  }

  function themeToggleLabel(button, mode) {
    var prefix = button.getAttribute('data-theme-label-prefix') || 'Darstellung';
    return prefix + ': ' + themeModeLabel(button, mode);
  }

  function refreshThemeToggles(mode) {
    Array.prototype.forEach.call(document.querySelectorAll('[data-theme-toggle]'), function (button) {
      var label = themeToggleLabel(button, mode);
      button.setAttribute('data-theme-mode', mode);
      button.setAttribute('aria-label', label);
      button.setAttribute('title', label);
      var currentLabel = button.querySelector('[data-theme-current-label]');
      if (currentLabel) {
        currentLabel.textContent = label;
      } else {
        button.textContent = label;
      }
    });
  }

  function nextThemeMode(mode) {
    switch (normalizeThemeMode(mode)) {
      case 'dark':
        return 'light';
      case 'light':
        return 'system';
      default:
        return 'dark';
    }
  }

  function initializeThemeController() {
    var root = document.documentElement;
    if (!root || !root.hasAttribute('data-theme-root')) return;
    applyThemeMode(root.getAttribute('data-theme-mode') || 'system');

    if (!window.matchMedia) return;
    var systemPreference = window.matchMedia('(prefers-color-scheme: dark)');
    var handleSystemChange = function () {
      if (normalizeThemeMode(root.getAttribute('data-theme-mode')) === 'system') {
        applyThemeMode('system');
      }
    };
    if (typeof systemPreference.addEventListener === 'function') {
      systemPreference.addEventListener('change', handleSystemChange);
      return;
    }
    if (typeof systemPreference.addListener === 'function') {
      systemPreference.addListener(handleSystemChange);
    }
  }

  function toggleHelpDialog() {
    var dialog = document.querySelector('[data-shortcut-help-dialog]');
    if (!dialog) return;
    if (dialog.hasAttribute('open')) {
      closeDialog(dialog);
      return;
    }
    dialog.__caldoReturnFocus = document.activeElement;
    showDialog(dialog);
    focusFirstDialogControl(dialog, '[data-shortcut-help-close]');
  }

  function openMobileNav(trigger) {
    var dialog = document.querySelector('[data-mobile-nav-dialog]');
    if (!dialog || dialog.hasAttribute('open')) return;
    dialog.__caldoReturnFocus = trigger || document.activeElement;
    if (trigger) {
      trigger.setAttribute('aria-expanded', 'true');
    }
    showDialog(dialog);
    focusFirstDialogControl(dialog, '[data-mobile-nav-close]');
  }

  function closeMobileNav() {
    var dialog = document.querySelector('[data-mobile-nav-dialog]');
    if (!dialog || !dialog.hasAttribute('open')) return;
    closeDialog(dialog);
  }

  function bindApplicationDialogs() {
    var mobileNav = document.querySelector('[data-mobile-nav-dialog]');
    if (mobileNav && mobileNav.dataset.mobileNavBound !== 'true') {
      mobileNav.dataset.mobileNavBound = 'true';
      mobileNav.addEventListener('close', function () {
        var trigger = mobileNav.__caldoReturnFocus;
        if (trigger && document.contains(trigger)) {
          trigger.setAttribute('aria-expanded', 'false');
        }
        returnDialogFocus(mobileNav);
      });
      mobileNav.addEventListener('click', function (event) {
        if (event.target === mobileNav) {
          closeMobileNav();
        }
      });
    }

    var help = document.querySelector('[data-shortcut-help-dialog]');
    if (help && help.dataset.shortcutHelpBound !== 'true') {
      help.dataset.shortcutHelpBound = 'true';
      help.addEventListener('close', function () {
        returnDialogFocus(help);
      });
    }
  }

  function quickAddOverlay() {
    return document.querySelector('[data-quick-add-overlay]');
  }

  function quickAddOverlayError(dialog) {
    return dialog ? dialog.querySelector('[data-quick-add-overlay-error]') : null;
  }

  function setQuickAddOverlayError(dialog, message) {
    var error = quickAddOverlayError(dialog);
    setErrorMessage(error, message);
  }

  function resetQuickAddOverlay(dialog) {
    if (!dialog) return;
    var form = dialog.querySelector('[data-quick-add-overlay-form]');
    var preview = dialog.querySelector('#quick-add-overlay-preview');
    if (form) {
      form.reset();
    }
    if (preview) {
      preview.outerHTML = '<div id="quick-add-overlay-preview" class="caldo-quick-add-preview-shell"></div>';
    }
    setQuickAddOverlayError(dialog, '');
  }

  function hideElement(element) {
    if (!element) return;
    element.hidden = true;
    element.classList.add('hidden');
  }

  function openQuickAddOverlay(trigger) {
    var dialog = quickAddOverlay();
    if (!dialog) {
      goTo('/quick-add');
      return;
    }
    dialog.__caldoReturnFocus = trigger || document.activeElement;
    setQuickAddOverlayError(dialog, '');
    showDialog(dialog);
    var input = dialog.querySelector('[data-quick-add-overlay-input]');
    window.setTimeout(function () {
      if (input) {
        input.focus();
        input.select();
      }
    }, 0);
  }

  function closeQuickAddOverlay(dialog, reset) {
    if (!dialog) return;
    if (reset) {
      resetQuickAddOverlay(dialog);
    }
    closeDialog(dialog);
  }

  function quickAddFocusableControls(root) {
    if (!root) return [];
    return Array.prototype.slice.call(root.querySelectorAll('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])')).filter(function (element) {
      return !element.disabled && !element.hidden && element.offsetParent !== null;
    });
  }

  function focusQuickAddSaveStart(root) {
    if (!root) return false;
    var firstCorrection = root.querySelector('[data-quick-add-correction]');
    var target = firstCorrection || quickAddFocusableControls(root)[0];
    if (!target || typeof target.focus !== 'function') return false;
    window.setTimeout(function () {
      if (!document.contains(target)) return;
      target.focus();
      if (typeof target.select === 'function' && target.tagName === 'INPUT' && target.type !== 'date') {
        target.select();
      }
    }, 0);
    return true;
  }

  function focusQuickAddPreviewResult(requestElement, targetElement) {
    var targetID = targetElement && targetElement.id ? targetElement.id : '';
    var overlayRequest = closestElement(requestElement, '[data-quick-add-overlay]');
    if (overlayRequest && (targetID === 'quick-add-overlay-preview' || closestElement(requestElement, '[data-quick-add-overlay-form]'))) {
      if (overlayRequest.open) {
        return focusQuickAddSaveStart(overlayRequest.querySelector('[data-quick-add-save-form]'));
      }
      return false;
    }

    var pagePreviewForm = closestElement(requestElement, '.caldo-quick-add-form');
    if (pagePreviewForm && (targetID === 'quick-add-preview' || pagePreviewForm.getAttribute('hx-target') === '#quick-add-preview')) {
      return focusQuickAddSaveStart(document.querySelector('#quick-add-preview [data-quick-add-save-form]'));
    }
    return false;
  }

  function rememberQuickAddPreviewFocusIntent(detail) {
    if (!detail || !detail.xhr) return;
    var previewForm = closestElement(detail.elt, '.caldo-quick-add-form');
    if (!previewForm || previewForm.getAttribute('action') !== '/quick-add/preview') return;
    var requestConfig = detail.requestConfig || {};
    var triggeringEvent = requestConfig.triggeringEvent || detail.triggeringEvent;
    if (triggeringEvent && triggeringEvent.type === 'submit') {
      explicitQuickAddPreviewRequests.add(detail.xhr);
    }
  }

  function pad2(value) {
    return String(value).padStart(2, '0');
  }

  function formatBrowserDateTimeFromISO(isoValue) {
    var date = new Date(isoValue || '');
    if (Number.isNaN(date.getTime())) return '';
    return pad2(date.getDate()) + '.' + pad2(date.getMonth() + 1) + '.' + date.getFullYear() + ' ' + pad2(date.getHours()) + ':' + pad2(date.getMinutes());
  }

  function initializeLocalDateTimes(root) {
    var scope = root && root.querySelectorAll ? root : document;
    var nodes = [];
    if (scope instanceof Element && scope.matches('[data-local-date-time]')) {
      nodes.push(scope);
    }
    Array.prototype.push.apply(nodes, scope.querySelectorAll('[data-local-date-time]'));
    Array.prototype.forEach.call(nodes, function (node) {
      var formatted = formatBrowserDateTimeFromISO(node.getAttribute('datetime') || '');
      if (formatted) {
        node.textContent = formatted;
      }
    });
    refreshSyncTooltips(scope);
  }

  function refreshSyncTooltips(root) {
    var scope = root && root.querySelectorAll ? root : document;
    var targets = [];
    if (scope instanceof Element && scope.matches('[data-sync-tooltip-template]')) {
      targets.push(scope);
    }
    Array.prototype.push.apply(targets, scope.querySelectorAll('[data-sync-tooltip-template]'));
    Array.prototype.forEach.call(targets, function (target) {
      var template = target.getAttribute('data-sync-tooltip-template') || '';
      var lastSync = target.querySelector('[data-sync-last-value]');
      if (!template || !lastSync) return;
      var label = template.replace('{last}', (lastSync.textContent || '').trim());
      target.setAttribute('aria-label', label);
      target.setAttribute('title', label);
    });
  }

  function requestManualSync() {
    var syncButton = document.querySelector('button[data-sync-request]');
    if (!syncButton || typeof syncButton.click !== 'function') return;
    syncButton.click();
  }

  function openFocusedTaskDateDropdown() {
    var focused = document.activeElement;
    if (!focused) return;
    var taskRow = closestElement(focused, '[data-task-id]');
    if (!taskRow) return;
    var dateDropdown = taskRow.querySelector('.caldo-date-dropdown');
    if (!dateDropdown) return;
    openDateDropdown(dateDropdown, true);
  }

  function bindQuickAddOverlay() {
    var dialog = quickAddOverlay();
    if (!dialog || dialog.dataset.quickAddOverlayBound === 'true') return;
    dialog.dataset.quickAddOverlayBound = 'true';
    dialog.addEventListener('close', function () {
      resetQuickAddOverlay(dialog);
      hideQuickAddTokenDropdown();
      var trigger = dialog.__caldoReturnFocus;
      dialog.__caldoReturnFocus = null;
      if (trigger && document.contains(trigger)) {
        trigger.focus();
      }
    });
    dialog.addEventListener('click', function (event) {
      if (event.target === dialog) {
        closeQuickAddOverlay(dialog, true);
      }
    });

    // Load suggestions and bind input handler for # and @ dropdowns
    var input = dialog.querySelector('[data-quick-add-overlay-input]');
    if (input) {
      quickAddLoadSuggestions(input);
      input.addEventListener('input', quickAddTokenInputHandler);
      input.addEventListener('keydown', quickAddTokenKeydownHandler);
    }
  }

  /* ── Quick Add # and @ token dropdown ────────────────────── */

  var quickAddSuggestionCache = null;
  var quickAddTokenDropdownState = { el: null, input: null, kind: null, triggerIdx: -1, items: [], activeIndex: -1 };

  function quickAddLoadSuggestions(input) {
    if (quickAddSuggestionCache) return;
    fetch('/quick-add/suggestions', {
      credentials: 'same-origin',
      headers: { 'Accept': 'application/json', 'X-Tab-ID': tabID }
    }).then(function (r) { return r.ok ? r.json() : null; }).then(function (data) {
      quickAddSuggestionCache = data || { projects: [], labels: [] };
      if (document.activeElement === input) {
        quickAddTokenInputHandler({ target: input });
      }
    }).catch(function () {
      quickAddSuggestionCache = { projects: [], labels: [] };
    });
  }

  function quickAddTokenInputHandler(event) {
    var input = event.target;
    if (!input || !input.matches || !input.matches('[data-quick-add-overlay-input]')) return;
    var value = input.value;
    var cursor = input.selectionStart || value.length;

    // Find the token trigger (# or @) before the cursor
    var before = value.slice(0, cursor);
    var hashIdx = before.lastIndexOf('#');
    var atIdx = before.lastIndexOf('@');
    var triggerChar = null;
    var triggerIdx = -1;
    if (hashIdx > atIdx) { triggerChar = '#'; triggerIdx = hashIdx; }
    else if (atIdx > hashIdx) { triggerChar = '@'; triggerIdx = atIdx; }

    if (!triggerChar || triggerIdx < 0) {
      hideQuickAddTokenDropdown();
      return;
    }

    // Check that there's no space between trigger and cursor (simple token mode)
    var afterTrigger = before.slice(triggerIdx + 1);
    if (/\s/.test(afterTrigger)) {
      hideQuickAddTokenDropdown();
      return;
    }

    var query = afterTrigger.toLowerCase();
    var suggestions = [];
    if (quickAddSuggestionCache) {
      if (triggerChar === '#') {
        suggestions = (quickAddSuggestionCache.projects || []).filter(function (p) {
          return p.name && p.name.toLowerCase().indexOf(query) !== -1;
        }).slice(0, 8);
      } else {
        suggestions = (quickAddSuggestionCache.labels || []).filter(function (l) {
          return l.name && l.name.toLowerCase().indexOf(query) !== -1;
        }).slice(0, 8);
      }
    }

    if (suggestions.length === 0) {
      hideQuickAddTokenDropdown();
      return;
    }

    showQuickAddTokenDropdown(input, triggerChar, triggerIdx, suggestions);
  }

  function showQuickAddTokenDropdown(input, kind, triggerIdx, suggestions) {
    hideQuickAddTokenDropdown();

    var dropdown = document.createElement('div');
    dropdown.className = 'caldo-quick-add-token-dropdown';
    dropdown.id = input.getAttribute('aria-controls') || 'quick-add-token-suggestions';
    dropdown.setAttribute('role', 'listbox');

    var items = [];

    suggestions.forEach(function (s, index) {
      var item = document.createElement('button');
      item.type = 'button';
      item.className = 'caldo-quick-add-token-item';
      item.id = dropdown.id + '-option-' + index;
      item.setAttribute('role', 'option');
      item.setAttribute('aria-selected', 'false');
      item.setAttribute('tabindex', '-1');
      item.setAttribute('data-quick-add-token-name', s.name);
      item.textContent = (kind === '#' ? '#' : '@') + s.name;
      if (s.id) {
        var badge = document.createElement('span');
        badge.className = 'caldo-quick-add-token-item-badge';
        badge.textContent = s.id;
        item.appendChild(badge);
      }
      item.addEventListener('click', function (event) {
        event.preventDefault();
        event.stopPropagation();
        insertQuickAddToken(item);
      });
      dropdown.appendChild(item);
      items.push(item);
    });

    var wrapper = input.parentElement;
    if (wrapper) {
      wrapper.classList.add('caldo-token-input-wrapper');
      wrapper.appendChild(dropdown);
    }
    quickAddTokenDropdownState = {
      el: dropdown,
      input: input,
      kind: kind,
      triggerIdx: triggerIdx,
      items: items,
      activeIndex: -1
    };
    input.setAttribute('aria-expanded', 'true');
    setQuickAddTokenActiveIndex(0);
  }

  function setQuickAddTokenActiveIndex(index) {
    var state = quickAddTokenDropdownState;
    if (!state.input || !state.items.length) return;
    var boundedIndex = Math.max(0, Math.min(index, state.items.length - 1));
    state.items.forEach(function (item, itemIndex) {
      var active = itemIndex === boundedIndex;
      item.classList.toggle('caldo-quick-add-token-item-active', active);
      item.setAttribute('aria-selected', String(active));
    });
    state.activeIndex = boundedIndex;
    state.input.setAttribute('aria-activedescendant', state.items[boundedIndex].id);
    state.items[boundedIndex].scrollIntoView({ block: 'nearest' });
  }

  function insertQuickAddToken(item) {
    var state = quickAddTokenDropdownState;
    if (!item || !state.input || !state.el || !state.el.contains(item)) return;
    var input = state.input;
    var name = item.getAttribute('data-quick-add-token-name') || '';
    var val = input.value;
    var cursor = input.selectionStart === null ? val.length : input.selectionStart;
    var before = val.slice(0, state.triggerIdx);
    var after = val.slice(cursor);
    var insert = state.kind + name + ' ';
    var newPos = before.length + insert.length;
    input.value = before + insert + after;
    hideQuickAddTokenDropdown();
    input.focus();
    input.setSelectionRange(newPos, newPos);
    input.dispatchEvent(new Event('input', { bubbles: true }));
  }

  function quickAddTokenKeydownHandler(event) {
    var state = quickAddTokenDropdownState;
    if (!state.el || state.input !== event.target || !state.items.length) return;
    var nextIndex = state.activeIndex;
    if (event.key === 'ArrowDown') {
      nextIndex = (state.activeIndex + 1) % state.items.length;
    } else if (event.key === 'ArrowUp') {
      nextIndex = (state.activeIndex - 1 + state.items.length) % state.items.length;
    } else if (event.key === 'Home') {
      nextIndex = 0;
    } else if (event.key === 'End') {
      nextIndex = state.items.length - 1;
    } else if (event.key === 'Enter') {
      event.preventDefault();
      event.stopPropagation();
      insertQuickAddToken(state.items[state.activeIndex]);
      return;
    } else if (event.key === 'Escape') {
      event.preventDefault();
      event.stopPropagation();
      hideQuickAddTokenDropdown();
      return;
    } else {
      return;
    }
    event.preventDefault();
    event.stopPropagation();
    setQuickAddTokenActiveIndex(nextIndex);
  }

  function hideQuickAddTokenDropdown() {
    var input = quickAddTokenDropdownState.input;
    if (quickAddTokenDropdownState.el && document.contains(quickAddTokenDropdownState.el)) {
      quickAddTokenDropdownState.el.remove();
    }
    if (input) {
      input.setAttribute('aria-expanded', 'false');
      input.removeAttribute('aria-activedescendant');
    }
    quickAddTokenDropdownState = { el: null, input: null, kind: null, triggerIdx: -1, items: [], activeIndex: -1 };
  }

  function clearQuickAddControl(button) {
    if (!button) return;
    var selector = button.getAttribute('data-quick-add-clear') || '';
    if (!selector) return;
    var preview = button.closest('[data-quick-add-preview]');
    var form = button.closest('[data-quick-add-save-form]');
    if (!form && preview) {
      form = preview.querySelector('[data-quick-add-save-form]');
    }
    if (!form) return;
    var controls = form.querySelectorAll(selector);
    if (!controls.length) return;

    Array.prototype.forEach.call(controls, function (control) {
      if (control.type === 'checkbox' || control.type === 'radio') {
        control.checked = false;
      } else {
        control.value = '';
      }
      control.dispatchEvent(new Event('input', { bubbles: true }));
      control.dispatchEvent(new Event('change', { bubbles: true }));
    });

    if (selector.indexOf('project_new_name') !== -1) {
      Array.prototype.forEach.call(form.querySelectorAll('[name="project_selection"]'), function (selection) {
        if (selection.type === 'radio') {
          selection.checked = false;
        } else {
          selection.value = '';
        }
        selection.dispatchEvent(new Event('input', { bubbles: true }));
        selection.dispatchEvent(new Event('change', { bubbles: true }));
      });
    }

    var chip = button.closest('.caldo-quick-add-chip');
    if (chip && chip.closest('[data-quick-add-chips]')) {
      hideElement(chip);
      var chipList = chip.closest('[data-quick-add-chips]');
      if (chipList && !chipList.querySelector('.caldo-quick-add-chip:not([hidden])')) {
        hideElement(chipList);
      }
    }

    if (typeof controls[0].focus === 'function') {
      controls[0].focus();
    }
  }

  function showQuickAddCorrections(element) {
    if (!element) return;
    var preview = element.closest('[data-quick-add-preview]');
    if (!preview) return;
    var corrections = preview.querySelector('[data-quick-add-corrections]');
    if (corrections && corrections.hidden) {
      corrections.hidden = false;
    }
  }

  function toggleQuickAddCorrections(trigger) {
    if (!trigger) return;
    var preview = trigger.closest('[data-quick-add-preview]');
    if (!preview) {
      var overlay = trigger.closest('[data-quick-add-overlay]');
      preview = overlay ? overlay.querySelector('[data-quick-add-preview]') : null;
    }
    if (!preview) return;
    var corrections = preview.querySelector('[data-quick-add-corrections]');
    if (!corrections) return;
    corrections.hidden = !corrections.hidden;
    if (!corrections.hidden) {
      var firstInput = corrections.querySelector('input:not([type="hidden"]), select');
      if (firstInput && typeof firstInput.focus === 'function') {
        window.setTimeout(function () { firstInput.focus(); }, 0);
      }
    }
  }

  function quickAddSaveFormFor(element) {
    if (!element) return null;
    var form = element.closest('[data-quick-add-save-form]');
    if (form) return form;
    var preview = element.closest('[data-quick-add-preview]');
    return preview ? preview.querySelector('[data-quick-add-save-form]') : null;
  }

  function requestQuickAddSubmit(form) {
    if (!form) return false;
    if (typeof form.requestSubmit === 'function') {
      form.requestSubmit();
      return true;
    }
    var submit = form.querySelector('button[type="submit"], input[type="submit"]');
    if (submit && typeof submit.click === 'function') {
      submit.click();
      return true;
    }
    return false;
  }

  function submitQuickAddKeyboardStep(dialog) {
    if (!dialog) return false;
    var saveForm = dialog.querySelector('[data-quick-add-save-form]');
    if (saveForm) {
      return requestQuickAddSubmit(saveForm);
    }
    return requestQuickAddSubmit(dialog.querySelector('[data-quick-add-overlay-form]'));
  }

  function quickAddTokenListFor(target) {
    return closestElement(target, '[data-quick-add-chips], [data-quick-add-label-suggestions]');
  }

  function quickAddTokenButtons(list) {
    if (!list) return [];
    return Array.prototype.slice.call(list.querySelectorAll('button')).filter(function (button) {
      return !button.disabled && !button.hidden && button.offsetParent !== null;
    });
  }

  function moveQuickAddTokenFocus(target, key) {
    var list = quickAddTokenListFor(target);
    var buttons = quickAddTokenButtons(list);
    if (!buttons.length) return false;
    var currentIndex = buttons.indexOf(target);
    if (currentIndex < 0) return false;

    var nextIndex = currentIndex;
    if (key === 'ArrowRight' || key === 'ArrowDown') {
      nextIndex = (currentIndex + 1) % buttons.length;
    } else if (key === 'ArrowLeft' || key === 'ArrowUp') {
      nextIndex = (currentIndex - 1 + buttons.length) % buttons.length;
    } else if (key === 'Home') {
      nextIndex = 0;
    } else if (key === 'End') {
      nextIndex = buttons.length - 1;
    } else {
      return false;
    }

    buttons[nextIndex].focus();
    return true;
  }

  function quickAddLabelsInputFor(element) {
    var form = quickAddSaveFormFor(element);
    if (!form) return null;
    return form.querySelector('[data-quick-add-labels-input], [name="labels"]');
  }

  function quickAddLabelValues(value) {
    return String(value || '').split(',').map(function (part) {
      return part.trim();
    }).filter(Boolean);
  }

  function setQuickAddLabelValues(input, labels) {
    if (!input) return;
    input.value = labels.join(', ');
    input.dispatchEvent(new Event('input', { bubbles: true }));
    input.dispatchEvent(new Event('change', { bubbles: true }));
    if (typeof input.focus === 'function') {
      input.focus();
    }
  }

  function appendQuickAddLabel(button) {
    if (!button) return;
    var label = (button.getAttribute('data-quick-add-append-label') || '').trim();
    if (!label) return;
    var input = quickAddLabelsInputFor(button);
    if (!input) return;
    var labels = quickAddLabelValues(input.value);
    var exists = labels.some(function (existing) {
      return existing.toLowerCase() === label.toLowerCase();
    });
    if (!exists) {
      labels.push(label);
    }
    setQuickAddLabelValues(input, labels);
    hideElement(button);
  }

  function removeQuickAddLabel(button) {
    if (!button) return;
    var label = (button.getAttribute('data-quick-add-remove-label') || '').trim();
    if (!label) return;
    var input = quickAddLabelsInputFor(button);
    if (!input) return;
    var labels = quickAddLabelValues(input.value).filter(function (existing) {
      return existing.toLowerCase() !== label.toLowerCase();
    });
    setQuickAddLabelValues(input, labels);

    var chip = button.closest('.caldo-quick-add-chip');
    if (chip && chip.closest('[data-quick-add-chips]')) {
      hideElement(chip);
      var chipList = chip.closest('[data-quick-add-chips]');
      if (chipList && !chipList.querySelector('.caldo-quick-add-chip:not([hidden])')) {
        hideElement(chipList);
      }
    }
  }

  function quickAddRecurrenceIsValid(value) {
    var recurrence = String(value || '').trim();
    if (!recurrence) return true;
    if (/[\r\n]/.test(recurrence)) return false;

    var hasFreq = false;
    var parts = recurrence.split(';');
    for (var i = 0; i < parts.length; i += 1) {
      var part = parts[i].trim();
      if (!part) return false;
      var index = part.indexOf('=');
      if (index <= 0 || index === part.length - 1) return false;
      var key = part.slice(0, index).trim();
      var partValue = part.slice(index + 1).trim();
      if (!key || !partValue) return false;
      if (!/^[A-Za-z0-9-]+$/.test(key)) return false;
      if (/[\r\n:;]/.test(partValue)) return false;
      if (key.toUpperCase() === 'FREQ') {
        hasFreq = true;
      }
    }
    return hasFreq;
  }

  function quickAddRecurrenceErrorFor(input) {
    var form = quickAddSaveFormFor(input);
    return form ? form.querySelector('[data-quick-add-recurrence-error]') : null;
  }

  function updateQuickAddRecurrenceValidity(input) {
    if (!input) return true;
    var valid = quickAddRecurrenceIsValid(input.value);
    var error = quickAddRecurrenceErrorFor(input);
    var message = error ? error.textContent || '' : '';
    input.setAttribute('aria-invalid', valid ? 'false' : 'true');
    if (typeof input.setCustomValidity === 'function') {
      input.setCustomValidity(valid ? '' : message);
    }
    if (error) {
      error.hidden = valid;
    }
    return valid;
  }

  function validateQuickAddSaveForm(form) {
    if (!form) return true;
    var recurrenceInput = form.querySelector('[data-quick-add-recurrence-input]');
    if (!recurrenceInput || updateQuickAddRecurrenceValidity(recurrenceInput)) {
      return true;
    }
    if (typeof recurrenceInput.reportValidity === 'function') {
      recurrenceInput.reportValidity();
    }
    if (typeof recurrenceInput.focus === 'function') {
      recurrenceInput.focus();
    }
    return false;
  }

  function handleQuickAddRecurrenceInputEvent(event) {
    var input = closestElement(event.target, '[data-quick-add-recurrence-input]');
    if (!input) return;
    updateQuickAddRecurrenceValidity(input);
  }

  function handleQuickAddSaveSubmit(event) {
    var form = closestElement(event.target, '[data-quick-add-save-form]');
    if (!form) return;
    if (validateQuickAddSaveForm(form)) return;
    event.preventDefault();
    event.stopPropagation();
  }

  function inlineCreateError(root) {
    return root ? root.querySelector('[data-inline-task-create-error]') : null;
  }

  function setInlineCreateError(root, message) {
    var error = inlineCreateError(root);
    setErrorMessage(error, message);
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
    setErrorMessage(error, message);
  }

  function taskRows() {
    return Array.prototype.slice.call(document.querySelectorAll('[data-task-id]:not([data-task-detail-subtask])'));
  }

  function initializeTaskCompletionControls(root) {
    var controls = (root || document).querySelectorAll('[data-task-completion-state-control]');
    Array.prototype.forEach.call(controls, function (control) {
      var row = control.closest('[data-task-id]');
      if (!row) return;
      var completed = (row.getAttribute('data-task-status') || '').trim().toLowerCase() === 'completed';
      control.checked = completed;
      control.defaultChecked = completed;
    });
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

  function taskProjectID(row) {
    return row ? row.getAttribute('data-task-project-id') || '' : '';
  }

  function taskMovePath(row) {
    var path = row ? row.getAttribute('data-task-move-path') || '' : '';
    if (path) {
      return path;
    }
    return '/tasks/' + encodeURIComponent(taskRowID(row)) + '/move';
  }

  function taskCanDragMove(row) {
    return !!(row && row.hasAttribute('data-task-drag-move') && taskRowID(row) && taskRowVersion(row) > 0 && taskProjectID(row));
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

  function inlineEditForms(row) {
    if (!row) return [];
    return Array.prototype.slice.call(row.querySelectorAll('[data-inline-task-edit-form]'));
  }

  function visibleInlineEditForm(row) {
    return inlineEditForms(row).find(function (form) {
      return !form.hidden && !form.hasAttribute('data-inline-task-edit-persistent');
    }) || null;
  }

  function openTaskForms(row) {
    var forms = [];
    if (!row) return forms;

    inlineEditForms(row).forEach(function (inlineEdit) {
      if (!inlineEdit.hidden) {
        forms.push(inlineEdit);
      }
    });

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
    if (!nextRow) return null;
    if (row.hasAttribute('data-task-parent-visible')) {
      nextRow.setAttribute('data-task-parent-visible', '');
      nextRow.classList.add('caldo-task-row-subtask');
    }
    row.replaceWith(nextRow);
    if (window.htmx && typeof window.htmx.process === 'function') {
      window.htmx.process(nextRow);
    }
    return nextRow;
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
        if (!document.contains(row)) return '';
        if (rowHasDirtyOpenForm(row)) {
          markStaleLocalChanges(row);
        } else {
          removeTaskRow(row);
        }
        return '';
      }
      if (!response.ok) {
        throw response;
      }
      return response.text();
    }).then(function (html) {
      if (html) {
        if (!document.contains(row)) return null;
        if (rowHasDirtyOpenForm(row)) {
          markStaleLocalChanges(row);
          return null;
        }
        return replaceTaskRow(row, html);
      }
      return null;
    }).catch(function () {
      // Focus refresh is opportunistic; normal writes still surface their own errors.
      return null;
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

  function closeInlineEditForm(root, form, shouldFocusTrigger) {
    if (!root || !form) return;
    var scope = closestElement(form, '[data-inline-task-edit-scope]');
    var focusTarget = scope ? scope.getAttribute('data-inline-task-edit-focus') || 'title' : root.dataset.inlineTaskEditFocus || 'title';
    var trigger = scope ? scope.querySelector('[data-inline-task-edit-open]') : root.querySelector('[data-inline-task-edit-open][data-inline-task-edit-focus="' + focusTarget + '"]');
    form.reset();
    if (form.hasAttribute('data-inline-task-edit-persistent')) {
      if (shouldFocusTrigger) {
        var persistentInput = form.querySelector('[data-inline-task-edit-title]');
        if (persistentInput) persistentInput.blur();
        root.focus({ preventScroll: true });
      }
      return;
    }
    form.hidden = true;
    if (trigger) {
      trigger.hidden = false;
      trigger.setAttribute('aria-expanded', 'false');
      if (shouldFocusTrigger) {
        trigger.focus();
      }
    }
  }

  function openInlineEdit(root, focusTarget) {
    if (!root) return;
    focusTarget = focusTarget || 'title';
    var scope = root.querySelector('[data-inline-task-edit-scope][data-inline-task-edit-focus="' + focusTarget + '"]') || root.querySelector('[data-inline-task-edit-scope]');
    var form = scope ? scope.querySelector('[data-inline-task-edit-form]') : null;
    var trigger = scope ? scope.querySelector('[data-inline-task-edit-open]') : null;
    if (!form || !trigger) return;
    inlineEditForms(root).forEach(function (otherForm) {
      if (otherForm !== form && !otherForm.hidden && !otherForm.hasAttribute('data-inline-task-edit-persistent')) {
        closeInlineEditForm(root, otherForm, false);
      }
    });
    root.dataset.inlineTaskEditState = 'open';
    root.dataset.inlineTaskEditFocus = focusTarget;
    form.reset();
    trigger.hidden = true;
    trigger.setAttribute('aria-expanded', 'true');
    form.hidden = false;
    setInlineEditError(root, '');
    var input = form.querySelector('[data-inline-task-edit-control]') || form.querySelector('[data-inline-task-edit-title]');
    if (input) {
      window.setTimeout(function () {
        input.focus();
      }, 0);
    }
  }

  function closeInlineEdit(root, form) {
    if (!root) return;
    form = form || visibleInlineEditForm(root);
    if (!form) return;
    closeInlineEditForm(root, form, true);
    if (!visibleInlineEditForm(root)) {
      root.dataset.inlineTaskEditState = 'closed';
    }
    setInlineEditError(root, '');
  }

  function taskDetailError(dialog) {
    return dialog ? dialog.querySelector('[data-task-detail-error]') : null;
  }

  function setTaskDetailError(dialog, message) {
    var error = taskDetailError(dialog);
    setErrorMessage(error, message);
  }

  function completeTaskFromDetail(trigger) {
    var dialog = trigger ? trigger.closest('[data-task-detail-dialog]') : null;
    var form = dialog ? dialog.querySelector('[data-task-detail-form]') : null;
    if (!dialog || !form) return;
    if (formIsDirty(form)) {
      setTaskDetailError(dialog, 'Ungespeicherte Änderungen zuerst speichern oder mit Schließen verwerfen.');
      return;
    }

    var row = trigger.closest('[data-task-id]');
    var decisionTrigger = row ? row.querySelector('.caldo-task-row-grid [data-task-complete-open]') : null;
    if (decisionTrigger) {
      closeTaskDetail(dialog);
      openTaskComplete(decisionTrigger);
      return;
    }

    var completionForm = row ? row.querySelector('.caldo-task-row-grid [data-task-action-form]') : null;
    if (!completionForm) {
      setTaskDetailError(dialog, 'Aufgabe konnte nicht erledigt werden.');
      return;
    }
    setTaskDetailError(dialog, '');
    completionForm.__caldoTaskDetailDialog = dialog;
    submitForm(completionForm);
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

  function removeParentContextRow(dialog) {
    var contextRow = dialog ? dialog.closest('[data-task-parent-context-row]') : null;
    if (contextRow) contextRow.remove();
  }

  function bindTaskDetail(dialog) {
    if (!dialog || dialog.dataset.taskDetailBound === 'true') return;
    dialog.dataset.taskDetailBound = 'true';
    dialog.addEventListener('close', function () {
      restoreEditableControlDragRow();
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
      removeParentContextRow(dialog);
    });
    dialog.addEventListener('click', function (event) {
      if (event.target === dialog) {
        closeTaskDetail(dialog);
      }
    });
  }

  function openTaskDetail(trigger, returnFocus) {
    if (!trigger) return;
    var task = trigger.closest('[data-task-id]');
    var dialog = controlledTaskDialog(trigger, task, '[data-task-detail-dialog]');
    if (!dialog) return;
    bindTaskDetail(dialog);
    dialog.__caldoReturnFocus = returnFocus || trigger;
    dialog.__caldoReturnFocus.setAttribute('aria-expanded', 'true');
    if (typeof dialog.showModal === 'function') {
      if (!dialog.open) {
        dialog.showModal();
      }
    } else {
      dialog.setAttribute('open', '');
    }
    requestTaskDetailSubtasks(dialog);
    var input = dialog.querySelector('[data-task-detail-title]');
    var close = dialog.querySelector('[data-task-detail-close]');
    window.setTimeout(function () {
      if (input && !input.disabled) {
        input.focus();
        return;
      }
      if (close) close.focus();
    }, 0);
  }

  function taskRowForID(taskID) {
    var rows = document.querySelectorAll('.caldo-task-row[data-task-id]');
    for (var i = 0; i < rows.length; i += 1) {
      if (!rows[i].hasAttribute('data-task-parent-context-row') && taskRowID(rows[i]) === taskID) {
        return rows[i];
      }
    }
    return null;
  }

  function openTaskRowContext(row, returnFocus) {
    if (!row) return false;
    var grid = row.firstElementChild && row.firstElementChild.matches('.caldo-task-row-grid') ? row.firstElementChild : null;
    var detailTrigger = grid ? grid.querySelector('[data-task-detail-open]') : null;
    if (detailTrigger) {
      openTaskDetail(detailTrigger, returnFocus);
      return true;
    }
    var conflictLink = grid ? grid.querySelector('[data-task-conflict-link]') : null;
    if (conflictLink && conflictLink.href) {
      window.location.href = conflictLink.href;
      return true;
    }
    return false;
  }

  function openParentTaskDetail(trigger) {
    if (!trigger || trigger.getAttribute('aria-busy') === 'true') return;
    var parentTaskID = (trigger.getAttribute('data-parent-task-id') || '').trim();
    if (!parentTaskID) return;

    var existingRow = taskRowForID(parentTaskID);
    if (existingRow && openTaskRowContext(existingRow, trigger)) return;

    trigger.setAttribute('aria-busy', 'true');
    window.fetch('/tasks/' + encodeURIComponent(parentTaskID), {
      method: 'GET',
      credentials: 'same-origin',
      headers: {
        'Accept': 'text/html',
        'X-Tab-ID': tabID
      }
    }).then(function (response) {
      if (!response.ok) throw response;
      return response.text();
    }).then(function (html) {
      var parentRow = taskRowFromHTML(html, parentTaskID);
      if (!parentRow) throw new Error('parent task fragment missing');
      parentRow.setAttribute('data-task-parent-context-row', '');
      document.body.appendChild(parentRow);
      if (window.htmx && typeof window.htmx.process === 'function') {
        window.htmx.process(parentRow);
      }
      if (!openTaskRowContext(parentRow, trigger)) {
        parentRow.remove();
        throw new Error('parent task details unavailable');
      }
    }).catch(function () {
      setTaskActionError(trigger.closest('[data-task-id]'), 'Elternaufgabe konnte nicht geladen werden.');
    }).finally(function () {
      if (document.contains(trigger)) trigger.removeAttribute('aria-busy');
    });
  }

  function requestTaskDetailSubtasks(dialog) {
    var list = dialog ? dialog.querySelector('[data-task-detail-subtask-list]') : null;
    if (!list || !window.htmx || typeof window.htmx.trigger !== 'function') return;
    window.htmx.trigger(list, 'caldo:task-detail-open');
  }

  function controlledTaskDialog(trigger, task, selector) {
    var controlledID = trigger ? trigger.getAttribute('aria-controls') || '' : '';
    var controlled = controlledID ? document.getElementById(controlledID) : null;
    if (controlled && controlled.matches(selector)) {
      return controlled;
    }
    return task ? task.querySelector(selector) : null;
  }

  function closeTaskDetail(dialog) {
    if (!dialog) return;
    restoreEditableControlDragRow();
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
    removeParentContextRow(dialog);
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
    setErrorMessage(error, message);
  }

  function taskCompleteError(dialog) {
    return dialog ? dialog.querySelector('[data-task-complete-error]') : null;
  }

  function setTaskCompleteError(dialog, message) {
    var error = taskCompleteError(dialog);
    setErrorMessage(error, message);
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
    var dialog = controlledTaskDialog(trigger, task, '[data-task-complete-dialog]');
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
    setErrorMessage(error, message);
  }

  function bindTaskDelete(dialog) {
    if (!dialog || dialog.dataset.taskDeleteBound === 'true') return;
    dialog.dataset.taskDeleteBound = 'true';
    dialog.addEventListener('close', function () {
      var form = dialog.querySelector('[data-task-delete-form]');
      if (form) form.reset();
      setTaskDeleteError(dialog, '');
      var trigger = dialog.__caldoDeleteTrigger;
      dialog.__caldoDeleteTrigger = null;
      if (trigger && document.contains(trigger)) {
        trigger.setAttribute('aria-expanded', 'false');
      }
      trigger = dialog.__caldoReturnFocus;
      dialog.__caldoReturnFocus = null;
      if (trigger && document.contains(trigger)) {
        trigger.focus();
      }
    });
    dialog.addEventListener('click', function (event) {
      if (event.target === dialog) {
        closeTaskDelete(dialog);
      }
    });
  }

  function openTaskDelete(trigger, returnFocus) {
    if (!trigger) return;
    var task = trigger.closest('[data-task-id]');
    var dialog = controlledTaskDialog(trigger, task, '[data-task-delete-dialog]');
    if (!dialog) return;
    bindTaskDelete(dialog);
    dialog.__caldoDeleteTrigger = trigger;
    dialog.__caldoReturnFocus = returnFocus || trigger;
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
    var trigger = dialog.__caldoDeleteTrigger;
    dialog.__caldoDeleteTrigger = null;
    if (trigger && document.contains(trigger)) {
      trigger.setAttribute('aria-expanded', 'false');
    }
    trigger = dialog.__caldoReturnFocus;
    dialog.__caldoReturnFocus = null;
    if (trigger && document.contains(trigger)) {
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

  function taskMoveFailureMessage(status) {
    if (status === 409) {
      return 'Aufgabe konnte nicht verschoben werden. Aufgabe prüfen.';
    }
    if (status === 400) {
      return 'Aufgabe konnte nicht in dieses Projekt verschoben werden.';
    }
    if (status === 424 || status === 502) {
      return 'Aufgabe konnte nicht auf dem CalDAV-Server verschoben werden.';
    }
    return 'Aufgabe konnte nicht verschoben werden.';
  }

  function projectDropTargetFrom(target) {
    return closestElement(target, '[data-project-drop-target]');
  }

  function projectDropTargetID(target) {
    return target ? target.getAttribute('data-project-id') || '' : '';
  }

  function clearProjectDropTargets() {
    document.querySelectorAll('[data-project-drop-target]').forEach(function (target) {
      target.classList.remove('caldo-project-drop-active', 'caldo-project-drop-blocked');
    });
  }

  function markProjectDropTarget(target, blocked) {
    clearProjectDropTargets();
    if (!target) return;
    target.classList.add(blocked ? 'caldo-project-drop-blocked' : 'caldo-project-drop-active');
  }

  function clearTaskMoveDragState() {
    if (taskMoveDragState && taskMoveDragState.row) {
      taskMoveDragState.row.classList.remove('caldo-task-row-dragging');
    }
    taskMoveDragState = null;
    clearProjectDropTargets();
  }

  function isProjectScopedSearchView() {
    if (window.location.pathname !== '/search') return false;
    var query = '';
    try {
      query = new URLSearchParams(window.location.search).get('q') || '';
    } catch (_) {
      return false;
    }
    return query.trim().charAt(0) === '#';
  }

  function updateMovedTaskRow(row) {
    if (!row || !document.contains(row)) return;
    if (isProjectScopedSearchView()) {
      removeTaskRow(row);
      return;
    }
    fetchTaskFragment(row);
  }

  function submitTaskMove(row, targetProjectID) {
    if (!taskCanDragMove(row) || !targetProjectID || targetProjectID === taskProjectID(row)) return;

    var form = new URLSearchParams();
    form.set('project_id', targetProjectID);
    form.set('expected_version', String(taskRowVersion(row)));

    setTaskActionError(row, '');
    row.classList.add('caldo-task-row-moving');
    beginWriteRequest('Verschieben ...');

    window.fetch(taskMovePath(row), {
      method: 'POST',
      credentials: 'same-origin',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
        'Accept': 'text/plain',
        'X-CSRF-Token': csrfToken(),
        'X-Tab-ID': tabID
      },
      body: form.toString()
    }).then(function (response) {
      if (!response.ok) {
        throw response;
      }
      return response.text();
    }).then(function () {
      row.classList.remove('caldo-task-row-moving');
      finishWriteRequest();
      setWriteStatus('saved', 'Gespeichert');
      clearSavedStatusSoon();
      refreshUndoNotification();
      updateMovedTaskRow(row);
    }).catch(function (response) {
      var status = response && response.status ? response.status : 0;
      var message = taskMoveFailureMessage(status);
      row.classList.remove('caldo-task-row-moving');
      finishWriteRequest();
      setWriteStatus('error', message);
      setTaskActionError(row, message);
    });
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
    beginWriteRequest('Speichern ...');
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
      finishWriteRequest();
      setWriteStatus('saved', 'Rückgängig ausgeführt.');
      rememberWriteStatus('saved', 'Rückgängig ausgeführt.');
      window.setTimeout(function () {
        window.location.reload();
      }, 1200);
    }).catch(function (response) {
      var status = response && response.status ? response.status : 0;
      button.disabled = false;
      finishWriteRequest();
      setWriteStatus('error', 'Rückgängig fehlgeschlagen. Änderungen prüfen.');
      setUndoNotificationError(button, undoFailureMessage(status));
    });
  }

  var writeState = { pendingRequests: 0 };
  var trackedHTMXWriteRequests = new WeakSet();

  function beginWriteRequest(message) {
    writeState.pendingRequests += 1;
    setWriteStatus('pending', message || 'Speichern ...');
  }

  function finishWriteRequest() {
    if (writeState.pendingRequests > 0) {
      writeState.pendingRequests -= 1;
    }
  }

  function isTaskDisplayPreferenceRequest(detail) {
    return !!closestElement(detail && detail.elt, '[data-task-display-form]');
  }

  function shouldTrackHTMXWrite(detail) {
    var method = ((detail && detail.requestConfig && detail.requestConfig.verb) || '').toUpperCase();
    if (!method || method === 'GET') return false;
    if (isSyncRequestElement(detail && detail.elt)) return false;
    return !isTaskDisplayPreferenceRequest(detail);
  }

  function beginTrackedHTMXWrite(detail) {
    var xhr = detail && detail.xhr;
    if (!xhr || !shouldTrackHTMXWrite(detail) || trackedHTMXWriteRequests.has(xhr)) return;
    trackedHTMXWriteRequests.add(xhr);
    beginWriteRequest('Speichern ...');
    xhr.addEventListener('loadend', function () {
      finishTrackedHTMXWrite(xhr);
    }, { once: true });
  }

  function finishTrackedHTMXWrite(xhr) {
    if (!xhr || !trackedHTMXWriteRequests.has(xhr)) return false;
    trackedHTMXWriteRequests.delete(xhr);
    finishWriteRequest();
    return true;
  }

  function rememberWriteStatus(kind, message) {
    if (!message) return;
    tabState.writeStatus = { kind: kind || 'saved', message: message };
    saveTabState(tabState);
  }

  function consumeRememberedWriteStatus() {
    var status = tabState.writeStatus;
    if (!status || !status.message) return;
    delete tabState.writeStatus;
    saveTabState(tabState);
    setWriteStatus(status.kind || 'saved', status.message);
    if ((status.kind || 'saved') === 'saved') {
      clearSavedStatusSoon();
    }
  }

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
    }, 2500);
  }

  function csrfToken() {
    var meta = document.querySelector('meta[name="csrf-token"]');
    return meta ? meta.getAttribute('content') || '' : '';
  }

  function setupPost(path) {
    return fetch(path, {
      method: 'POST',
      credentials: 'same-origin',
      headers: {
        'X-CSRF-Token': csrfToken(),
        'X-Tab-ID': tabID
      }
    });
  }

  function setupImportRoots(root) {
    var scope = root && root.querySelectorAll ? root : document;
    var roots = [];
    if (scope instanceof Element && scope.matches('[data-setup-import]')) {
      roots.push(scope);
    }
    Array.prototype.forEach.call(scope.querySelectorAll('[data-setup-import]'), function (element) {
      roots.push(element);
    });
    return roots;
  }

  function setupImportElement(root, selector) {
    return root ? root.querySelector(selector) : null;
  }

  function setSetupImportMessage(root, status, detail) {
    var statusElement = setupImportElement(root, '[data-setup-import-status]');
    var detailElement = setupImportElement(root, '[data-setup-import-detail]');
    if (statusElement && status) {
      statusElement.textContent = status;
    }
    if (detailElement && detail) {
      detailElement.textContent = detail;
    }
  }

  function setSetupImportProgress(root, percent) {
    var bar = setupImportElement(root, '[data-setup-import-progress-bar]');
    if (!bar) return;
    var clamped = Math.max(0, Math.min(100, percent || 0));
    bar.value = clamped;
  }

  function setSetupImportError(root, message) {
    var error = setupImportElement(root, '[data-setup-import-error]');
    var retry = setupImportElement(root, '[data-setup-import-retry]');
    setErrorMessage(error, message);
    if (retry) {
      retry.hidden = !message;
    }
  }

  function parseSetupImportEvent(data) {
    if (!data) return {};
    try {
      return JSON.parse(data);
    } catch (_) {
      return {};
    }
  }

  function updateSetupImportProgress(root, event) {
    var payload = parseSetupImportEvent(event && event.data);
    if (payload.total && payload.completed) {
      setSetupImportProgress(root, (payload.completed / payload.total) * 100);
      setSetupImportMessage(root, 'Import läuft ...', 'Kalender ' + payload.completed + ' von ' + payload.total + ' wurde importiert.');
      return;
    }
    setSetupImportProgress(root, 15);
    setSetupImportMessage(root, 'Import läuft ...', 'Aufgaben werden vom CalDAV-Server geladen.');
  }

  function completeSetupImport(root) {
    var completeURL = root.getAttribute('data-setup-import-complete-url') || '/setup/complete';
    setSetupImportProgress(root, 100);
    setSetupImportMessage(root, 'Import abgeschlossen', 'Setup wird abgeschlossen ...');
    setupPost(completeURL).then(function (response) {
      if (response.redirected && response.url) {
        window.location.assign(response.url);
        return;
      }
      if (response.ok) {
        window.location.assign('/');
        return;
      }
      throw new Error('setup complete failed with status ' + response.status);
    }).catch(function () {
      setSetupImportError(root, 'Setup konnte nach dem Import nicht abgeschlossen werden.');
    });
  }

  function startSetupImport(root) {
    if (!root || root.dataset.setupImportRunning === 'true') return;
    var startURL = root.getAttribute('data-setup-import-start-url') || '/setup/import';
    root.dataset.setupImportRunning = 'true';
    root.dataset.setupImportDone = 'false';
    setSetupImportError(root, '');
    setSetupImportProgress(root, 5);
    setSetupImportMessage(root, 'Import startet ...', 'Verbindung zum Setup-Import wird vorbereitet.');

    var source = new EventSource('/setup/import/events');
    var importStarted = false;
    root.__caldoSetupImportSource = source;
    var requestImportStart = function () {
      if (importStarted) return;
      importStarted = true;
      setupPost(startURL).then(function (response) {
        if (response.status === 202 || response.status === 409) {
          return;
        }
        throw new Error('setup import failed with status ' + response.status);
      }).catch(function () {
        root.dataset.setupImportRunning = 'false';
        source.close();
        setSetupImportError(root, 'Initialimport konnte nicht gestartet werden.');
      });
    };
    source.addEventListener('ready', function () {
      requestImportStart();
    });
    source.addEventListener('progress', function (event) {
      updateSetupImportProgress(root, event);
    });
    source.addEventListener('done', function () {
      root.dataset.setupImportDone = 'true';
      root.dataset.setupImportRunning = 'false';
      source.close();
      completeSetupImport(root);
    });
    source.addEventListener('error', function (event) {
      if (event && typeof event.data === 'string' && event.data) {
        root.dataset.setupImportRunning = 'false';
        source.close();
        setSetupImportError(root, 'Initialimport ist fehlgeschlagen.');
        return;
      }
      if (root.dataset.setupImportDone === 'true') return;
      setSetupImportMessage(root, 'Import wartet ...', 'Die Fortschrittsverbindung wird erneut aufgebaut.');
    });
  }

  function initializeSetupImport(root) {
    Array.prototype.forEach.call(setupImportRoots(root), function (importRoot) {
      if (importRoot.dataset.setupImportBound === 'true') return;
      importRoot.dataset.setupImportBound = 'true';
      var retry = setupImportElement(importRoot, '[data-setup-import-retry]');
      if (retry) {
        retry.addEventListener('click', function () {
          startSetupImport(importRoot);
        });
      }
      startSetupImport(importRoot);
    });
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

  function conflictResolutionFormFor(element) {
    return closestElement(element, '[data-conflict-resolve-form], [data-conflict-split-form], [data-conflict-manual-form]');
  }

  function conflictResolutionErrorFor(element) {
    var region = closestElement(element, '[data-conflict-resolution]');
    return region ? region.querySelector('[data-conflict-resolution-error]') : null;
  }

  function setConflictResolutionError(element, message) {
    var error = conflictResolutionErrorFor(element);
    setErrorMessage(error, message);
  }

  function conflictResolutionFailureMessage(status) {
    if (status === 409) {
      return 'Konflikt wurde remote erneut geändert. Bitte erneut prüfen.';
    }
    if (status === 400) {
      return 'Konflikt konnte mit diesen Feldwerten nicht gespeichert werden.';
    }
    if (status === 502 || status === 424) {
      return 'Konflikt konnte nicht auf dem CalDAV-Server gespeichert werden.';
    }
    return 'Konflikt konnte nicht gespeichert werden.';
  }

  function conflictManualInputDisplay(input) {
    if (!input) return '';
    var emptyLabel = input.getAttribute('data-conflict-manual-empty') || '';
    if (input.tagName === 'SELECT') {
      var selected = input.options[input.selectedIndex];
      return selected && selected.textContent ? selected.textContent.trim() : emptyLabel;
    }
    var value = String(input.value || '').trim();
    return value || emptyLabel;
  }

  function updateConflictManualPreview(form) {
    if (!form) return;
    var preview = form.querySelector('[data-conflict-manual-preview]');
    if (!preview) return;

    Array.prototype.forEach.call(preview.querySelectorAll('[data-conflict-preview-value]'), function (target) {
      var field = target.getAttribute('data-conflict-preview-value') || '';
      if (!field || field === 'project') return;
      var checked = form.querySelector('input[name="' + field + '_source"]:checked');
      if (!checked) return;
      if (checked.value === 'manual') {
        target.textContent = conflictManualInputDisplay(form.querySelector('[data-conflict-manual-input="' + field + '"]'));
        target.removeAttribute('data-conflict-preview-iso');
        return;
      }
      var option = closestElement(checked, '[data-conflict-source-option]');
      if (!option) {
        target.textContent = '';
        target.removeAttribute('data-conflict-preview-iso');
        return;
      }
      var optionISO = option.getAttribute('data-conflict-option-display-iso') || '';
      var optionDisplay = option.getAttribute('data-conflict-option-display') || option.textContent.trim();
      var renderedDescription = field === 'description' ? option.querySelector('[data-conflict-description-rendered]') : null;
      if (!cloneRenderedDescription(target, renderedDescription)) {
        target.textContent = optionISO ? formatBrowserDateTimeFromISO(optionISO) || optionDisplay : optionDisplay;
      }
      if (optionISO) {
        target.setAttribute('data-conflict-preview-iso', optionISO);
      } else {
        target.removeAttribute('data-conflict-preview-iso');
      }
    });

    var projectTarget = preview.querySelector('[data-conflict-preview-value="project"]');
    var projectSelect = form.querySelector('[data-conflict-project-select]');
    if (projectTarget && projectSelect) {
      var projectOption = projectSelect.options[projectSelect.selectedIndex];
      projectTarget.textContent = projectOption && projectOption.textContent ? projectOption.textContent.trim() : '';
    }
  }

  function initializeConflictManualPreviews(root) {
    Array.prototype.forEach.call((root || document).querySelectorAll('[data-conflict-manual-form]'), updateConflictManualPreview);
  }

  function updateConflictSplitConfirmation(form) {
    if (!form) return;
    var checkbox = form.querySelector('[data-conflict-split-confirm]');
    var submit = form.querySelector('[data-conflict-split-submit]');
    if (!checkbox || !submit) return;
    submit.disabled = !checkbox.checked;
  }

  function initializeConflictSplitConfirmations(root) {
    Array.prototype.forEach.call((root || document).querySelectorAll('[data-conflict-split-form]'), updateConflictSplitConfirmation);
  }

  function handleConflictManualControlEvent(event) {
    var manualInput = closestElement(event.target, '[data-conflict-manual-input]');
    if (manualInput) {
      var field = manualInput.getAttribute('data-conflict-manual-input') || '';
      var fieldset = closestElement(manualInput, '[data-conflict-field-source]');
      var manualRadio = fieldset ? fieldset.querySelector('input[name="' + field + '_source"][value="manual"]') : null;
      if (manualRadio) {
        manualRadio.checked = true;
      }
      updateConflictManualPreview(closestElement(manualInput, '[data-conflict-manual-form]'));
      return;
    }

    var sourceControl = closestElement(event.target, '[data-conflict-field-source] input[type="radio"]');
    if (sourceControl) {
      updateConflictManualPreview(closestElement(sourceControl, '[data-conflict-manual-form]'));
      return;
    }

    var projectSelect = closestElement(event.target, '[data-conflict-project-select]');
    if (projectSelect) {
      updateConflictManualPreview(closestElement(projectSelect, '[data-conflict-manual-form]'));
    }
  }

  function handleConflictSplitConfirmationEvent(event) {
    var checkbox = closestElement(event.target, '[data-conflict-split-confirm]');
    if (!checkbox) return;
    updateConflictSplitConfirmation(closestElement(checkbox, '[data-conflict-split-form]'));
  }

  document.body.addEventListener('htmx:configRequest', function (event) {
    if (!event.detail || !event.detail.headers) return;
    event.detail.headers['X-Tab-ID'] = tabID;
    var token = csrfToken();
    if (token && !event.detail.headers['X-CSRF-Token']) {
      event.detail.headers['X-CSRF-Token'] = token;
    }
  });

  document.body.addEventListener('htmx:beforeSwap', function (event) {
    var overlaySave = closestElement(event.detail && event.detail.elt, '[data-quick-add-overlay-save-form]');
    if (!overlaySave || !event.detail || !event.detail.xhr) return;
    var status = event.detail.xhr.status;
    if (status >= 200 && status < 300) {
      event.detail.shouldSwap = false;
    }
  });

  document.body.addEventListener('htmx:beforeRequest', function (event) {
    rememberQuickAddPreviewFocusIntent(event.detail);
    var method = ((event.detail && event.detail.requestConfig && event.detail.requestConfig.verb) || '').toUpperCase();
    if (method === 'GET') {
      var target = event.detail && event.detail.target;
      if (target && target.classList && target.classList.contains('caldo-content')) {
        showNavigationSkeleton();
      }
      return;
    }
    if (isSyncRequestElement(event.detail && event.detail.elt)) {
      setWriteStatus(null, '');
      return;
    }
    var quickAddOverlayRequest = closestElement(event.detail && event.detail.elt, '[data-quick-add-overlay]');
    if (quickAddOverlayRequest) {
      setQuickAddOverlayError(quickAddOverlayRequest, '');
    }
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
    var conflictResolution = conflictResolutionFormFor(event.detail && event.detail.elt);
    if (conflictResolution) {
      setConflictResolutionError(conflictResolution, '');
    }
  });

  document.body.addEventListener('htmx:beforeSend', function (event) {
    beginTrackedHTMXWrite(event.detail);
  });

  document.body.addEventListener('htmx:beforeOnLoad', function (event) {
    finishTrackedHTMXWrite(event.detail && event.detail.xhr);
  });

  ['htmx:sendAbort', 'htmx:sendError', 'htmx:timeout'].forEach(function (eventName) {
    document.body.addEventListener(eventName, function (event) {
      finishTrackedHTMXWrite(event.detail && event.detail.xhr);
    });
  });

  document.body.addEventListener('htmx:afterSwap', function (event) {
    var targetElement = event.detail && event.detail.target instanceof Element ? event.detail.target : null;
    initializeFormErrors(targetElement || document);
    initializeTaskCompletionControls(targetElement || document);
    initializeLocalDateTimes(targetElement || document);
    initializeSetupImport(targetElement || document);
    initializeConflictManualPreviews(targetElement || document);
    initializeConflictSplitConfirmations(targetElement || document);
  });

  document.body.addEventListener('htmx:afterSettle', function (event) {
    if (!event.detail || !event.detail.xhr || !explicitQuickAddPreviewRequests.has(event.detail.xhr)) return;
    var requestElement = event.detail && event.detail.elt;
    var targetElement = event.detail && event.detail.target instanceof Element ? event.detail.target : null;
    focusQuickAddPreviewResult(requestElement, targetElement);
  });

  document.body.addEventListener('htmx:afterRequest', function (event) {
    var method = ((event.detail && event.detail.requestConfig && event.detail.requestConfig.verb) || '').toUpperCase();
    if (!method || method === 'GET') return;
    var successful = !!(event.detail && event.detail.successful);
    if (isSyncRequestElement(event.detail && event.detail.elt)) {
      if (!successful) {
        setWriteStatus('error', 'Synchronisierung konnte nicht gestartet werden.');
      }
      return;
    }
    finishTrackedHTMXWrite(event.detail && event.detail.xhr);

    if (!successful) {
      setWriteStatus('error', 'Speichern fehlgeschlagen. Änderungen prüfen.');
      var failedQuickAddOverlaySave = closestElement(event.detail && event.detail.elt, '[data-quick-add-overlay-save-form]');
      if (failedQuickAddOverlaySave) {
        setQuickAddOverlayError(failedQuickAddOverlaySave.closest('[data-quick-add-overlay]'), 'Aufgabe konnte nicht gespeichert werden.');
        return;
      }
      var failedQuickAddOverlayPreview = closestElement(event.detail && event.detail.elt, '[data-quick-add-overlay-form]');
      if (failedQuickAddOverlayPreview) {
        setQuickAddOverlayError(failedQuickAddOverlayPreview.closest('[data-quick-add-overlay]'), 'Vorschau konnte nicht erstellt werden.');
        return;
      }
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
        var detailDialog = failedTaskAction.__caldoTaskDetailDialog;
        var actionMessage = detailDialog ? taskCompleteFailureMessage(actionStatus) : taskActionFailureMessage(actionStatus);
        setTaskActionError(failedTaskAction.closest('[data-task-id]'), actionMessage);
        if (detailDialog && document.contains(detailDialog)) {
          setTaskDetailError(detailDialog, actionMessage);
        }
        failedTaskAction.__caldoTaskDetailDialog = null;
        return;
      }
      var failedConflictResolution = conflictResolutionFormFor(event.detail && event.detail.elt);
      if (failedConflictResolution) {
        var conflictStatus = event.detail && event.detail.xhr ? event.detail.xhr.status : 0;
        setConflictResolutionError(failedConflictResolution, conflictResolutionFailureMessage(conflictStatus));
        return;
      }
      var failedInlineEdit = closestElement(event.detail && event.detail.elt, '[data-task-id]');
      if (failedInlineEdit && failedInlineEdit.querySelector('[data-inline-task-edit-form]')) {
        setInlineEditError(failedInlineEdit, 'Aufgabe konnte nicht gespeichert werden.');
      }
      return;
    }

    if (writeState.pendingRequests === 0) {
      setWriteStatus('saved', 'Gespeichert');
      clearSavedStatusSoon();
    }

    var successfulSubtaskCreate = closestElement(event.detail && event.detail.elt, '[data-subtask-create-form]');
    if (successfulSubtaskCreate) {
      var subtaskCreateRoot = successfulSubtaskCreate.closest('[data-subtask-create]');
      var subtaskParentRow = successfulSubtaskCreate.closest('.caldo-task-row[data-task-id]');
      closeInlineCreate(subtaskCreateRoot);
      fetchTaskFragment(subtaskParentRow).then(function (refreshedParentRow) {
        if (!refreshedParentRow) {
          requestTaskDetailSubtasks(subtaskParentRow && subtaskParentRow.querySelector('[data-task-detail-dialog]'));
          return;
        }
        openTaskDetail(refreshedParentRow.querySelector('[data-task-detail-open]'));
      });
      return;
    }

    var successfulQuickAddOverlaySave = closestElement(event.detail && event.detail.elt, '[data-quick-add-overlay-save-form]');
    if (successfulQuickAddOverlaySave) {
      closeQuickAddOverlay(successfulQuickAddOverlaySave.closest('[data-quick-add-overlay]'), true);
      refreshUndoNotification();
      return;
    }

    var successfulTaskComplete = closestElement(event.detail && event.detail.elt, '[data-task-complete-form]');
    if (successfulTaskComplete) {
      closeTaskComplete(successfulTaskComplete.closest('[data-task-complete-dialog]'));
    }

    var successfulTaskDelete = closestElement(event.detail && event.detail.elt, '[data-task-delete-form]');
    if (successfulTaskDelete) {
      var deleteRow = successfulTaskDelete.closest('[data-task-id]');
      var deletedSubtaskParentRow = deleteRow && deleteRow.matches('[data-task-detail-subtask]')
        ? deleteRow.closest('.caldo-task-row[data-task-id]')
        : null;
      closeTaskDelete(successfulTaskDelete.closest('[data-task-delete-dialog]'));
      removeDeletedTaskRows(deleteRow);
      setWriteStatus('saved', 'Gespeichert');
      clearSavedStatusSoon();
      refreshUndoNotification();
      if (deletedSubtaskParentRow) {
        fetchTaskFragment(deletedSubtaskParentRow).then(function (refreshedParentRow) {
          if (refreshedParentRow) {
            openTaskDetail(refreshedParentRow.querySelector('[data-task-detail-open]'));
          }
        });
      }
      return;
    }

    if (closestElement(event.detail && event.detail.elt, '[data-refresh-after-write]')) {
      if (writeState.pendingRequests === 0) {
        rememberWriteStatus('saved', 'Gespeichert');
      }
      window.location.reload();
    }
  });

  window.addEventListener('beforeunload', function (event) {
    if (writeState.pendingRequests <= 0) return;
    event.preventDefault();
    event.returnValue = '';
  });

  document.addEventListener('input', handleTaskRecurrenceControlEvent, true);
  document.addEventListener('input', function (event) {
    var searchInput = closestElement(event.target, '[data-live-search-input]');
    if (!searchInput) return;
    var searchQuery = document.querySelector('[data-task-display-form] input[name="search_query"]');
    if (searchQuery) searchQuery.value = searchInput.value;
  }, true);
  document.addEventListener('change', handleTaskRecurrenceControlEvent, true);
  document.addEventListener('input', handleQuickAddRecurrenceInputEvent, true);
  document.addEventListener('change', handleQuickAddRecurrenceInputEvent, true);
  document.addEventListener('input', handleConflictManualControlEvent, true);
  document.addEventListener('change', handleConflictManualControlEvent, true);
  document.addEventListener('change', handleConflictSplitConfirmationEvent, true);

  function updateInlinePriorityAppearance(control) {
    if (!control) return;
    var priority = parseInt(control.value || '', 10);
    control.classList.remove('caldo-task-priority-p1', 'caldo-task-priority-p2', 'caldo-task-priority-p3');
    if (!Number.isFinite(priority) || priority <= 0) return;
    if (priority <= 4) {
      control.classList.add('caldo-task-priority-p1');
      return;
    }
    if (priority <= 6) {
      control.classList.add('caldo-task-priority-p2');
      return;
    }
    control.classList.add('caldo-task-priority-p3');
  }

  document.addEventListener('change', function (event) {
    var taskDisplaySort = closestElement(event.target, '[data-task-display-sort]');
    if (taskDisplaySort) {
      var taskDisplayForm = taskDisplaySort.closest('[data-task-display-form]');
      var orderField = taskDisplayForm && taskDisplayForm.querySelector('[data-task-display-order-field]');
      var orderSelect = taskDisplayForm && taskDisplayForm.querySelector('[data-task-display-order]');
      if (orderField) orderField.hidden = taskDisplaySort.value === 'default';
      if (orderSelect && taskDisplaySort.value === 'default') orderSelect.value = 'asc';
      return;
    }
    var inlineEditAutosave = closestElement(event.target, '[data-inline-task-edit-autosave]');
    if (inlineEditAutosave && !inlineEditAutosave.disabled) {
      updateInlinePriorityAppearance(closestElement(inlineEditAutosave, '[data-inline-task-priority-select]'));
      submitForm(inlineEditAutosave.closest('[data-inline-task-edit-form]'));
      return;
    }
    var checkbox = closestElement(event.target, '[data-task-completion-checkbox]');
    if (!checkbox) return;
    var form = checkbox.closest('[data-task-action-form]');
    submitForm(form);
  }, true);
  document.addEventListener('submit', handleQuickAddSaveSubmit, true);
  document.addEventListener('keydown', handleBulkSelectionKeydown, true);

  function restoreEditableControlDragRow() {
    if (editableControlDragRow && document.contains(editableControlDragRow) && taskCanDragMove(editableControlDragRow)) {
      editableControlDragRow.setAttribute('draggable', 'true');
    }
    editableControlDragRow = null;
  }

  function editableControlInDraggableTask(target) {
    return closestElement(target, [
      '[data-inline-task-edit-title]',
      '[data-task-detail-dialog] input:not([type="hidden"])',
      '[data-task-detail-dialog] textarea',
      '[data-task-detail-dialog] select'
    ].join(', '));
  }

  document.addEventListener('pointerdown', function (event) {
    var control = editableControlInDraggableTask(event.target);
    var row = control ? control.closest('[data-task-drag-move]') : null;
    if (!row) return;
    restoreEditableControlDragRow();
    editableControlDragRow = row;
    row.setAttribute('draggable', 'false');
  }, true);
  document.addEventListener('pointerup', restoreEditableControlDragRow, true);
  document.addEventListener('pointercancel', restoreEditableControlDragRow, true);
  window.addEventListener('blur', restoreEditableControlDragRow);

  document.addEventListener('dragstart', function (event) {
    var row = closestElement(event.target, '[data-task-drag-move]');
    if (!taskCanDragMove(row)) return;

    taskMoveDragState = { row: row, projectID: taskProjectID(row) };
    row.classList.add('caldo-task-row-dragging');
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = 'move';
      event.dataTransfer.setData('text/plain', taskRowID(row));
    }
  });

  document.addEventListener('dragover', function (event) {
    if (!taskMoveDragState) return;
    var target = projectDropTargetFrom(event.target);
    if (!target) {
      clearProjectDropTargets();
      return;
    }

    var targetProjectID = projectDropTargetID(target);
    var blocked = !targetProjectID || targetProjectID === taskMoveDragState.projectID;
    event.preventDefault();
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = blocked ? 'none' : 'move';
    }
    markProjectDropTarget(target, blocked);
  });

  document.addEventListener('dragleave', function (event) {
    if (!taskMoveDragState) return;
    var target = projectDropTargetFrom(event.target);
    if (!target) return;
    var related = event.relatedTarget instanceof Element ? event.relatedTarget : null;
    if (related && target.contains(related)) return;
    target.classList.remove('caldo-project-drop-active', 'caldo-project-drop-blocked');
  });

  document.addEventListener('drop', function (event) {
    if (!taskMoveDragState) return;
    var target = projectDropTargetFrom(event.target);
    if (!target) {
      clearTaskMoveDragState();
      return;
    }

    event.preventDefault();
    var row = taskMoveDragState.row;
    var targetProjectID = projectDropTargetID(target);
    clearTaskMoveDragState();
    submitTaskMove(row, targetProjectID);
  });

  document.addEventListener('dragend', clearTaskMoveDragState);

  document.addEventListener('click', function (event) {
    if (handleBulkSelectionClick(event)) return;

    var quickAddAppendLabel = closestElement(event.target, '[data-quick-add-append-label]');
    if (quickAddAppendLabel) {
      event.preventDefault();
      appendQuickAddLabel(quickAddAppendLabel);
      showQuickAddCorrections(quickAddAppendLabel);
      return;
    }

    var quickAddRemoveLabel = closestElement(event.target, '[data-quick-add-remove-label]');
    if (quickAddRemoveLabel) {
      event.preventDefault();
      removeQuickAddLabel(quickAddRemoveLabel);
      showQuickAddCorrections(quickAddRemoveLabel);
      return;
    }

    var quickAddClear = closestElement(event.target, '[data-quick-add-clear]');
    if (quickAddClear) {
      event.preventDefault();
      clearQuickAddControl(quickAddClear);
      showQuickAddCorrections(quickAddClear);
      return;
    }

    var quickAddCorrectionsToggle = closestElement(event.target, '[data-quick-add-corrections-toggle]');
    if (quickAddCorrectionsToggle) {
      event.preventDefault();
      toggleQuickAddCorrections(quickAddCorrectionsToggle);
      return;
    }

    var quickAddOpen = closestElement(event.target, '[data-quick-add-open]');
    if (quickAddOpen) {
      event.preventDefault();
      closeMobileNav();
      openQuickAddOverlay(quickAddOpen);
      return;
    }

    var quickAddClose = closestElement(event.target, '[data-quick-add-close]');
    if (quickAddClose) {
      event.preventDefault();
      closeQuickAddOverlay(quickAddClose.closest('[data-quick-add-overlay]'), true);
      return;
    }

    var themeToggle = closestElement(event.target, '[data-theme-toggle]');
    if (themeToggle) {
      event.preventDefault();
      var currentMode = document.documentElement ? document.documentElement.getAttribute('data-theme-mode') : '';
      applyThemeMode(nextThemeMode(currentMode));
      return;
    }

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

    var inlineEditOpen = closestElement(event.target, '[data-inline-task-edit-open]');
    if (inlineEditOpen && !closestElement(event.target, '[data-inline-task-edit-form]')) {
      event.preventDefault();
      var inlineEditFocus = inlineEditOpen.getAttribute('data-inline-task-edit-focus') || 'title';
      openInlineEdit(inlineEditOpen.closest('[data-task-id]'), inlineEditFocus);
      return;
    }

    var taskDetailClose = closestElement(event.target, '[data-task-detail-close]');
    if (taskDetailClose) {
      event.preventDefault();
      closeTaskDetail(taskDetailClose.closest('[data-task-detail-dialog]'));
      return;
    }

    var taskDetailComplete = closestElement(event.target, '[data-task-detail-complete]');
    if (taskDetailComplete) {
      event.preventDefault();
      completeTaskFromDetail(taskDetailComplete);
      return;
    }

    var taskDetailOpen = closestElement(event.target, '[data-task-detail-open]');
    if (taskDetailOpen) {
      event.preventDefault();
      openTaskDetail(taskDetailOpen);
      return;
    }

    var taskParentOpen = closestElement(event.target, '[data-task-parent-open]');
    if (taskParentOpen) {
      event.preventDefault();
      openParentTaskDetail(taskParentOpen);
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
      var deleteDetailDialog = closestElement(taskDeleteOpen, '[data-task-detail-dialog]');
      var deleteReturnFocus = taskDeleteOpen;
      if (deleteDetailDialog && deleteDetailDialog.open) {
        var deleteTaskRow = taskDeleteOpen.closest('[data-task-id]');
        deleteReturnFocus = deleteTaskRow ? deleteTaskRow.querySelector('[data-task-detail-open]') || taskDeleteOpen : taskDeleteOpen;
        closeTaskDetail(deleteDetailDialog);
      }
      openTaskDelete(taskDeleteOpen, deleteReturnFocus);
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
      openMobileNav(mobileNavOpen);
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
      closeDialog(dialog);
    }
  }, true);

  /* ── Hover preview handlers ──────────────────────────────── */

  document.addEventListener('mouseover', function (event) {
    var row = closestElement(event.target, '[data-task-id]');
    if (!row) { hideTaskHoverPreview(); return; }
    if (hoverPreviewState.el && hoverPreviewState.el.parentElement === row) return;
    hideTaskHoverPreview();
    hoverPreviewState.hideTimer = setTimeout(function () { showTaskHoverPreview(row, event); }, 500);
  });

  document.addEventListener('mouseout', function (event) {
    var row = closestElement(event.target, '[data-task-id]');
    if (!row) { hideTaskHoverPreview(); return; }
    var related = event.relatedTarget instanceof Element ? event.relatedTarget : null;
    if (related && row.contains(related)) return;
    hideTaskHoverPreview();
  });

  /* ── Date dropdown handlers (Alpine.js replacement) ─────── */

  function dateDropdownMenuItems(menu) {
    if (!menu) return [];
    return Array.prototype.filter.call(menu.querySelectorAll('[role="menuitem"]'), function (item) {
      return !item.hasAttribute('disabled');
    });
  }

  function closeDateDropdown(dropdown, restoreFocus) {
    if (!dropdown) return;
    var menu = dropdown.querySelector('.caldo-date-dropdown-menu');
    var trigger = dropdown.querySelector('[data-date-dropdown-trigger]');
    var customInput = dropdown.querySelector('[data-date-custom-input]');
    if (menu) menu.hidden = true;
    if (customInput) {
      customInput.hidden = true;
      customInput.value = '';
    }
    if (trigger) {
      trigger.setAttribute('aria-expanded', 'false');
      if (restoreFocus) trigger.focus();
    }
  }

  function openDateDropdown(dropdown, focusFirstItem) {
    if (!dropdown) return;
    var menu = dropdown.querySelector('.caldo-date-dropdown-menu');
    var trigger = dropdown.querySelector('[data-date-dropdown-trigger]');
    if (!menu || !trigger) return;
    closeAllDateDropdowns(dropdown, false);
    menu.hidden = false;
    trigger.setAttribute('aria-expanded', 'true');
    if (focusFirstItem) {
      var items = dateDropdownMenuItems(menu);
      if (items.length > 0) items[0].focus();
    }
  }

  function closeAllDateDropdowns(except, restoreFocus) {
    var openMenus = document.querySelectorAll('.caldo-date-dropdown-menu:not([hidden])');
    for (var i = 0; i < openMenus.length; i++) {
      var menu = openMenus[i];
      var dropdown = menu.closest('.caldo-date-dropdown');
      if (dropdown && dropdown !== except) {
        closeDateDropdown(dropdown, restoreFocus);
      }
    }
  }

  function formatLocalCalendarDate(date) {
    var year = String(date.getFullYear());
    var month = String(date.getMonth() + 1).padStart(2, '0');
    var day = String(date.getDate()).padStart(2, '0');
    return year + '-' + month + '-' + day;
  }

  function openCustomDateInput(trigger) {
    var dropdown = trigger ? trigger.closest('.caldo-date-dropdown') : null;
    var input = dropdown ? dropdown.querySelector('[data-date-custom-input]') : null;
    if (!input) return;
    input.hidden = false;
    input.focus();
    if (typeof input.showPicker === 'function') {
      try {
        input.showPicker();
      } catch (_) {
        // The focused visible input remains a usable fallback.
      }
    }
  }

  document.addEventListener('click', function (event) {
    var trigger = closestElement(event.target, '[data-date-dropdown-trigger]');
    if (trigger) {
      event.preventDefault();
      event.stopPropagation();
      var dropdown = trigger.closest('.caldo-date-dropdown');
      var menu = dropdown.querySelector('.caldo-date-dropdown-menu');
      var isOpen = !menu.hidden;
      if (isOpen) {
        closeDateDropdown(dropdown, false);
      } else {
        openDateDropdown(dropdown, false);
      }
      return;
    }

    var action = closestElement(event.target, '[data-date-dropdown-action]');
    if (action) {
      event.preventDefault();
      event.stopPropagation();
      var dropdown = action.closest('.caldo-date-dropdown');
      if (!dropdown) return;
      var form = dropdown.querySelector('form');
      if (!form) return;
      var input = form.querySelector('[data-date-hidden-input]');
      if (!input) return;

      var days = action.getAttribute('data-date-days');
      if (days === 'clear') {
        input.value = '';
      } else {
        var selectedDate = new Date();
        var selectedDay = selectedDate.getDay();
        if (days === 'next-monday') {
          selectedDate.setDate(selectedDate.getDate() + (((1 + 7 - selectedDay) % 7) || 7));
        } else if (days === 'next-weekend') {
          selectedDate.setDate(selectedDate.getDate() + (((6 + 7 - selectedDay) % 7) || 7));
        } else {
          selectedDate.setDate(selectedDate.getDate() + (parseInt(days, 10) || 0));
        }
        input.value = formatLocalCalendarDate(selectedDate);
      }

      form.requestSubmit();
      closeDateDropdown(dropdown, true);
      return;
    }

    var customTrigger = closestElement(event.target, '[data-date-custom-trigger]');
    if (customTrigger) {
      event.preventDefault();
      event.stopPropagation();
      openCustomDateInput(customTrigger);
      return;
    }

    var customInput = closestElement(event.target, '[data-date-custom-input]');
    if (!customInput && !event.target.matches('[data-date-custom-input]')) {
      if (!closestElement(event.target, '.caldo-date-dropdown')) {
        closeAllDateDropdowns(null);
      }
    }
  }, true);

  document.addEventListener('change', function (event) {
    var input = event.target;
    if (!input.matches || !input.matches('[data-date-custom-input]')) return;
    var dropdown = input.closest('.caldo-date-dropdown');
    if (!dropdown) return;
    var form = dropdown.querySelector('form');
    if (!form) return;
    var hidden = form.querySelector('[data-date-hidden-input]');
    if (!hidden) return;
    hidden.value = input.value;
    form.requestSubmit();
    closeDateDropdown(dropdown, true);
  }, true);

  document.addEventListener('keydown', function (event) {
    var menu = closestElement(event.target, '.caldo-date-dropdown-menu:not([hidden])');
    if (menu) {
      var dropdown = menu.closest('.caldo-date-dropdown');
      var items = dateDropdownMenuItems(menu);
      var activeItem = closestElement(event.target, '[role="menuitem"]');
      var activeIndex = items.indexOf(activeItem);
      var nextIndex = -1;
      if (event.key === 'ArrowDown') {
        nextIndex = activeIndex < 0 ? 0 : (activeIndex + 1) % items.length;
      } else if (event.key === 'ArrowUp') {
        nextIndex = activeIndex <= 0 ? items.length - 1 : activeIndex - 1;
      } else if (event.key === 'Home') {
        nextIndex = 0;
      } else if (event.key === 'End') {
        nextIndex = items.length - 1;
      }
      if (nextIndex >= 0 && items[nextIndex]) {
        event.preventDefault();
        items[nextIndex].focus();
        return;
      }
      if ((event.key === 'Enter' || event.key === ' ' || event.key === 'Spacebar') && activeItem) {
        event.preventDefault();
        activeItem.click();
        return;
      }
      if (event.key === 'Escape') {
        event.preventDefault();
        closeDateDropdown(dropdown, true);
        return;
      }
      if (event.key === 'Tab') {
        closeDateDropdown(dropdown, false);
      }
      return;
    }

    if (event.key === 'Escape') {
      var openMenu = document.querySelector('.caldo-date-dropdown-menu:not([hidden])');
      if (openMenu) {
        event.preventDefault();
        closeDateDropdown(openMenu.closest('.caldo-date-dropdown'), true);
      }
    }
  });

  window.addEventListener('resize', function () {
    if (window.innerWidth >= 768) {
      closeMobileNav();
    }
  });

  window.addEventListener('focus', scheduleFocusRefresh);
  window.addEventListener('pageshow', function () {
    window.setTimeout(function () {
      initializeTaskCompletionControls(document);
    }, 0);
  });
  document.addEventListener('visibilitychange', function () {
    if (document.visibilityState === 'visible') {
      scheduleFocusRefresh();
    }
  });

  document.addEventListener('keydown', function (event) {
    if (event.key === 'Tab' && trapFocusInDialog(topOpenDialog(), event)) {
      return;
    }

    var quickAddDialog = closestElement(event.target, '[data-quick-add-overlay]');
    if (quickAddDialog && quickAddDialog.open) {
      if (event.key === 'Escape') {
        event.preventDefault();
        closeQuickAddOverlay(quickAddDialog, true);
        return;
      }
      if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
        event.preventDefault();
        submitQuickAddKeyboardStep(quickAddDialog);
        return;
      }
      if (moveQuickAddTokenFocus(event.target, event.key)) {
        event.preventDefault();
        return;
      }
    }

    var inlineCreate = closestElement(event.target, '[data-inline-task-create]');
    if (inlineCreate) {
      if (event.key === 'Escape') {
        event.preventDefault();
        closeInlineCreate(inlineCreate);
        return;
      }
      if (event.key === 'Enter' && closestElement(event.target, '[data-inline-task-create-title]')) {
        event.preventDefault();
        submitForm(closestElement(event.target, '[data-inline-task-create-form]'));
        return;
      }
    }

    var inlineEditForm = closestElement(event.target, '[data-inline-task-edit-form]');
    if (inlineEditForm) {
      if (event.key === 'Escape') {
        if (closestElement(event.target, '[data-inline-task-priority-select]')) return;
        event.preventDefault();
        closeInlineEdit(inlineEditForm.closest('[data-task-id]'), inlineEditForm);
        return;
      }
      if (event.key === 'Enter' && !event.shiftKey) {
        if (closestElement(event.target, '[data-inline-task-edit-autosave]')) return;
        event.preventDefault();
        submitForm(inlineEditForm);
        return;
      }
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

    var mobileDialog = closestElement(event.target, '[data-mobile-nav-dialog]');
    if (mobileDialog && mobileDialog.open && event.key === 'Escape' && typeof mobileDialog.close !== 'function') {
      event.preventDefault();
      closeMobileNav();
      return;
    }

    var helpDialog = closestElement(event.target, '[data-shortcut-help-dialog]');
    if (helpDialog && helpDialog.open && event.key === 'Escape' && typeof helpDialog.close !== 'function') {
      event.preventDefault();
      closeDialog(helpDialog);
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
      openQuickAddOverlay(null);
      return;
    }

    if (key === 'r') {
      event.preventDefault();
      requestManualSync();
      return;
    }

    if (key === 'd') {
      event.preventDefault();
      openFocusedTaskDateDropdown();
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

  consumeRememberedWriteStatus();
  initializeFormErrors();
  initializeTaskCompletionControls(document);
  bindApplicationDialogs();
  bindQuickAddOverlay();
  initializeThemeController();
  initializeSetupImport();
  initializeUndoSurface();
  initializeLocalDateTimes();
  initializeConflictManualPreviews();
  initializeConflictSplitConfirmations();

  /* ── Animate first task row after a write-reload ─────────── */

  function animateFirstTaskRowAfterWrite() {
    if (!tabState.writeStatus) return;
    var firstRow = document.querySelector('.caldo-task-list [data-task-id]');
    if (firstRow) animateNewTaskRow(firstRow);
  }

  animateFirstTaskRowAfterWrite();
})();
