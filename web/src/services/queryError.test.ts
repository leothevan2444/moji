import { beforeEach, describe, expect, it } from "vitest";
import i18n from "../i18n/i18n";
import { describeQueryError } from "./queryError";

describe("describeQueryError", () => {
  beforeEach(async () => { await i18n.changeLanguage("en"); });

  it("localizes stable GraphQL error codes without exposing the server message", () => {
    const result = describeQueryError({
      message: "[GraphQL] secret database path",
      graphQLErrors: [{ message: "request failed", extensions: { code: "DUPLICATE_CODE_TASK", params: {}, correlationId: "req-123" } }]
    });
    expect(result).toContain("Another Moji task already uses this code");
    expect(result).toContain("req-123");
    expect(result).not.toContain("database path");
  });

  it("keeps the legacy message compatibility mapping", () => {
    expect(describeQueryError({ graphQLErrors: [{ message: "duplicate torrent task" }] }))
      .toContain("Another Moji task already uses this torrent");
  });

  it("localizes performer scene batch limits", () => {
    const result = describeQueryError({
      graphQLErrors: [{ message: "request failed", extensions: { code: "PERFORMER_SCENE_BATCH_TOO_LARGE", params: {} } }]
    });
    expect(result).toBe("A batch can contain at most 100 performer scenes.");
  });
});
