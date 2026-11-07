import { useCallback } from "react";
import { useClient, useQuery } from "urql";
import { useFragment } from "../graphql/generated/fragment-masking";
import { IndexerEventKind, IndexerNetworkSettingsDocument, IndexerNetworkSettingsFieldsFragmentDoc, UpdateIndexerNetworkDocument, type IndexerNetworkPatchInput } from "../graphql/generated/graphql";
import { useIndexerEvents } from "./useIndexerEvents";
import i18n from "../i18n/i18n";

export function useIndexerNetwork() {
  const client = useClient();
  const [query, requery] = useQuery({ query: IndexerNetworkSettingsDocument, requestPolicy: "network-only" });
  const read = useCallback(() => requery({ requestPolicy: "network-only" }), [requery]);
  const events = useIndexerEvents(read, [IndexerEventKind.Configuration]);
  return {
    snapshot: query.data ? useFragment(IndexerNetworkSettingsFieldsFragmentDoc, query.data.indexerNetworkSettings) : null,
    error: query.error ?? events.error,
    refresh: () => { read(); if (events.error) events.retry(); },
    load: async () => {
      const result=await client.query(IndexerNetworkSettingsDocument,{}, {requestPolicy:"network-only"}).toPromise();
      if(result.error)throw result.error;
      if(!result.data?.indexerNetworkSettings)throw new Error(i18n.t("settings.autosave.noAcknowledgement"));
      return useFragment(IndexerNetworkSettingsFieldsFragmentDoc,result.data.indexerNetworkSettings);
    },
    save: async (expectedRevision: string, input: IndexerNetworkPatchInput) => {
      const result = await client.mutation(UpdateIndexerNetworkDocument, { expectedRevision, input }).toPromise();
      if (result.error) throw result.error;
      if (!result.data?.updateIndexerNetwork) throw new Error(i18n.t("settings.autosave.noAcknowledgement"));
      return useFragment(IndexerNetworkSettingsFieldsFragmentDoc, result.data.updateIndexerNetwork);
    }
  };
}
