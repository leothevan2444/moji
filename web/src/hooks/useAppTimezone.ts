import { useSyncExternalStore } from "react";
import { getAppTimezone, subscribeAppTimezone } from "../utils/appTimezone";

export function useAppTimezone() {
  return useSyncExternalStore(subscribeAppTimezone, getAppTimezone);
}
