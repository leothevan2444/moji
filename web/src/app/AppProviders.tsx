import type { PropsWithChildren } from "react";
import { Provider } from "urql";
import { client } from "../graphql/client";

export function AppProviders({ children }: PropsWithChildren) {
  return <Provider value={client}>{children}</Provider>;
}
