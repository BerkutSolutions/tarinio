import { escapeHtml } from "../ui.js";
import { WIDGETS, getLayoutItem, snap, constrainDraggedItem, constrainItem, resolveOverlaps } from "./dashboard.layout-core.js";
import { applyAllGeometry } from "./dashboard.layout-geometry.js";

function createFrame(widget, ctx) {
  const frameNode = document.createElement("section");
  frameNode.className = "waf-card dashboard-frame";
  frameNode.dataset.widgetId = widget.id;
  frameNode.innerHTML = `
    <div class="waf-card-head dashboard-frame-header"><h3>${escapeHtml(ctx.t(widget.titleKey))}</h3></div>
    <div class="waf-card-body" data-widget-body="${escapeHtml(widget.id)}"><div class="waf-empty">${escapeHtml(ctx.t("common.loading"))}</div></div>
    <div class="frame-resize-handle resize-se" data-resize-dir="se" title="${escapeHtml(ctx.t("dashboard.action.resize"))}"></div>
    <div class="frame-resize-handle resize-e" data-resize-dir="e"></div>
    <div class="frame-resize-handle resize-s" data-resize-dir="s"></div>
    <div class="frame-resize-handle resize-w" data-resize-dir="w"></div>
    <div class="frame-resize-handle resize-n" data-resize-dir="n"></div>
  `;
  return frameNode;
}

function wireFrameInteractions(pageNode, boardNode, layout, frameNode, onLayoutMutated) {
  let dragState = null;
  const handlePointerDown = (event) => {
    if (pageNode.dataset.editMode !== "1" || window.matchMedia("(max-width: 900px)").matches) {
      return;
    }
    const item = getLayoutItem(layout, String(frameNode.dataset.widgetId || ""));
    if (!item) {
      return;
    }
    const resizeHandle = event.target.closest("[data-resize-dir]");
    const header = event.target.closest(".dashboard-frame-header");
    if (!resizeHandle && !header) {
      return;
    }
    event.preventDefault();
    dragState = {
      id: item.id,
      pointerID: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      baseX: item.x,
      baseY: item.y,
      baseWidth: item.width,
      baseHeight: item.height,
      changed: false,
      mode: resizeHandle ? `resize-${String(resizeHandle.dataset.resizeDir || "se")}` : "move"
    };
    frameNode.classList.add("dragging");
    frameNode.setPointerCapture(event.pointerId);
  };

  const handlePointerMove = (event) => {
    if (!dragState || dragState.pointerID !== event.pointerId) {
      return;
    }
    const item = getLayoutItem(layout, dragState.id);
    if (!item) {
      return;
    }
    const boardWidth = Math.max(600, boardNode.clientWidth);
    const dx = event.clientX - dragState.startX;
    const dy = event.clientY - dragState.startY;
    if (dragState.mode === "move") {
      item.x = snap(dragState.baseX + dx);
      item.y = snap(dragState.baseY + dy);
    } else {
      let x = dragState.baseX;
      let y = dragState.baseY;
      let width = dragState.baseWidth;
      let height = dragState.baseHeight;
      if (dragState.mode.includes("e")) width = snap(dragState.baseWidth + dx);
      if (dragState.mode.includes("s")) height = snap(dragState.baseHeight + dy);
      if (dragState.mode.includes("w")) {
        x = snap(dragState.baseX + dx);
        width = snap(dragState.baseWidth - dx);
      }
      if (dragState.mode.includes("n")) {
        y = snap(dragState.baseY + dy);
        height = snap(dragState.baseHeight - dy);
      }
      item.x = x;
      item.y = y;
      item.width = width;
      item.height = height;
    }
    dragState.changed = true;
    constrainDraggedItem(item, boardWidth, dragState.mode);
    applyAllGeometry(boardNode, layout);
  };

  const handlePointerEnd = (event) => {
    if (!dragState || dragState.pointerID !== event.pointerId) {
      return;
    }
    frameNode.classList.remove("dragging");
    frameNode.releasePointerCapture(event.pointerId);
    const boardWidth = Math.max(600, boardNode.clientWidth);
    layout.forEach((entry) => constrainItem(entry, boardWidth));
    resolveOverlaps(layout, dragState.id);
    applyAllGeometry(boardNode, layout);
    if (dragState.changed) onLayoutMutated?.();
    dragState = null;
  };

  frameNode.addEventListener("pointerdown", handlePointerDown);
  frameNode.addEventListener("pointermove", handlePointerMove);
  frameNode.addEventListener("pointerup", handlePointerEnd);
  frameNode.addEventListener("pointercancel", handlePointerEnd);
}

export {
  createFrame,
  wireFrameInteractions
};
