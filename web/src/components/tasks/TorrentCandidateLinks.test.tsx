// @vitest-environment jsdom
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import i18n from "../../i18n/i18n";
import { candidateURL, TorrentCandidateLinks } from "./TorrentCandidateLinks";

afterEach(cleanup);
describe("candidate links", () => {
  it.each(["javascript:alert(1)", "data:text/html,test", "file:///tmp/torrent", "https://user:secret@example.com/file", "/relative", ""])("rejects unsupported or credential-bearing URL %s", url => {
    expect(candidateURL(url)).toBeUndefined();
    expect(candidateURL(url, true)).toBeUndefined();
  });
  it("uses magnet-valued download links without mislabelling them as torrent downloads", async () => {
    await i18n.changeLanguage("en");
    const magnet = "magnet:?xt=urn:btih:example";
    render(<TorrentCandidateLinks candidate={{ link: magnet, details: "javascript:alert(1)" }} />);
    const links = screen.getAllByRole("link");
    expect(links).toHaveLength(1);
    expect(links[0].getAttribute("href")).toBe(magnet);
    expect(links[0].getAttribute("target")).toBeNull();
  });
});
