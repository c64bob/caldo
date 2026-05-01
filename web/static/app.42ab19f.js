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
