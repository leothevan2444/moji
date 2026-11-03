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

describe("search request policies", () => {
  beforeEach(() => {
    mocks.useQuery.mockClear();
    mocks.useMutation.mockClear();
  });

  it("shows cached discovery results while refreshing them", () => {
    renderHook(() => useDiscoverScenes("query", { enabled: true }));
    expect(mocks.useQuery).toHaveBeenCalledWith(expect.objectContaining({ requestPolicy: "cache-and-network" }));
  });
});
