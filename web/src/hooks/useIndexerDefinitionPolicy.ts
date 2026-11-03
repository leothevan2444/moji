import {useCallback} from "react";
import {useClient,useQuery} from "urql";
import {useFragment} from "../graphql/generated/fragment-masking";
import {IndexerDefinitionPolicyDocument,IndexerDefinitionPolicyFieldsFragmentDoc,IndexerEventKind,UpdateIndexerDefinitionPolicyDocument,type IndexerDefinitionPolicyPatchInput} from "../graphql/generated/graphql";
import {useIndexerEvents} from "./useIndexerEvents";
import i18n from "../i18n/i18n";
export function useIndexerDefinitionPolicy(){
 const client=useClient();const [query,requery]=useQuery({query:IndexerDefinitionPolicyDocument,requestPolicy:"network-only"});
 const read=useCallback(()=>requery({requestPolicy:"network-only"}),[requery]);const events=useIndexerEvents(read,[IndexerEventKind.Configuration]);
 const refresh=()=>{read();if(events.error)events.retry()};
 const reload=async()=>{const result=await client.query(IndexerDefinitionPolicyDocument,{},{requestPolicy:"network-only"}).toPromise();if(result.error)throw result.error;if(!result.data?.indexerDefinitionPolicy)throw new Error(i18n.t("settings.autosave.noAcknowledgement"));return useFragment(IndexerDefinitionPolicyFieldsFragmentDoc,result.data.indexerDefinitionPolicy)};
 const update=async(input:IndexerDefinitionPolicyPatchInput,expectedRevision:string)=>{const result=await client.mutation(UpdateIndexerDefinitionPolicyDocument,{input,expectedRevision}).toPromise();if(result.error)throw result.error;if(!result.data?.updateIndexerDefinitionPolicy)throw new Error(i18n.t("settings.autosave.noAcknowledgement"));return useFragment(IndexerDefinitionPolicyFieldsFragmentDoc,result.data.updateIndexerDefinitionPolicy).revision};
 return {snapshot:query.data?.indexerDefinitionPolicy?useFragment(IndexerDefinitionPolicyFieldsFragmentDoc,query.data.indexerDefinitionPolicy):null,loading:query.fetching,error:query.error??events.error,refresh,reload,update};
}
