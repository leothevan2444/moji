import { useEffect, useRef, useState } from "react";
import i18n from "../i18n/i18n";
import { useMutation } from "urql";
import { PreviewTorrentSelectionDocument, type PreviewTorrentSelectionInput, type PreviewTorrentSelectionMutation } from "../graphql/generated/graphql";

export function useTorrentSelectionPreview() {
  const [, mutation] = useMutation(PreviewTorrentSelectionDocument);
  const [value, setValue] = useState<PreviewTorrentSelectionMutation["previewTorrentSelection"] | null>(null);
  const [error, setError] = useState<unknown>(null); const [busy, setBusy] = useState(false);
  const generation = useRef(0); const pending = useRef(false);
  useEffect(() => () => { generation.current++; }, []);
  const clear = () => { generation.current++; setValue(null); setError(null); };
  const preview = async (input: PreviewTorrentSelectionInput) => {
    if (pending.current) return false;
    const ticket = ++generation.current; pending.current = true; setBusy(true); setError(null);
    try {
      const result = await mutation({ input }); if (ticket !== generation.current) return false;
      if (result.error) throw result.error;
      if (!result.data?.previewTorrentSelection) throw new Error(i18n.t("indexerErrors.INDEXER_SEARCH_INCOMPLETE"));
      setValue(result.data.previewTorrentSelection); return true;
    } catch (failure) { if (ticket === generation.current) setError(failure); return false; }
    finally { pending.current = false; setBusy(false); }
  };
  return { value, error, busy, preview, clear };
}
