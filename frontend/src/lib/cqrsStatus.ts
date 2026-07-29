export function markQueryProjectionSampled() {
  sessionStorage.setItem("artificialflow.lastQueryProjectionAt", String(Date.now()));
}

export function readQueryProjectionSampledAt(): number | null {
  const raw = sessionStorage.getItem("artificialflow.lastQueryProjectionAt");
  return raw ? Number(raw) : null;
}
