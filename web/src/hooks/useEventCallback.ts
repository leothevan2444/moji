import { useCallback, useLayoutEffect, useRef } from "react";

// Event handlers keep their identity while reading the last committed render.
// Updating in a layout effect avoids exposing an abandoned render's closures.
export function useEventCallback<Args extends unknown[], Result>(callback: (...args: Args) => Result) {
  const latest = useRef(callback);
  useLayoutEffect(() => { latest.current = callback; });
  return useCallback((...args: Args) => latest.current(...args), []);
}
