// Window events used to keep the TUN-service UI in sync across components
// (the service panel in settings and the TUN toggle in the main menu live in
// different component trees).

// Something changed the state of the privileged px-service — re-read its
// installed/running/admin status.
export const SERVICE_STATUS_UPDATED_EVENT = 'service-status-updated';

// The privileged service was removed, so TUN cannot be on any more: the toggle
// must drop back to "off" instead of keeping a stale "on" state.
export const TUN_FORCE_OFF_EVENT = 'tun-force-off';

export function notifyServiceStatusChanged(): void {
    window.dispatchEvent(new CustomEvent(SERVICE_STATUS_UPDATED_EVENT));
}

export function notifyTunForcedOff(): void {
    window.dispatchEvent(new CustomEvent(TUN_FORCE_OFF_EVENT));
}
