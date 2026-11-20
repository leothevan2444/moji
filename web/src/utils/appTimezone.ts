// The server resolves `auto`; browser timezone must never replace Moji's setting.
let timezone = "UTC";
const listeners = new Set<() => void>();
export function getAppTimezone() { return timezone; }
export function subscribeAppTimezone(listener: () => void) {
  listeners.add(listener);
  return () => { listeners.delete(listener); };
}
export function setAppTimezone(value: string) {
  try { new Intl.DateTimeFormat("en", { timeZone: value }); } catch { return; }
  if (value === timezone) return;
  timezone = value;
  listeners.forEach(listener => listener());
}
