// @vitest-environment jsdom
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { useNsfwMask } from "./useNsfwMask";

function Harness() {
  const { masked, toggle } = useNsfwMask();
  return <button aria-pressed={masked} onClick={toggle}>Mask images</button>;
}

beforeEach(() => localStorage.clear());
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  delete document.documentElement.dataset.nsfwMask;
});

it("defaults off, toggles immediately and restores the browser preference after remount", () => {
  const first = render(<Harness />);
  expect(document.documentElement.dataset.nsfwMask).toBe("false");
  fireEvent.click(screen.getByRole("button"));
  expect(localStorage.getItem("moji:nsfw-mask")).toBe("true");
  expect(document.documentElement.dataset.nsfwMask).toBe("true");
  first.unmount();
  render(<Harness />);
  expect(screen.getByRole("button").getAttribute("aria-pressed")).toBe("true");
  fireEvent.click(screen.getByRole("button"));
  expect(localStorage.getItem("moji:nsfw-mask")).toBe("false");
  expect(document.documentElement.dataset.nsfwMask).toBe("false");
});

it("synchronizes changes and storage clearing from another tab", () => {
  render(<Harness />);
  act(() => {
    localStorage.setItem("moji:nsfw-mask", "true");
    window.dispatchEvent(new StorageEvent("storage", { key: "moji:nsfw-mask", storageArea: localStorage }));
  });
  expect(document.documentElement.dataset.nsfwMask).toBe("true");
  act(() => {
    localStorage.clear();
    window.dispatchEvent(new StorageEvent("storage", { key: null, storageArea: localStorage }));
  });
  expect(document.documentElement.dataset.nsfwMask).toBe("false");
});

it("still toggles when browser storage is unavailable", () => {
  vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => { throw new Error("Storage blocked"); });
  vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new Error("Storage blocked"); });
  render(<Harness />);
  fireEvent.click(screen.getByRole("button"));
  expect(document.documentElement.dataset.nsfwMask).toBe("true");
});
