// @vitest-environment jsdom
import { memo, useState } from "react";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, it } from "vitest";
import { useAppTimezone } from "./useAppTimezone";
import { setAppTimezone } from "../utils/appTimezone";
import { formatPublishDate } from "../utils/formatters";

afterEach(() => { cleanup(); setAppTimezone("UTC"); });
it("updates mounted memoized time consumers without losing local edits", () => {
  const View = memo(function View() {
    useAppTimezone();
    const [value, setValue] = useState("");
    return <><input aria-label="draft" value={value} onChange={event => setValue(event.target.value)} /><time>{formatPublishDate("2026-11-19T23:30:00Z")}</time></>;
  });
  render(<View />);
  fireEvent.change(screen.getByLabelText("draft"), { target: { value: "pending edit" } });
  expect(screen.getByText("2026-11-19")).toBeTruthy();
  act(() => setAppTimezone("Asia/Tokyo"));
  expect(screen.getByText("2026-11-20")).toBeTruthy();
  expect((screen.getByLabelText("draft") as HTMLInputElement).value).toBe("pending edit");
});
