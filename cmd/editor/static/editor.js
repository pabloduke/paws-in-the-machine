(() => {
  let dirty = false;

  document.addEventListener("input", (event) => {
    if (event.target.closest("[data-dirty-form]")) {
      dirty = true;
    }
  });

  document.addEventListener("change", (event) => {
    if (event.target.closest("[data-dirty-form]")) {
      dirty = true;
    }
  });

  document.addEventListener("click", (event) => {
    const destination = event.target.closest("[data-discard-check]");
    if (!destination || !dirty) {
      return;
    }
    if (!window.confirm("Discard unsaved content changes?")) {
      event.preventDefault();
      event.stopImmediatePropagation();
      return;
    }
    dirty = false;
  }, true);

  document.addEventListener("contentSaved", () => {
    dirty = false;
  });

  document.addEventListener("contentDeleted", () => {
    dirty = false;
  });

  window.addEventListener("beforeunload", (event) => {
    if (!dirty) {
      return;
    }
    event.preventDefault();
    event.returnValue = "";
  });
})();
