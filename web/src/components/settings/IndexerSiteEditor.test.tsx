// @vitest-environment jsdom
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createMemoryRouter, RouterProvider } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "../../i18n/i18n";
import { IndexerSiteEditor } from "./IndexerSiteEditor";
import type { IndexerSite } from "../../hooks/useIndexerManagement";
import { IndexerSourceKind } from "../../graphql/generated/graphql";

const site: IndexerSite = {health:{search:null,fetch:null}, id: "source", name: "Example", definitionId: "example", kind: IndexerSourceKind.Builtin, enabled: true, baseUrl: "", siteUrl: "https://example.invalid/", language: "en-US", privacy: "private", networkProfile: "direct", trustedOrigins: [], modes: ["search"], ruleDigest: "digest", errorCode: "", settings: [{ name: "password", label: "Password", type: "password", secret: true, configured: true, value: null, defaultValue: null, options: [] }] };
beforeEach(async () => { await i18n.changeLanguage("en"); });
afterEach(cleanup);
function mount(save = vi.fn().mockResolvedValue(false)) {
  const close = vi.fn(); const router = createMemoryRouter([{ path: "/", element: <IndexerSiteEditor site={site} definition={{ id: "example", name: "Example", links: [], settings: site.settings }} revision="config-1" profiles={["direct"]} busy={false} error={null} onSave={save} onClose={close} /> }, { path: "/away", element: <p>Away</p> }], { initialEntries: ["/"] });
  render(<RouterProvider router={router} />); return { save, close, router };
}
describe("built-in site editor", () => {
  it("submits only explicit edits and retains them after a failed save", async () => {
    const user = userEvent.setup(); const { save } = mount();
    const input = screen.getByLabelText("Name"); await user.clear(input); await user.type(input, "Renamed");
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(save).toHaveBeenCalledWith("source", "config-1", { name: "Renamed" }));
    expect((screen.getByLabelText("Name") as HTMLInputElement).value).toBe("Renamed");
    expect((screen.getByLabelText("Password") as HTMLInputElement).value).toBe("");
  });
  it("clears a saved secret only through an explicit clear edit", async () => {
    const user = userEvent.setup(); const { save } = mount();
    await user.click(screen.getByRole("button", { name: "Clear saved value" }));
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(save).toHaveBeenCalledWith("source", "config-1", { settings: [{ name: "password", clear: true }] }));
  });
  it("retains unsaved input when navigation is cancelled", async () => {
    const user = userEvent.setup(); const { router } = mount();
    await user.type(screen.getByLabelText("Name"), " changed");
    await act(async () => { void router.navigate("/away"); });
    expect(await screen.findByRole("dialog", { name: "Discard unsaved changes?" })).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Keep editing" }));
    expect(router.state.location.pathname).toBe("/");
    expect((screen.getByLabelText("Name") as HTMLInputElement).value).toBe("Example changed");
  });
});

it("sends explicit origin replacement and empty-list revocation without changing credentials", async () => {
  const user = userEvent.setup(); const { save } = mount();
  await user.click(screen.getByText("Advanced settings"));
  const field = screen.getByLabelText("Additional trusted origins");
  await user.type(field, "https://download.example\nhttps://auth.example:8443/");
  await user.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() => expect(save).toHaveBeenLastCalledWith("source", "config-1", { trustedOrigins: ["https://download.example", "https://auth.example:8443/"] }));
  // A non-empty initial list must be explicitly replaced by [] when cleared.
  cleanup();
  const initial = { ...site, trustedOrigins: ["https://download.example"] };
  const router = createMemoryRouter([{ path: "/", element: <IndexerSiteEditor site={initial} definition={{id:"example",name:"Example",links:[],settings:site.settings}} revision="config-2" profiles={["direct"]} busy={false} error={null} onSave={save} onClose={()=>{}} /> }]);
  render(<RouterProvider router={router} />);
  await user.click(screen.getByText("Advanced settings"));
  await user.clear(screen.getByLabelText("Additional trusted origins"));
  await user.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() => expect(save).toHaveBeenLastCalledWith("source", "config-2", { trustedOrigins: [] }));
});

it("invalid origins prevent both save and draft testing and preserve unsaved input", async () => {
  const user = userEvent.setup(); const { save } = mount();
  await user.click(screen.getByText("Advanced settings"));
  const field = screen.getByLabelText("Additional trusted origins");
  for (const invalid of ["https://download.example/path", "https://user:secret@download.example", "https://download.example?", "https://download.example#", "https://download.example/.", "https://download.example\\", "https://download.example?token=secret", "https://*.example", "*.example", Array.from({length:17},(_,i)=>`https://site${i}.example`).join("\n")]) {
    await user.clear(field); await user.paste(invalid);
    expect(screen.getByRole("button", { name: "Save" }).hasAttribute("disabled")).toBe(true);
    expect(screen.getByRole("button", { name: "Test site" }).hasAttribute("disabled")).toBe(true);
    expect(screen.getByRole("alert")).toBeTruthy();
  }
  expect(save).not.toHaveBeenCalled();
  await user.click(screen.getByRole("button", { name: "Close" }));
  expect(await screen.findByRole("dialog", { name: "Discard unsaved changes?" })).toBeTruthy();
});
