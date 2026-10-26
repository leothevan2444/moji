import { beforeAll, describe, expect, it } from "vitest";
import i18n from "../i18n/i18n";
import { resources } from "../i18n/resources";
import { TaskStageStatus } from "../graphql/generated/graphql";
import official from "./__fixtures__/qbittorrent-statuses.json";
import { qbittorrentStateLabel } from "./qbittorrentState";
import { taskFailureSummary, taskPresentation, type DashboardTask } from "./taskUtils";

const states = Object.assign({}, ...official.releases.map(release => release.states)) as Record<string, string[]>;

describe("official qBittorrent status copy (verbatim)", () => {
  beforeAll(async () => { await i18n.loadLanguages(["en", "zh-CN"]); });
  // This independently extracted upstream fixture is intentionally separate
  // from the editable locale resources. See its pinned source URLs and hashes.
  for (const [locale, column] of [["en", 0], ["zh-CN", 1]] as const) {
    const translated: Record<string, string> = resources[locale].translation.downloader.states;
    it.each(Object.entries(states))(`${locale}: %s matches upstream, including prefixes and spaces`, (state, labels) => {
      expect(translated[state]).toBe(labels[column]);
      expect(qbittorrentStateLabel(state, key => i18n.t(key, { lng: locale }))).toBe(labels[column]);
      expect(qbittorrentStateLabel(` ${state.toUpperCase()} `, key => i18n.t(key, { lng: locale }))).toBe(labels[column]);
    });

    it(`${locale}: covers every project status and uses the official fallback`, () => {
      expect(Object.keys(translated).sort()).toEqual([...Object.keys(states), "missing", "removed"].sort());
      for (const state of [null, undefined, "", "futureQbState", "toString", "__proto__"]) {
        expect(qbittorrentStateLabel(state, key => i18n.t(key, { lng: locale }))).toBe(states.unknown[column]);
      }
      for (const state of ["missing", "removed"]) {
        expect(qbittorrentStateLabel(state, key => i18n.t(key, { lng: locale }))).toBe(translated[state]);
      }
    });

    it(`${locale}: uses official copy in task summaries, lifecycle details and torrent errors`, async () => {
      await i18n.changeLanguage(locale);
      for (const [state, labels] of Object.entries(states)) {
        const task = { id: "download", stage: "DOWNLOADING", stageStatus: "RUNNING", progress: 0.42, qbittorrentState: state } as DashboardTask;
        expect(taskFailureSummary(task).detail).toBe(labels[column]);
        expect(taskPresentation(task).lifecycle.find(step => step.key === "downloading")?.detail).toBe(`${labels[column]} · 42%`);
        if (["error", "missingFiles"].includes(state)) {
          expect(taskFailureSummary({ ...task, stageStatus: TaskStageStatus.Blocked, stageErrorCode: "TORRENT_ERROR" }).title).toBe(labels[column]);
        }
      }
    });
  }
});
