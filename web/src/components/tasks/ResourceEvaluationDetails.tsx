import "../../styles/resource-selection.scss";
import { useTranslation } from "react-i18next";
import type { ResourceEvaluation } from "../../graphql/generated/graphql";
type Evaluation=Pick<ResourceEvaluation,"evidenceOmitted"|"groupName"|"tierName"|"title"|"status"|"preferred"|"reasons"|"unknownRules"|"evidence">;
export function ResourceEvaluationDetails({evaluation}:{evaluation:Evaluation}){
 const {t}=useTranslation();const reason=(v:string)=>t(`resourceSelection.reasons.${v}`,{defaultValue:v});
 return <details className="resource-evaluation"><summary>{t(`resourceSelection.statuses.${evaluation.status}`)} · {evaluation.reasons.slice(0,2).map(reason).join(" · ")}</summary>
 {evaluation.groupName?<p>{evaluation.reasons.includes("FALLBACK_ALLOWED")?t("resourceSelection.fallbackGroup",{group:evaluation.groupName}):t("resourceSelection.matchedTier",{group:evaluation.groupName,tier:evaluation.tierName||"—"})}</p>:null}
 {evaluation.unknownRules.length?<p>{t("resourceSelection.unknown",{rules:evaluation.unknownRules.join("、")})}</p>:null}
 <ul>{evaluation.reasons.map((v,i)=><li key={i}>{reason(v)}</li>)}</ul>
 {evaluation.evidence.length?<ul>{evaluation.evidence.map((v,i)=><li key={i}>{t("resourceSelection.conditionResult",{condition:v.attribute,result:t(`resourceSelection.truth.${v.value}`,{defaultValue:v.value})})} · {t(`resourceSelection.scopes.${v.source}`,{defaultValue:v.source})}: {v.text}</li>)}</ul>:null}
 {evaluation.evidenceOmitted>0?<p>{t("resourceSelection.evidenceOmitted",{count:evaluation.evidenceOmitted})}</p>:null}
 </details>;
}
