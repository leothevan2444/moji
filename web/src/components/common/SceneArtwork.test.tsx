// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { SceneArtwork } from "./SceneArtwork";

afterEach(cleanup);
describe("scene artwork", () => {
  it("keeps a poster fallback after an image fails and retries a new source", () => {
    const view = render(<SceneArtwork src="/first.jpg" alt="Film cover" />);
    fireEvent.error(screen.getByAltText("Film cover"));
    expect(screen.queryByAltText("Film cover")).toBeNull();
    expect(view.container.querySelector(".performer-scene-card__image-placeholder")).not.toBeNull();
    view.rerender(<SceneArtwork src="/second.jpg" alt="Film cover" />);
    expect(screen.getByAltText("Film cover").getAttribute("src")).toBe("/second.jpg");
  });
});
