// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { useMemo } from "react";
import { afterEach, describe, expect, it } from "vitest";
import { SettingsDraftProvider, useSettingsDraft } from "./SettingsDraftStore";

afterEach(cleanup);

function Editor({ visible, initial = "server" }: { visible: boolean; initial?: string }) {
  if (!visible) return null;
  return <DraftField initial={initial} />;
}

function DraftField({ initial }: { initial: string }) {
  const value = useMemo(() => ({ text: initial }), [initial]);
  const [draft, setDraft, dirty] = useSettingsDraft("connections", value);
  return <><input aria-label="draft" value={draft.text} onChange={(event) => setDraft({ text: event.target.value })} /><output>{dirty ? "dirty" : "clean"}</output></>;
}

describe("SettingsDraftProvider", () => {
  it("preserves dirty values while switching tabs and ignores background snapshots", () => {
    const view = render(<SettingsDraftProvider><Editor visible /></SettingsDraftProvider>);
    fireEvent.change(screen.getByLabelText("draft"), { target: { value: "unsaved" } });
    expect(screen.getByText("dirty")).toBeTruthy();
    view.rerender(<SettingsDraftProvider><Editor visible={false} /></SettingsDraftProvider>);
    view.rerender(<SettingsDraftProvider><Editor visible initial="new-server-value" /></SettingsDraftProvider>);
    expect((screen.getByLabelText("draft") as HTMLInputElement).value).toBe("unsaved");
  });

  it("clears drafts after leaving the settings provider", () => {
    const view = render(<SettingsDraftProvider><Editor visible /></SettingsDraftProvider>);
    fireEvent.change(screen.getByLabelText("draft"), { target: { value: "unsaved" } });
    view.unmount();
    render(<SettingsDraftProvider><Editor visible /></SettingsDraftProvider>);
    expect((screen.getByLabelText("draft") as HTMLInputElement).value).toBe("server");
  });
});
