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
