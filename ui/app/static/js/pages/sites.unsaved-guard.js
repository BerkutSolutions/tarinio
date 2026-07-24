export function installUnsavedChangesGuard({ container, state, message, confirmAction, navigate }) {
  if (state.unsavedBeforeUnloadHandler) window.removeEventListener("beforeunload", state.unsavedBeforeUnloadHandler);
  const beforeUnload = (event) => {
    if (!state.editorDirty) return;
    event.preventDefault();
    event.returnValue = "";
  };
  state.unsavedBeforeUnloadHandler = beforeUnload;
  window.addEventListener("beforeunload", beforeUnload);

  const form = container.querySelector("#service-editor-form");
  if (form) form.dataset.unsaved = state.editorDirty ? "true" : "false";
  const markDirty = () => {
    state.editorDirty = true;
    if (form) form.dataset.unsaved = "true";
  };
  form?.addEventListener("input", markDirty);
  form?.addEventListener("change", markDirty);

  const clear = () => {
    state.editorDirty = false;
    if (form) form.dataset.unsaved = "false";
    if (state.unsavedBeforeUnloadHandler) window.removeEventListener("beforeunload", state.unsavedBeforeUnloadHandler);
    state.unsavedBeforeUnloadHandler = null;
  };
  const back = () => {
    if (state.editorDirty && !confirmAction(message)) return false;
    clear();
    navigate();
    return true;
  };
  return { back, clear };
}
