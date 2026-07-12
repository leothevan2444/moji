import type { PropsWithChildren } from "react";
import { Provider } from "urql";
import { client } from "../graphql/client";
import { LocaleProvider } from "../i18n/LocaleProvider";
import "../i18n/i18n";

export function AppProviders({ children }: PropsWithChildren) {
  return <LocaleProvider><Provider value={client}>{children}</Provider></LocaleProvider>;
}
