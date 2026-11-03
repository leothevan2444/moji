import { useEffect, useRef, useState } from "react";
import { useClient, useMutation } from "urql";
import {
  AnswerIndexerChallengeDocument, BeginIndexerLoginDocument, CancelIndexerLoginDocument,
  IndexerLoginDocument, IndexerLoginResultFragmentDoc, type IndexerLoginInput, type IndexerLoginResultFragment,
} from "../graphql/generated/graphql";
import { useFragment } from "../graphql/generated/fragment-masking";
import i18n from "../i18n/i18n";

type Report = IndexerLoginResultFragment;
type Operation = "restore" | "begin" | "answer" | "cancel";
interface State { scope: string; busy: boolean; report?: Report; error?: unknown; cancelled?: boolean; operation?: Operation }

export function useIndexerLogin(input: IndexerLoginInput) {
  const scope = JSON.stringify(input);
  const active = useRef(scope);
  active.current = scope;
  const client = useClient();
  const [, beginMutation] = useMutation(BeginIndexerLoginDocument);
  const [, answerMutation] = useMutation(AnswerIndexerChallengeDocument);
  const [, cancelMutation] = useMutation(CancelIndexerLoginDocument);
  const request = useRef<AbortController | null>(null);
  const requestKind = useRef<Operation | null>(null);
  const [state, setState] = useState<State>({ scope, busy: true });
  const current = useRef(state);
  current.current = state;

  const perform = async (operation: Operation, answer?: string) => {
    if (request.current) return;
    const old = current.current.scope === scope ? current.current.report : undefined;
    if (operation === "answer" && old?.state !== "WAITING") return;
    const controller = new AbortController();
    request.current = controller;
    requestKind.current = operation;
    const options = { fetchOptions: { signal: controller.signal } };
    let retained = operation === "cancel" || (operation === "begin" && old?.state === "WAITING") ? old : undefined;
    setState({ scope, busy: true, report: old, operation });
    try {
      let report: Report | undefined;
      if (operation === "restore") {
        const result = await client.query(IndexerLoginDocument, { input }, { ...options, requestPolicy: "network-only" }).toPromise();
        if (result.error) throw result.error;
        if (!result.data) throw new Error(i18n.t("indexerLogin.failed"));
        report = result.data.indexerLogin ? useFragment(IndexerLoginResultFragmentDoc,result.data.indexerLogin) : undefined;
      } else if (operation === "cancel" || (operation === "begin" && old?.state === "WAITING")) {
        if (old) {
          const result = await cancelMutation({ id: old.id, input }, options);
          if (result.error) throw result.error;
          if (!result.data?.cancelIndexerLogin) throw new Error(i18n.t("indexerLogin.failed"));
          retained = undefined;
        }
      }
      // A changed scope or unmount must never advance a cancelled sequence.
      if (controller.signal.aborted || active.current !== scope) return;
      if (operation === "begin") {
        const result = await beginMutation({ input }, options);
        if (result.error) throw result.error;
        if (!result.data?.beginIndexerLogin) throw new Error(i18n.t("indexerLogin.failed"));
        report = useFragment(IndexerLoginResultFragmentDoc,result.data.beginIndexerLogin);
      } else if (operation === "answer") {
        if (!old) return;
        const result = await answerMutation({ id: old.id, input, answer: answer ?? "" }, options);
        if (result.error) throw result.error;
        if (!result.data?.answerIndexerChallenge) throw new Error(i18n.t("indexerLogin.failed"));
        report = useFragment(IndexerLoginResultFragmentDoc,result.data.answerIndexerChallenge);
      }
      if (controller.signal.aborted || active.current !== scope) return;
      if (report && (report.instanceId !== input.instanceId || report.configRevision !== input.expectedRevision || (report.state === "WAITING" && !report.challenge) || (report.challenge && report.challenge.id !== report.id))) {
        throw new Error(i18n.t("indexerLogin.failed"));
      }
      setState({ scope, busy: false, report, cancelled: operation === "cancel" });
    } catch (error) {
      if (!controller.signal.aborted && active.current === scope) {
        setState({ scope, busy: false, error, report: retained });
      }
    } finally {
      if (request.current === controller) { request.current = null; requestKind.current = null; }
    }
  };

  useEffect(() => {
    void perform("restore");
    return () => { request.current?.abort(); request.current = null; requestKind.current = null; };
    // Only a scope change restores state; mutation/client identity is not a trigger.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [scope]);

  const visible = state.scope === scope ? state : undefined;
  return {
    report: visible?.report, error: visible?.error, cancelled: visible?.cancelled,
    busy: !visible || visible.busy, cancelling: visible?.busy && visible.operation === "cancel",
    refresh: () => perform("restore"), begin: () => perform("begin"),
    answer: (value: string) => perform("answer", value),
    cancel: () => {
      if (requestKind.current === "cancel") return;
      request.current?.abort(); request.current = null; requestKind.current = null;
      return perform("cancel");
    },
  };
}
