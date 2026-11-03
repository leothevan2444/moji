import i18n from "../i18n/i18n";
import { useEffect, useRef, useState } from "react";
import { useMutation } from "urql";
import { TestIndexerDocument, type TestIndexerInput, type TestIndexerMutation } from "../graphql/generated/graphql";

export function useIndexerCheck(input: TestIndexerInput) {
  // Includes draft secrets; this key stays in memory and must never be logged or persisted.
  const scope = JSON.stringify(input);
  const active = useRef(scope);
  active.current = scope;
  const controller = useRef<AbortController | null>(null);
  const [, execute] = useMutation(TestIndexerDocument);
  const [state, setState] = useState<{
    scope: string;
    busy: boolean;
    report?: TestIndexerMutation["testIndexer"];
    error?: unknown;
  }>({ scope, busy: false });

  useEffect(() => () => {
    controller.current?.abort();
    controller.current = null;
  }, [scope]);

  const cancel = () => {
    controller.current?.abort();
    controller.current = null;
    setState({ scope, busy: false });
  };
  const run = async () => {
    if (controller.current) return;
    const request = new AbortController();
    controller.current = request;
    setState({ scope, busy: true });
    try {
      const result = await execute({ input }, { fetchOptions: { signal: request.signal } });
      if (request.signal.aborted || active.current !== scope) return;
      if (result.error) throw result.error;
      if (!result.data?.testIndexer) throw new Error(i18n.t("indexerCheck.failed"));
      setState({ scope, busy: false, report: result.data.testIndexer });
    } catch (error) {
      if (!request.signal.aborted && active.current === scope) setState({ scope, busy: false, error });
    } finally {
      if (controller.current === request) controller.current = null;
    }
  };
  return {
    run, cancel,
    busy: state.scope === scope && state.busy,
    report: state.scope === scope ? state.report : undefined,
    error: state.scope === scope ? state.error : undefined,
  };
}
