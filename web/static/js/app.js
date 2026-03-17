(function () {
  const toggle = document.getElementById("sidebar-toggle");
  if (!toggle) {
    return;
  }

  toggle.addEventListener("click", function () {
    if (window.matchMedia("(max-width: 960px)").matches) {
      document.body.classList.toggle("sidebar-open");
      return;
    }
    document.body.classList.toggle("sidebar-collapsed");
  });
})();
