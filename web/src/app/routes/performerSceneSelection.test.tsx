// @vitest-environment jsdom
import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, expect, it } from "vitest";
import { usePerformerSceneSelection, type SceneSnapshotSource } from "./performerSceneSelection";
afterEach(cleanup);
it("removes newly deferred visible films while preserving selections on other pages", () => {
 const first: SceneSnapshotSource = {key:"first",sourceSceneId:"first",inLibrary:false};
 const other: SceneSnapshotSource = {key:"other",sourceSceneId:"other",inLibrary:false};
 const {result,rerender} = renderHook(({scenes})=>usePerformerSceneSelection("p1",scenes),{initialProps:{scenes:[first]}});
 act(()=>result.current.addVisible(["first"]));
 rerender({scenes:[other]});
 act(()=>result.current.addVisible(["other"]));
 expect(result.current.selectedKeys).toEqual(["first","other"]);
 rerender({scenes:[{...first,upcomingState:"WAITING"}]});
 expect(result.current.selectedKeys).toEqual(["other"]);
 act(()=>{result.current.toggle("first");result.current.addVisible(["first"]);});
 expect(result.current.selectedKeys).toEqual(["other"]);
});
