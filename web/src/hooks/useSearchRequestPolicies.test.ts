// @vitest-environment jsdom

import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  useQuery: vi.fn(() => [{ data: undefined, fetching: false, error: undefined }]),
  useMutation: vi.fn(() => [{ fetching: false }, vi.fn()])
}));

vi.mock("urql", () => ({
  useQuery: mocks.useQuery,
  useMutation: mocks.useMutation
}));

import { useDiscoverScenes } from "./useDiscoverScenes";
import { useJackettSearch } from "./useJackettSearch";

describe("search request policies", () => {
  beforeEach(() => {
    mocks.useQuery.mockClear();
    mocks.useMutation.mockClear();
  });

  it("always fetches fresh Jackett search results", () => {
    renderHook(() => useJackettSearch("query", { enabled: true }));
    expect(mocks.useQuery).toHaveBeenCalledWith(expect.objectContaining({ requestPolicy: "network-only" }));
  });

  it("shows cached discovery results while refreshing them", () => {
    renderHook(() => useDiscoverScenes("query", { enabled: true }));
    expect(mocks.useQuery).toHaveBeenCalledWith(expect.objectContaining({ requestPolicy: "cache-and-network" }));
  });
});
