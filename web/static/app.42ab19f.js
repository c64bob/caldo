(function () {
  'use strict';

  var navState = { pendingView: null };

  function isTypingTarget(target) {
    if (!target || !(target instanceof Element)) return false;
    if (target.closest('[contenteditable="true"]')) return true;
    var tag = target.tagName;
    return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || target.isContentEditable;
  }

  function goTo(path) {
    window.location.assign(path);
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

  document.body.addEventListener('htmx:beforeRequest', function (event) {
    var method = ((event.detail && event.detail.requestConfig && event.detail.requestConfig.verb) || '').toUpperCase();
    if (!method || method === 'GET') return;
    writeState.pendingRequests += 1;
    setWriteStatus('pending', 'Speichern …');
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
      return;
    }

    if (writeState.pendingRequests === 0) {
      setWriteStatus('saved', 'Gespeichert');
      clearSavedStatusSoon();
    }
  });

  window.addEventListener('beforeunload', function (event) {
    if (writeState.pendingRequests <= 0) return;
    event.preventDefault();
    event.returnValue = '';
  });

  document.addEventListener('click', function (event) {
    var closeButton = event.target.closest('[data-shortcut-help-close]');
    if (!closeButton) return;
    var dialog = closeButton.closest('dialog');
    if (dialog) {
      dialog.close();
    }
  });

  document.addEventListener('keydown', function (event) {
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
