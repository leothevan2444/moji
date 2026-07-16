import { beforeEach, describe, expect, it } from "vitest";
import i18n from "../i18n/i18n";
import { describePerformerSceneQueueResult } from "./performerSceneQueueResult";

describe("describePerformerSceneQueueResult", () => {
  beforeEach(async () => i18n.changeLanguage("en"));

  it("localizes every known reason code without using backend messages", () => {
    const reasons = [
      "QUEUED",
      "ALREADY_IN_LIBRARY",
      "DUPLICATE_CODE_TASK",
      "DUPLICATE_TORRENT_TASK",
      "MISSING_CODE",
      "NO_STASHBOX_SOURCE",
      "SCENE_NOT_FOUND",
      "QUEUE_FAILED"
    ];
    for (const reason of reasons) {
      expect(describePerformerSceneQueueResult(reason)).not.toContain(reason);
    }
    expect(describePerformerSceneQueueResult("QUEUE_FAILED")).toBe("The download task could not be created. Try again later.");
  });

  it("supports Chinese and falls back safely for unknown reasons", async () => {
    await i18n.changeLanguage("zh-CN");
    expect(describePerformerSceneQueueResult("ALREADY_IN_LIBRARY")).toBe(i18n.t("performerSceneQueue.reasons.ALREADY_IN_LIBRARY"));
    expect(describePerformerSceneQueueResult("UNRECOGNIZED")).toBe(i18n.t("performerSceneQueue.reasons.UNKNOWN"));
  });
});
