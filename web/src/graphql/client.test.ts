// @vitest-environment jsdom

import { describe, expect, it } from "vitest";
import { getGraphQLWebSocketUrl } from "./client";

describe("getGraphQLWebSocketUrl", () => {
  it("uses ws for an HTTP page", () => {
    expect(getGraphQLWebSocketUrl({ protocol: "http:", host: "localhost:5173" })).toBe(
      "ws://localhost:5173/graphql"
    );
  });

  it("uses wss for an HTTPS page", () => {
    expect(getGraphQLWebSocketUrl({ protocol: "https:", host: "moji.example" })).toBe(
      "wss://moji.example/graphql"
    );
  });
});
