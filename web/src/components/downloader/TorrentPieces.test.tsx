// @vitest-environment jsdom
import {cleanup,render,screen} from "@testing-library/react";
import {afterEach,beforeEach,describe,expect,it,vi} from "vitest";
import i18n from "../../i18n/i18n";
import type {TorrentPiecesDocumentQuery} from "../../graphql/generated/graphql";
import {TorrentPieces} from "./TorrentPieces";

type Result={data?:TorrentPiecesDocumentQuery;error?:Error;fetching?:boolean};
const mocks=vi.hoisted(()=>({result:{} as Result,reload:vi.fn(),query:vi.fn()}));
vi.mock("urql",()=>({useQuery:(options:unknown)=>{mocks.query(options);return [mocks.result,mocks.reload];}}));
const hash="a".repeat(40);
const map={hash,total:3,observedAt:"2026-11-20T12:00:00Z",buckets:[{start:0,count:1,missing:1,downloading:0,downloaded:0},{start:1,count:1,missing:0,downloading:1,downloaded:0},{start:2,count:1,missing:0,downloading:0,downloaded:1}]};
const view=(key=hash,at="first",enabled=true)=><TorrentPieces hash={key} observedAt={at} enabled={enabled}/>;
beforeEach(async()=>{vi.clearAllMocks();await i18n.changeLanguage("en");mocks.result={data:{torrentPieces:map}};});
afterEach(cleanup);
describe("real torrent piece map",()=>{
 it("renders actual piece positions and all three states without inventing a percentage",()=>{
  const timer=vi.spyOn(window,"setInterval");const {container}=render(view());
  expect(screen.getByRole("img",{name:"3 pieces: 1 missing, 1 downloading, 1 downloaded"})).toBeTruthy();
  expect(container.querySelectorAll("svg g")).toHaveLength(3);
  expect(container.querySelector('rect.torrent-pieces__downloaded')?.getAttribute("x")).toBe("2");
  expect(container.querySelector('rect.torrent-pieces__downloading')?.getAttribute("x")).toBe("1");
  expect(timer).not.toHaveBeenCalled();timer.mockRestore();
 });
 it("refreshes with downloader observations and pauses while unavailable",()=>{
  const {rerender}=render(view());expect(mocks.reload).not.toHaveBeenCalled();
  rerender(view(hash,"second"));expect(mocks.reload).toHaveBeenCalledOnce();
  rerender(view(hash,"third",false));expect(mocks.reload).toHaveBeenCalledOnce();
  expect(mocks.query.mock.lastCall?.[0].pause).toBe(true);
  expect(screen.getByRole("status").textContent).toContain("last observation");
 });
 it("retains a confirmed map on failure and clears it on a successful empty response",()=>{
  const {rerender}=render(view());mocks.result={error:new Error("private upstream details")};rerender(view());
  expect(screen.getByRole("img")).toBeTruthy();expect(screen.queryByText("private upstream details")).toBeNull();
  expect(screen.getByRole("status").textContent).toContain("last observation");
  mocks.result={data:{torrentPieces:{...map,total:0,buckets:[]}}};rerender(view());
  expect(screen.queryByRole("img")).toBeNull();expect(screen.getByText("No piece information yet")).toBeTruthy();
 });
 it("does not show another torrent's retained map",()=>{
  const {rerender}=render(view());mocks.result={error:new Error("offline")};rerender(view("b".repeat(40)));
  expect(screen.queryByRole("img")).toBeNull();expect(screen.getByText("Piece states are temporarily unavailable")).toBeTruthy();
 });
 it("keeps initial errors and loading states distinct",()=>{
  mocks.result={fetching:true};const {rerender}=render(view());expect(screen.getByText("Loading piece states…")).toBeTruthy();
  mocks.result={error:new Error("offline")};rerender(view());expect(screen.getByText("Piece states are temporarily unavailable")).toBeTruthy();
  expect(screen.queryByRole("img")).toBeNull();
 });
});
