import type { ResourceConditionInput } from "../../graphql/generated/graphql";

export const resourceRuleID = () => "r-" + Array.from(crypto.getRandomValues(new Uint8Array(16)), value => value.toString(16).padStart(2, "0")).join("");
export const emptyResourceCondition = (): ResourceConditionInput => ({ scope: "TITLE", value: "", minimum: "", maximum: "" });

// Keep unfinished rows in the editor draft, but never send blank conditions.
export const hasResourceConditionValue = (condition: ResourceConditionInput) =>
  condition.scope === "SIZE" || condition.scope === "SEEDERS"
    ? Boolean(condition.minimum.trim() || condition.maximum.trim())
    : Boolean(condition.value.trim());
