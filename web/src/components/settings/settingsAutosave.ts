export type SettingsObject = Record<string, unknown>;
export type SaveState = "idle" | "pending" | "saving" | "saved" | "error";
const object = (value: unknown): value is SettingsObject => value !== null && typeof value === "object" && !Array.isArray(value);
const equal = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b);

/** Objects merge recursively; arrays are atomic ordered configuration values. */
export function settingsDelta(before: SettingsObject, after: SettingsObject): SettingsObject {
  const delta: SettingsObject = {};
  for (const [key, value] of Object.entries(after)) {
    if (equal(before[key], value)) continue;
    if (object(value) && object(before[key])) {
      const nested = settingsDelta(before[key], value);
      if (Object.keys(nested).length) delta[key] = nested;
    } else delta[key] = value;
  }
  return delta;
}
export function mergeSettings(before: SettingsObject, delta: SettingsObject): SettingsObject {
  const next = { ...before };
  for (const [key, value] of Object.entries(delta)) next[key] = object(value) && object(before[key]) ? mergeSettings(before[key], value) : value;
  return next;
}

export interface AutosaveSnapshot<T> { value: T; state: SaveState; error: unknown; pending: SettingsObject }
export class SettingsAutosave<T> {
  snapshot: AutosaveSnapshot<T>;
  private baseline: SettingsObject;
  private timer?: ReturnType<typeof setTimeout>;
  private saving = false;
  private due = 0;
  private listeners = new Set<() => void>();
  encode: (value: T) => SettingsObject;
  persist: (patch: SettingsObject) => Promise<void>;
  onSaved?: (value: T, patch: SettingsObject) => void;
  constructor(initial: T, encode: (value: T) => SettingsObject, persist: (patch: SettingsObject) => Promise<void>) {
    this.encode = encode; this.persist = persist; this.baseline = encode(initial);
    this.snapshot = { value: initial, state: "idle", error: null, pending: {} };
  }
  subscribe = (listener: () => void) => { this.listeners.add(listener); return () => { this.listeners.delete(listener); }; };
  getSnapshot = () => this.snapshot;
  private publish(patch: Partial<AutosaveSnapshot<T>>) {
    this.snapshot = { ...this.snapshot, ...patch };
    this.listeners.forEach(listener => listener());
  }
  initialize(initial: T) {
    if (this.saving || ["pending", "error"].includes(this.snapshot.state)) return;
    this.baseline = this.encode(initial);
    this.publish({ value: initial });
  }
  edit(next: T | ((value: T) => T)) {
    const value = typeof next === "function" ? (next as (value: T) => T)(this.snapshot.value) : next;
    const pending = settingsDelta(this.baseline, this.encode(value));
    this.publish({ value, pending, state: this.saving ? "saving" : Object.keys(pending).length ? "pending" : "idle", error: null });
    this.due = Date.now() + 500;
    if (this.timer) clearTimeout(this.timer);
    this.timer = setTimeout(() => void this.flush(), 500);
  }
  retry = () => { this.due = Date.now(); void this.flush(); };
  private async flush() {
    if (this.saving) return;
    const patch = settingsDelta(this.baseline, this.encode(this.snapshot.value));
    if (!Object.keys(patch).length) { this.publish({ state: "idle", pending: {} }); return; }
    this.saving = true;
    const sentValue = this.snapshot.value;
    this.publish({ state: "saving", error: null, pending: patch });
    try {
      await this.persist(patch);
      this.baseline = mergeSettings(this.baseline, patch);
      const pending = settingsDelta(this.baseline, this.encode(this.snapshot.value));
      this.publish({ pending, state: Object.keys(pending).length ? "pending" : "saved" });
      this.onSaved?.(sentValue, patch);
    } catch (error) {
      this.publish({ state: "error", error, pending: settingsDelta(this.baseline, this.encode(this.snapshot.value)) });
    } finally {
      this.saving = false;
      // New edits are serialized after the in-flight request, still respecting
      // their own quiet period. Failed requests wait for retry or another edit.
      if (this.snapshot.state === "pending") {
        if (this.timer) clearTimeout(this.timer);
        this.timer = setTimeout(() => void this.flush(), Math.max(0, this.due - Date.now()));
      }
    }
  }
}
