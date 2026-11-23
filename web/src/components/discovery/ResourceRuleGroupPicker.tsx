import { useQuery } from "urql";
import { useTranslation } from "react-i18next";
import { ResourceRuleGroupsDocument } from "../../graphql/generated/graphql";
import { describeQueryError } from "../../services/queryError";
export function ResourceRuleGroupPicker({value,onChange,disabled}:{value?:string;onChange:(value:string)=>void;disabled?:boolean}) {
 const {t}=useTranslation();const [query]=useQuery({query:ResourceRuleGroupsDocument});
 const policy=query.data?.settings.policies.torrentSelection;
 if(query.error)return <p className="inline-error" role="alert">{describeQueryError(query.error)}</p>;
 if(!policy?.groups.length)return null;
 return <label>{t("resourceSelection.groupForPreview")}<select value={value??policy.activeGroup} disabled={disabled||query.fetching} onChange={e=>onChange(e.target.value)}>{policy.groups.map(g=><option value={g.id} key={g.id}>{g.name}</option>)}</select></label>;
}
