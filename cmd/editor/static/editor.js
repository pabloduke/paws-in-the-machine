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

  // Ordinary POST forms navigate after saving; that is an intentional submit.
  document.addEventListener("submit", (event) => {
    if (event.target.matches("[data-dirty-form]") && !event.target.hasAttribute("hx-post")) {
      dirty = false;
    }
  });

  // Delegated handlers also cover grids replaced by HTMX navigation.
  let dragging = null;
  const clearDrag = () => {
    document.querySelectorAll(".dragging, .drop-target").forEach((cell) => {
      cell.classList.remove("dragging", "drop-target");
    });
    dragging = null;
  };
  document.addEventListener("dragstart", (event) => {
    const cell = event.target.closest("[data-child-id][draggable]");
    if (!cell || !window.htmx) return;
    if (dirty && !window.confirm("Discard unsaved content changes?")) {
      event.preventDefault();
      return;
    }
    dragging = {cell, grid: cell.closest("[data-move-url]")};
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("text/plain", cell.dataset.childId);
    cell.classList.add("dragging");
  });
  document.addEventListener("dragover", (event) => {
    const cell = event.target.closest(".room-cell.empty");
    if (!dragging || !cell || cell.closest("[data-move-url]") !== dragging.grid) return;
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
    document.querySelectorAll(".drop-target").forEach((el) => el.classList.remove("drop-target"));
    cell.classList.add("drop-target");
  });
  document.addEventListener("dragleave", (event) => {
    event.target.closest(".drop-target")?.classList.remove("drop-target");
  });
  document.addEventListener("drop", (event) => {
    if (!dragging) return;
    event.preventDefault();
    const cell = event.target.closest(".room-cell.empty");
    if (cell && cell.closest("[data-move-url]") === dragging.grid) {
      const source = dragging.cell.dataset;
      dirty = false;
      window.htmx.ajax("POST", dragging.grid.dataset.moveUrl, {
        target: "#workspace", swap: "innerHTML",
        values: {action: "move-child", child_id: source.childId,
          source_x: source.x, source_y: source.y, x: cell.dataset.x, y: cell.dataset.y, z: dragging.grid.dataset.level}
      });
    }
    clearDrag();
  });
  document.addEventListener("dragend", clearDrag);
  const focusCreate = () => {
    document.querySelector("[data-focus-create] input[name=child_name]")?.focus({preventScroll: true});
  };
  document.addEventListener("htmx:afterSettle", focusCreate);
  document.addEventListener("DOMContentLoaded", focusCreate);

  window.addEventListener("beforeunload", (event) => {
    if (!dirty) {
      return;
    }
    event.preventDefault();
    event.returnValue = "";
  });
})();
