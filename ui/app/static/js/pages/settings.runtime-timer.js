let runtimeAutoCheckTimer = null;

export function clearRuntimeAutoCheckTimer() {
  if (runtimeAutoCheckTimer) {
    window.clearInterval(runtimeAutoCheckTimer);
    runtimeAutoCheckTimer = null;
  }
}

export function setRuntimeAutoCheckTimer(timerID) {
  runtimeAutoCheckTimer = timerID;
}
