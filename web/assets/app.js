(function () {
  'use strict';

  var navState = { pendingView: null };
  var tabID = newTabID();

  function newTabID() {
    if (window.crypto && typeof window.crypto.randomUUID === 'function') {
      return window.crypto.randomUUID();
    }
    return 'tab-' + Date.now().toString(36) + '-' + Math.random().toString(36).slice(2);
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

  function bindTaskDetail(dialog) {
    if (!dialog || dialog.dataset.taskDetailBound === 'true') return;
    dialog.dataset.taskDetailBound = 'true';
    dialog.addEventListener('close', function () {
      var form = dialog.querySelector('[data-task-detail-form]');
      if (form) form.reset();
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
    if (form) form.reset();
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

  function taskDeleteFailureMessage(status) {
    if (status === 409) {
      return 'Aufgabe konnte nicht gelöscht werden. Aufgabe prüfen.';
    }
    if (status === 502) {
      return 'Aufgabe konnte nicht auf dem CalDAV-Server gelöscht werden.';
    }
    return 'Aufgabe konnte nicht gelöscht werden.';
  }

  function showUndoNotification(message) {
    var root = document.getElementById('notifications');
    if (!root) return;
    root.textContent = '';

    var toast = document.createElement('div');
    toast.className = 'caldo-toast caldo-undo-toast';
    toast.setAttribute('role', 'status');

    var text = document.createElement('span');
    text.setAttribute('data-undo-status', '');
    text.textContent = message;

    var button = document.createElement('button');
    button.type = 'button';
    button.className = 'caldo-button caldo-button-secondary caldo-undo-button';
    button.setAttribute('data-undo-action', '');
    button.textContent = 'Rückgängig';

    toast.appendChild(text);
    toast.appendChild(button);
    root.appendChild(toast);
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

  function executeUndo(button) {
    if (!button || button.disabled) return;
    button.disabled = true;
    setWriteStatus('pending', 'Speichern ...');
    window.fetch('/tasks/undo', {
      method: 'POST',
      credentials: 'same-origin',
      headers: {
        'X-CSRF-Token': csrfToken(),
        'X-Tab-ID': tabID
      }
    }).then(function (response) {
      if (!response.ok) {
        throw response;
      }
      setWriteStatus('saved', 'Gespeichert');
      window.location.reload();
    }).catch(function () {
      button.disabled = false;
      setWriteStatus('error', 'Rückgängig fehlgeschlagen. Änderungen prüfen.');
      setUndoNotificationError(button, 'Rückgängig fehlgeschlagen.');
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

    var successfulTaskDelete = closestElement(event.detail && event.detail.elt, '[data-task-delete-form]');
    if (successfulTaskDelete) {
      var deleteRow = successfulTaskDelete.closest('[data-task-id]');
      closeTaskDelete(successfulTaskDelete.closest('[data-task-delete-dialog]'));
      if (deleteRow) {
        deleteRow.remove();
      }
      setWriteStatus(null, '');
      showUndoNotification('Aufgabe gelöscht.');
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
    if (navState.pendingView) {
      if (key === 't') {
        event.preventDefault();
        goTo('/today');
      } else if (key === 'u') {
        event.preventDefault();
        goTo('/upcoming');
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

    if (key === 's' || key === '/') {
      event.preventDefault();
      goTo('/search');
      return;
    }

    if (key === '?') {
      event.preventDefault();
      toggleHelpDialog();
    }
  });
})();
