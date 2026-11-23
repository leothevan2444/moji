import { useEffect, useRef, useState } from "react";
import i18n from "../i18n/i18n";
import { useMutation } from "urql";
import { PreviewTorrentSelectionDocument, type PreviewTorrentSelectionInput, type PreviewTorrentSelectionMutation } from "../graphql/generated/graphql";

type Preview = PreviewTorrentSelectionMutation["previewTorrentSelection"];
export function useTorrentSelectionPreview() {
 const [,mutation]=useMutation(PreviewTorrentSelectionDocument);
 const [value,setValue]=useState<Preview|null>(null);const [error,setError]=useState<unknown>(null);const [busy,setBusy]=useState(false);
 const generation=useRef(0);const pending=useRef(false);
 useEffect(()=>()=>{generation.current++;},[]);
 const clear=()=>{generation.current++;setValue(null);setError(null);setBusy(false);};
 const preview=async(input:PreviewTorrentSelectionInput)=>{if(pending.current)return null;const ticket=++generation.current;pending.current=true;setBusy(true);setError(null);try{const result=await mutation({input});if(ticket!==generation.current)return null;if(result.error)throw result.error;const preview=result.data?.previewTorrentSelection;if(!preview)throw new Error(i18n.t("resourceSelection.previewFailed"));setValue(preview);return preview;}catch(e){if(ticket===generation.current)setError(e);return null;}finally{pending.current=false;setBusy(false);}};
 return {value,error,busy,preview,clear};
}
