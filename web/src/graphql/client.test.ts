// @vitest-environment jsdom

import { gql } from "urql";
import { describe, expect, it, vi } from "vitest";
import { client, getGraphQLWebSocketUrl, settingsClient } from "./client";

describe("settingsClient", () => {
  it("reads changed settings on every query without using or updating the shared cache", async () => {
    const query = gql`query SettingsFreshness { settings { stash { url } } }`;
    let stashUrl = "http://old-stash";
    const fetchSettings = vi.fn<typeof fetch>(async () => new Response(JSON.stringify({
      data: { settings: { __typename: "Settings", stash: { __typename: "StashSettings", url: stashUrl } } }
    }), { headers: { "Content-Type": "application/json" } }));
    const context = { fetch: fetchSettings };
    type Result = { settings: { stash: { url: string } } };

    await client.query<Result>(query, {}, context).toPromise();
    stashUrl = "http://new-stash";
    const first = await settingsClient.query<Result>(query, {}, context).toPromise();
    expect(first.error).toBeUndefined();
    expect(first.data?.settings.stash.url).toBe("http://new-stash");

    stashUrl = "http://newest-stash";
    const second = await settingsClient.query<Result>(query, {}, context).toPromise();
    expect(second.error).toBeUndefined();
    expect(second.data?.settings.stash.url).toBe("http://newest-stash");
    expect(fetchSettings).toHaveBeenCalledTimes(3);

    const shared = await client.query<Result>(query, {}, { requestPolicy: "cache-only" }).toPromise();
    expect(shared.data?.settings.stash.url).toBe("http://old-stash");
    expect(fetchSettings).toHaveBeenCalledTimes(3);
  });
});

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
