import { describe, expect, it } from "vitest";
import { ActivityEvaluationStatus, type PerformerActivityQuery } from "../../graphql/generated/graphql";
import { activityDisplayStatus, activityRankForDisplay } from "./performerActivityStatus";

type Activity=PerformerActivityQuery["performerActivity"];
const initial:Activity={id:"p",evaluationStatus:ActivityEvaluationStatus.NotChecked,lowerBound:false};
describe("activity availability",()=>{
 it("distinguishes loading from a failed read and from an unassessed actor",()=>{
  expect(activityDisplayStatus(undefined,false)).toBe("LOADING");
  expect(activityDisplayStatus(undefined,true)).toBe("READ_FAILED");
  expect(activityDisplayStatus(initial,false)).toBe("NOT_CHECKED");
 });
 it("shows actual assessment only when an admitted check is running without an observation",()=>{
  expect(activityDisplayStatus(initial,false,"PENDING")).toBe("WAITING_EVALUATION");
  expect(activityDisplayStatus(initial,false,"RUNNING")).toBe("EVALUATING");
  expect(activityDisplayStatus({...initial,evaluationStatus:ActivityEvaluationStatus.InvalidDates,observedAt:"2026-11-09T00:00:00Z"},false,"RUNNING")).toBe("INVALID_DATES");
 });
 it("preserves a confirmed rating during read errors and subsequent checks",()=>{
  expect(activityDisplayStatus({...initial,value:3,rank:4,evaluationStatus:ActivityEvaluationStatus.Ready},true,"RUNNING")).toBe("READY");
 });
 it.each([ActivityEvaluationStatus.IncompleteCatalog,ActivityEvaluationStatus.InvalidDates,ActivityEvaluationStatus.InvalidMembership,ActivityEvaluationStatus.CheckFailed])("retains backend evidence reason %s",evaluationStatus=>{
  expect(activityDisplayStatus({...initial,evaluationStatus},false)).toBe(evaluationStatus);
 });
});

it("distinguishes a confirmed zero value from an unknown rating",()=>{
 expect(activityRankForDisplay({...initial,value:0,rank:1})).toBe(1);
 expect(activityRankForDisplay({...initial,value:null,rank:4})).toBe(0);
 expect(activityRankForDisplay({...initial,value:3,rank:null})).toBe(0);
 expect(activityRankForDisplay(undefined)).toBe(0);
});
