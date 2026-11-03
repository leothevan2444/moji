// @vitest-environment jsdom
import {act,cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import i18n from "../../i18n/i18n";
import {IndexerTestDialog} from "./IndexerTestDialog";
const mock=vi.hoisted(()=>({execute:vi.fn()}));vi.mock("urql",()=>({useMutation:()=>[{},mock.execute]}));
const target={instanceId:"site",expectedRevision:"config",draft:{name:"Unsaved name"}};
const report={id:"test-correlation",instanceId:"site",kind:"BUILTIN",configRevision:"config",ruleRevision:"rules",startedAt:"2026-01-01",finishedAt:"2026-01-01",results:0,partial:false,truncated:false,stages:[{name:"CONFIGURATION",state:"PASSED",errorCode:"",elapsedMs:1},{name:"AUTHENTICATION",state:"FAILED",errorCode:"INDEXER_CHALLENGE_REQUIRED",elapsedMs:2},{name:"SEARCH",state:"SKIPPED",errorCode:"",elapsedMs:0},{name:"DOWNLOAD",state:"NOT_REQUESTED",errorCode:"",elapsedMs:0}]};
beforeEach(async()=>{await i18n.changeLanguage("en");mock.execute.mockReset().mockResolvedValue({data:{testIndexer:report}})});afterEach(cleanup);
it("does not test automatically and submits only the actual draft with default authentication scope",async()=>{
 render(<IndexerTestDialog target={target} onClose={()=>{}}/>);expect(mock.execute).not.toHaveBeenCalled();fireEvent.click(screen.getByRole("button",{name:"Start test"}));await screen.findByRole("region",{name:"Site test results"});expect(mock.execute.mock.calls[0][0].input).toEqual({...target,search:false,download:false,keywords:undefined});expect(screen.getByText("Failed")).toBeTruthy();expect(screen.getByText("Not executed")).toBeTruthy();expect(screen.getByText("Not selected")).toBeTruthy();
});
it("requires an explicit query and opt-in before resource acquisition",async()=>{
 render(<IndexerTestDialog target={target} onClose={()=>{}}/>);fireEvent.click(screen.getByRole("checkbox",{name:"Test search"}));expect(screen.getByRole("button",{name:"Start test"}).hasAttribute("disabled")).toBe(true);fireEvent.change(screen.getByRole("textbox",{name:"Search keywords"}),{target:{value:"ABCD-123"}});fireEvent.click(screen.getByRole("checkbox",{name:"Test resource acquisition"}));expect(screen.getByText(/consume a site download quota/)).toBeTruthy();fireEvent.click(screen.getByRole("button",{name:"Start test"}));await waitFor(()=>expect(mock.execute).toHaveBeenCalledTimes(1));expect(mock.execute.mock.calls[0][0].input).toMatchObject({search:true,download:true,keywords:"ABCD-123"});
});
it("aborts and ignores a late result after options change",async()=>{
 let done!:(v:unknown)=>void;mock.execute.mockImplementation(()=>new Promise(resolve=>{done=resolve}));render(<IndexerTestDialog target={target} onClose={()=>{}}/>);fireEvent.click(screen.getByRole("button",{name:"Start test"}));const signal=mock.execute.mock.calls[0][1].fetchOptions.signal as AbortSignal;fireEvent.click(screen.getByRole("checkbox",{name:"Test search"}));expect(signal.aborted).toBe(true);await act(async()=>done({data:{testIndexer:report}}));expect(screen.queryByRole("region",{name:"Site test results"})).toBeNull();expect(mock.execute).toHaveBeenCalledTimes(1);
});
it("invalidates finished results when the editing draft changes",async()=>{
 const view=render(<IndexerTestDialog target={target} onClose={()=>{}}/>);fireEvent.click(screen.getByRole("button",{name:"Start test"}));await screen.findByRole("region",{name:"Site test results"});view.rerender(<IndexerTestDialog target={{...target,draft:{name:"Different draft"}}} onClose={()=>{}}/>);expect(screen.queryByRole("region",{name:"Site test results"})).toBeNull();expect(mock.execute).toHaveBeenCalledTimes(1);
});
it("cancels without retrying and releases the in-flight request on close",async()=>{
 mock.execute.mockImplementation(()=>new Promise(()=>{}));const close=vi.fn();render(<IndexerTestDialog target={target} onClose={close}/>);fireEvent.click(screen.getByRole("button",{name:"Start test"}));const first=mock.execute.mock.calls[0][1].fetchOptions.signal as AbortSignal;fireEvent.click(screen.getByRole("button",{name:"Stop test"}));expect(first.aborted).toBe(true);expect(screen.getByRole("button",{name:"Start test"}).hasAttribute("disabled")).toBe(false);fireEvent.click(screen.getByRole("button",{name:"Start test"}));const second=mock.execute.mock.calls[1][1].fetchOptions.signal as AbortSignal;fireEvent.click(screen.getByRole("button",{name:"Close"}));expect(close).toHaveBeenCalledTimes(1);expect(second.aborted).toBe(true);expect(mock.execute).toHaveBeenCalledTimes(2);
});

it.each(["SKIPPED", "FAILED", "EMPTY", "PARTIAL"])("shows a result count only for an observed search: %s", async state => {
 mock.execute.mockResolvedValue({data:{testIndexer:{...report,stages:report.stages.map(stage=>stage.name==="SEARCH"?{...stage,state}:stage)}}});
 render(<IndexerTestDialog target={target} onClose={()=>{}}/>);
 fireEvent.click(screen.getByRole("checkbox",{name:"Test search"}));fireEvent.change(screen.getByRole("textbox",{name:"Search keywords"}),{target:{value:"ABCD-123"}});fireEvent.click(screen.getByRole("button",{name:"Start test"}));await screen.findByRole("region",{name:"Site test results"});expect(Boolean(screen.queryByText("0 search results"))).toBe(["EMPTY","PARTIAL"].includes(state));
});
