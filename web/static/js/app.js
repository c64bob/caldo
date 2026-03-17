(function () {
  let activeTaskIndex = -1;

  function initDatepickers(root) {
    if (typeof AirDatepicker === "undefined") {
      return;
    }

    const scope = root || document;
    const inputs = scope.querySelectorAll("input[data-air-datepicker]");
    inputs.forEach(function (input) {
      if (input.dataset.airDatepickerReady === "1") {
        return;
      }

      new AirDatepicker(input, {
        autoClose: true,
        dateFormat: "yyyy-MM-dd",
      });
      input.dataset.airDatepickerReady = "1";
    });
  }

  function taskRows() {
    return Array.from(document.querySelectorAll("[data-task-row]"));
  }

  function setActiveTaskRow(row, ensureVisible) {
    const rows = taskRows();
    rows.forEach(function (candidate) {
      candidate.classList.toggle("active-row", candidate === row);
    });

    if (!row) {
      activeTaskIndex = -1;
      return;
    }

    activeTaskIndex = rows.indexOf(row);
    row.focus();
    if (ensureVisible) {
      row.scrollIntoView({ block: "nearest" });
    }
  }

  function ensureActiveRow() {
    const rows = taskRows();
    if (rows.length === 0) {
      setActiveTaskRow(null, false);
      return null;
    }

    if (activeTaskIndex < 0 || activeTaskIndex >= rows.length) {
      setActiveTaskRow(rows[0], false);
      return rows[0];
    }

    return rows[activeTaskIndex];
  }

  function focusById(id) {
    const el = document.getElementById(id);
    if (!el) {
      return;
    }
    el.focus();
    if (typeof el.select === "function") {
      el.select();
    }
  }

  function switchView(shortcut) {
    const link = document.querySelector('[data-shortcut-view="' + shortcut + '"]');
    if (link) {
      link.click();
    }
  }

  function switchToIndexedTab(index) {
    const activeViewLink = document.querySelector(".sidebar-link.active");
    const section = activeViewLink ? activeViewLink.closest(".sidebar-section") : null;
    const links = section ? section.querySelectorAll(".sidebar-link") : [];
    if (links.length >= index) {
      links[index - 1].click();
    }
  }

  function submitVisibleSummaryEditors() {
    const forms = document.querySelectorAll("td.task-name form");
    forms.forEach(function (form) {
      if (form.offsetParent === null) {
        return;
      }
      form.requestSubmit();
    });
  }

  function toggleShortcutOverlay(forceState) {
    const overlay = document.getElementById("shortcut-overlay");
    if (!overlay) {
      return;
    }

    const shouldOpen = typeof forceState === "boolean" ? forceState : overlay.hidden;
    overlay.hidden = !shouldOpen;
    overlay.setAttribute("aria-hidden", shouldOpen ? "false" : "true");
  }

  function isShortcutOverlayOpen() {
    const overlay = document.getElementById("shortcut-overlay");
    return !!(overlay && !overlay.hidden);
  }

  function isTaskSummaryEditorOpen() {
    return Array.from(document.querySelectorAll("td.task-name form")).some(function (form) {
      return form.offsetParent !== null;
    });
  }

  function isEditableTarget(target) {
    return !!(target && target.closest("input, textarea, select, [contenteditable='true']"));
  }

  function onKeydown(event) {
    if (event.defaultPrevented || event.ctrlKey || event.metaKey || event.altKey) {
      return;
    }

    if (isShortcutOverlayOpen()) {
      if (event.key === "Escape" || event.key === "?" || event.key === "Enter") {
        event.preventDefault();
      }
      toggleShortcutOverlay(false);
      return;
    }

    if (event.key === "Escape") {
      toggleShortcutOverlay(false);
      return;
    }

    if (isEditableTarget(event.target)) {
      return;
    }

    if (event.key === "?") {
      if (isTaskSummaryEditorOpen()) {
        return;
      }
      event.preventDefault();
      toggleShortcutOverlay();
      return;
    }

    if (/^[1-9]$/.test(event.key)) {
      event.preventDefault();
      switchToIndexedTab(Number(event.key));
      return;
    }

    const viewKeys = ["m", "o", "c", "d", "g", "p", "h", "e"];
    if (viewKeys.includes(event.key)) {
      event.preventDefault();
      switchView(event.key);
      return;
    }

    if (event.key === "n") {
      event.preventDefault();
      focusById("quick-add-input");
      return;
    }

    if (event.key === "f") {
      event.preventDefault();
      focusById("global-search-input");
      return;
    }

    if (event.key === "s") {
      event.preventDefault();
      submitVisibleSummaryEditors();
      return;
    }

    const row = ensureActiveRow();
    if (!row) {
      return;
    }

    if (event.key === "ArrowDown") {
      event.preventDefault();
      const rows = taskRows();
      const next = Math.min(activeTaskIndex + 1, rows.length - 1);
      setActiveTaskRow(rows[next], true);
      return;
    }

    if (event.key === "ArrowUp") {
      event.preventDefault();
      const rows = taskRows();
      const prev = Math.max(activeTaskIndex - 1, 0);
      setActiveTaskRow(rows[prev], true);
      return;
    }

    if (event.key === " ") {
      event.preventDefault();
      const checkbox = row.querySelector(".js-complete-toggle");
      if (checkbox) {
        checkbox.click();
      }
      return;
    }

    if (event.key === "x") {
      event.preventDefault();
      row.classList.toggle("task-batch-selected");
      return;
    }

    if (event.key === "Delete") {
      event.preventDefault();
      const summary = (row.querySelector(".task-name span") || {}).textContent || "diese Aufgabe";
      if (!window.confirm('Aufgabe "' + summary.trim() + '" wirklich löschen?')) {
        return;
      }
      const deleteForm = row.parentElement.querySelector(".js-delete-task-form");
      if (deleteForm) {
        deleteForm.requestSubmit();
      }
    }
  }

  function initTaskKeyboard(root) {
    const scope = root || document;
    scope.querySelectorAll("[data-task-row]").forEach(function (row) {
      if (row.dataset.keyboardBound === "1") {
        return;
      }
      row.addEventListener("click", function () {
        setActiveTaskRow(row, false);
      });
      row.dataset.keyboardBound = "1";
    });

    ensureActiveRow();
  }

  document.addEventListener("DOMContentLoaded", function () {
    initDatepickers(document);
    initTaskKeyboard(document);
    document.addEventListener("keydown", onKeydown);

    const overlay = document.getElementById("shortcut-overlay");
    const closeButton = document.getElementById("shortcut-overlay-close");
    toggleShortcutOverlay(false);

    if (overlay) {
      overlay.addEventListener("click", function (event) {
        if (event.target === overlay) {
          toggleShortcutOverlay(false);
        }
      });
    }

    if (closeButton) {
      closeButton.addEventListener("click", function () {
        toggleShortcutOverlay(false);
      });
    }
  });

  document.addEventListener("htmx:afterSwap", function (e) {
    if (typeof Alpine !== "undefined") {
      Alpine.initTree(e.detail.target);
    }
    initDatepickers(e.detail.target);
    initTaskKeyboard(e.detail.target);
  });
})();
