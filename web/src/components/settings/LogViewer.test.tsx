// @vitest-environment jsdom
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { LogLevel } from "../../graphql/generated/graphql";
import { LogViewer } from "./LogViewer";

vi.mock("react-i18next", async (importOriginal) => ({ ...await importOriginal<typeof import("react-i18next")>(), useTranslation: () => ({ t: (key: string) => key }) }));
const entry = (sequence: number) => ({ sequence, level: LogLevel.Info, time: "2026-10-31T12:00:00Z", message: `line ${sequence}` });

describe("LogViewer", () => {
  it("follows the bottom, freezes history through eviction and resumes with the latest bounded list", () => {
    const copy = vi.fn();
    const view = render(<LogViewer entries={[entry(2), entry(1)]} onCopy={copy} />);
    const log = screen.getByRole("log");
    Object.defineProperties(log, { scrollHeight: { configurable: true, value: 1000 }, clientHeight: { value: 200 } });
    view.rerender(<LogViewer entries={[entry(3), entry(2), entry(1)]} onCopy={copy} />);
    expect(log.scrollTop).toBe(1000);
    expect([...log.querySelectorAll(".log-line__message")].map(node => node.textContent)).toEqual(["line 1", "line 2", "line 3"]);
    log.scrollTop = 100;
    fireEvent.scroll(log);
    view.rerender(<LogViewer entries={[entry(300), entry(299)]} onCopy={copy} />);
    expect(log.scrollTop).toBe(100);
    expect(log.textContent).toContain("line 1");
    expect(log.textContent).not.toContain("line 300");
    fireEvent.click(screen.getByText("logsUi.copy"));
    expect(copy).toHaveBeenLastCalledWith([entry(1), entry(2), entry(3)]);
    fireEvent.click(screen.getByText("logsUi.resumeNew"));
    expect(log.scrollTop).toBe(1000);
    expect(log.textContent).toContain("line 300");
    expect(log.getAttribute("aria-live")).toBe("polite");
  });
});
