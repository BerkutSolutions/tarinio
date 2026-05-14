import { GRID, getLayoutItem } from "./dashboard.layout-core.js";

function applyFrameGeometry(boardNode, layout, frameNode) {
  const item = getLayoutItem(layout, String(frameNode.dataset.widgetId || ""));
  if (!item) {
    return;
  }
  const mobile = window.matchMedia("(max-width: 900px)").matches;
  if (mobile) {
    frameNode.style.left = "";
    frameNode.style.top = "";
    frameNode.style.width = "";
    frameNode.style.height = "";
  } else {
    frameNode.style.left = `${item.x}px`;
    frameNode.style.top = `${item.y}px`;
    frameNode.style.width = `${item.width}px`;
    frameNode.style.height = `${item.height}px`;
  }
}

function applyAllGeometry(boardNode, layout) {
  const frameNodes = Array.from(boardNode.querySelectorAll(".dashboard-frame"));
  const visibleIDs = new Set(frameNodes.map((node) => String(node.dataset.widgetId || "")));
  frameNodes.forEach((frameNode) => {
    applyFrameGeometry(boardNode, layout, frameNode);
  });
  const visibleLayout = layout.filter((entry) => visibleIDs.has(entry.id));
  const maxBottom = visibleLayout.reduce((acc, current) => Math.max(acc, current.y + current.height), 0);
  boardNode.style.minHeight = `${Math.max(560, maxBottom + GRID)}px`;
}

export {
  applyFrameGeometry,
  applyAllGeometry
};
