import i18n from "../i18n/i18n";

export function describePerformerSceneQueueResult(reasonCode: string) {
  const key = `performerSceneQueue.reasons.${reasonCode}`;
  return i18n.exists(key) ? i18n.t(key) : i18n.t("performerSceneQueue.reasons.UNKNOWN");
}
