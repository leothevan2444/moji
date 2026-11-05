// @vitest-environment jsdom
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, expect, it } from "vitest";
import i18n from "../../i18n/i18n";
import { IndexerConnectionButton } from "./IndexerConnectionButton";

beforeEach(async () => { await i18n.changeLanguage("en"); });
afterEach(cleanup);
it.each(["en", "zh-CN"])("unknown connection errors remain retryable in %s", async locale => {
  await i18n.changeLanguage(locale);
  render(<IndexerConnectionButton result={{ state: "failed", errorCode: "SYNTHETIC_UNKNOWN_ERROR" }} disabled={false} onClick={() => {}} />);
  const button = screen.getByRole("button", { name: i18n.t("indexers.connectionFailed") });
  expect(button.hasAttribute("disabled")).toBe(false);
  expect(button.title).not.toContain("SYNTHETIC_UNKNOWN_ERROR");
});

it.each(["passed", "failed"] as const)("the batch button hides its %s result", state => {
  render(<IndexerConnectionButton all result={{ state }} disabled={false} onClick={() => {}} />);
  const button = screen.getByRole("button", { name: i18n.t("indexers.testAll") });
  expect(button.className).not.toContain("is-passed");
  expect(button.className).not.toContain("is-failed");
});
