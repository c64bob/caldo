(function () {
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

  document.addEventListener("DOMContentLoaded", function () {
    initDatepickers(document);
  });

  document.addEventListener("htmx:afterSwap", function (e) {
    if (typeof Alpine !== "undefined") {
      Alpine.initTree(e.detail.target);
    }
    initDatepickers(e.detail.target);
  });
})();
