// @vitest-environment jsdom
import {act,cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import i18n from "../../i18n/i18n";
import {PreparedTorrentDownloadActions} from "./PreparedTorrentDownloadActions";
const mock=vi.hoisted(()=>({execute:vi.fn()}));
vi.mock("urql",()=>({useMutation:()=>[{},mock.execute]}));
const url="/api/indexers/downloads/"+"a".repeat(64);
const receipt=(kind="TORRENT",downloadUrl=url)=>({data:{prepareTorrentDownload:{kind,downloadUrl,expiresAt:new Date(Date.now()+5000).toISOString()}}});
beforeEach(async()=>{mock.execute.mockReset();await i18n.changeLanguage("en")});
afterEach(()=>{cleanup();vi.useRealTimers();vi.unstubAllGlobals()});
it("acquires only after an explicit click, sending IDs and displaying a same-origin attachment",async()=>{
 mock.execute.mockResolvedValue(receipt());render(<PreparedTorrentDownloadActions searchId="run" candidateId="candidate"/>);
 expect(mock.execute).not.toHaveBeenCalled();expect(screen.queryByRole("link")).toBeNull();fireEvent.click(screen.getByRole("button"));
 const link=await screen.findByRole("link",{name:"Download torrent"});expect(link.getAttribute("href")).toBe(url);expect(link.getAttribute("target")).toBe("_blank");expect(link.getAttribute("rel")).toContain("noopener");
 expect(mock.execute.mock.calls[0][0]).toEqual({input:{searchId:"run",candidateId:"candidate"}});
});
it.each(["https://private.example/grab?secret=synthetic","/api/indexers/downloads/../../other",url+"?url=https://example.test"])("rejects a replacement locator %s without fetching it",async target=>{
 mock.execute.mockResolvedValue(receipt("TORRENT",target));const fetch=vi.fn();vi.stubGlobal("fetch",fetch);render(<PreparedTorrentDownloadActions searchId="run" candidateId="candidate"/>);fireEvent.click(screen.getByRole("button"));
 expect(await screen.findByRole("alert")).toBeTruthy();expect(fetch).not.toHaveBeenCalled();expect(screen.queryByRole("link")).toBeNull();expect(screen.getByRole("button").hasAttribute("disabled")).toBe(false);
});
it("reads a prepared magnet attachment before offering a native magnet link",async()=>{
 const magnet="magnet:?xt=urn:btih:synthetic&dn=ABCD-123";mock.execute.mockResolvedValue(receipt("MAGNET"));const fetch=vi.fn().mockResolvedValue(new Response(magnet,{headers:{"content-type":"text/uri-list; charset=utf-8"}}));vi.stubGlobal("fetch",fetch);
 render(<PreparedTorrentDownloadActions searchId="run" candidateId="candidate"/>);fireEvent.click(screen.getByRole("button"));const link=await screen.findByRole("link");expect(link.getAttribute("href")).toBe(magnet);expect(fetch.mock.calls[0][0]).toBe(url);expect(fetch.mock.calls[0][1]).toMatchObject({redirect:"error",cache:"no-store",credentials:"same-origin"});
});
it("rejects a failed magnet response and keeps an explicit retry",async()=>{
 mock.execute.mockResolvedValue(receipt("MAGNET"));const fetch=vi.fn().mockResolvedValue(new Response("expired",{status:410}));vi.stubGlobal("fetch",fetch);render(<PreparedTorrentDownloadActions searchId="run" candidateId="candidate"/>);fireEvent.click(screen.getByRole("button"));expect(await screen.findByRole("alert")).toBeTruthy();expect(screen.queryByRole("link")).toBeNull();expect(mock.execute).toHaveBeenCalledTimes(1);fireEvent.click(screen.getByRole("button"));await waitFor(()=>expect(mock.execute).toHaveBeenCalledTimes(2));
});
it("aborts on unmount and never applies a late receipt to another candidate",async()=>{
 let resolve!:(value:ReturnType<typeof receipt>)=>void;mock.execute.mockImplementation(()=>new Promise(done=>{resolve=done}));const view=render(<PreparedTorrentDownloadActions searchId="one" candidateId="first"/>);fireEvent.click(screen.getByRole("button"));const signal=mock.execute.mock.calls[0][1].fetchOptions.signal as AbortSignal;
 view.rerender(<PreparedTorrentDownloadActions searchId="two" candidateId="second"/>);expect(signal.aborted).toBe(true);await act(async()=>resolve(receipt()));expect(screen.queryByRole("link")).toBeNull();
 mock.execute.mockResolvedValue(receipt());fireEvent.click(screen.getByRole("button"));await screen.findByRole("link");expect(mock.execute.mock.calls[1][0]).toEqual({input:{searchId:"two",candidateId:"second"}});view.unmount();
});
it("expires the displayed handle without starting another acquisition",async()=>{
 vi.useFakeTimers();mock.execute.mockResolvedValue(receipt());render(<PreparedTorrentDownloadActions searchId="run" candidateId="candidate"/>);await act(async()=>fireEvent.click(screen.getByRole("button")));expect(screen.getByRole("link")).toBeTruthy();await act(async()=>vi.advanceTimersByTime(5001));expect(screen.queryByRole("link")).toBeNull();expect(screen.getByRole("button")).toBeTruthy();expect(mock.execute).toHaveBeenCalledTimes(1);
});
