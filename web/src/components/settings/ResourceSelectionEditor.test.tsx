// @vitest-environment jsdom
import { useState } from "react";
import { fireEvent, render, screen, within, cleanup } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { ResourceSelectionEditor } from "./ResourceSelectionEditor";
import { DEFAULT_RESOURCE_SELECTION } from "../../constants/forms";

vi.mock("react-i18next", async importOriginal => ({ ...await importOriginal<typeof import("react-i18next")>(), useTranslation: () => ({ t: (key: string) => key }) }));
afterEach(cleanup);
const rule = { id: "shared", name: "Shared", conditions:[{scope:"TITLE",value:"user label",minimum:"",maximum:""}] };
const group = (id: string) => ({ id, name: id, fallback:false, require: [], exclude: [], tiers: [{ id: "first", name: "First", rules: ["shared"] }] });
function SharedHarness({ active = true }: { active?: boolean }) {
  const [value, setValue] = useState<typeof DEFAULT_RESOURCE_SELECTION>({ ...DEFAULT_RESOURCE_SELECTION, rules: [rule], activeGroup: "one", groups: [group("one"), group("two")] });
  return <ResourceSelectionEditor value={value} onChange={setValue} active={active} />;
}
const close = () => fireEvent.click(screen.getByRole("button", { name: "common.close" }));

it("shows two settings cards and opens rule fields only in a separate dialog", () => {
  const view = render(<SharedHarness />);
  expect(view.container.querySelectorAll(".resource-selection > .drawer-card")).toHaveLength(2);
  const tile = screen.getByRole("button", { name: "Shared" });
  expect(tile.textContent).toBe("Shared");
  expect(screen.queryByRole("textbox")).toBeNull();
  tile.focus();
  fireEvent.click(tile);
  expect(screen.getByRole("dialog", { name: "Shared" })).toBeTruthy();
  fireEvent.change(screen.getByRole("textbox", { name: "resourceSelection.name" }), { target: { value: "Renamed" } });
  fireEvent.keyDown(document, { key: "Escape" });
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(document.activeElement).toBe(screen.getByRole("button", { name: "Renamed" }));
  fireEvent.click(screen.getByRole("button", { name: "Renamed" }));
  expect((screen.getByRole("textbox", { name: "resourceSelection.name" }) as HTMLInputElement).value).toBe("Renamed");
});

it.each(["card", "editor"])("deletes a referenced rule from the %s only after confirming its affected groups", entry => {
  const changes = vi.fn();
  render(<ResourceSelectionEditor value={{ ...DEFAULT_RESOURCE_SELECTION, rules: [rule], activeGroup: "one", groups: [group("one"), group("two")] }} onChange={changes} />);
  if (entry === "editor") fireEvent.click(screen.getByRole("button", { name: "Shared" }));
  fireEvent.click(screen.getByRole("button", { name: entry === "editor" ? "common.delete" : "resourceSelection.deleteRuleNamed" }));
  const confirm = screen.getByRole("dialog", { name: "resourceSelection.confirmDeleteRule" });
  expect(within(confirm).getByText("resourceSelection.deleteReferencedRule")).toBeTruthy();
  expect(changes).not.toHaveBeenCalled();
  fireEvent.click(within(confirm).getByRole("button", { name: "common.delete" }));
  expect(changes.mock.calls[0][0].rules).toEqual([]);
  expect(changes.mock.calls[0][0].groups.map((group: {tiers:{rules:unknown[]}[]}) => group.tiers.map(tier => tier.rules))).toEqual([[[]], [[]]]);
});

it("edits a literal expression without an operator or rule combination selector",()=>{
 render(<SharedHarness/>);fireEvent.click(screen.getByRole("button",{name:"Shared"}));
 expect((screen.getByRole("textbox",{name:"resourceSelection.expression"}) as HTMLInputElement).value).toBe("user label");
 expect(screen.queryByRole("combobox",{name:"resourceSelection.operator"})).toBeNull();
 expect(screen.queryByRole("combobox",{name:"resourceSelection.mode"})).toBeNull();
 fireEvent.change(screen.getByRole("textbox",{name:"resourceSelection.expression"}),{target:{value:"/alpha|beta/"}});close();
 fireEvent.click(screen.getByRole("button",{name:"Shared"}));expect((screen.getByRole("textbox",{name:"resourceSelection.expression"}) as HTMLInputElement).value).toBe("/alpha|beta/");
});

it("keeps save failures visible in the open dialog and after closing it", () => {
  render(<ResourceSelectionEditor value={{ ...DEFAULT_RESOURCE_SELECTION, rules: [rule] }} onChange={vi.fn()} saveStatus={<p role="alert">Unsaved edits</p>} />);
  fireEvent.click(screen.getByRole("button", { name: "Shared" }));
  expect(within(screen.getByRole("dialog")).getByRole("alert").textContent).toBe("Unsaved edits");
  close();
  expect(screen.getByRole("alert").textContent).toBe("Unsaved edits");
});

it("dismisses the portal when its settings subpanel becomes inactive", () => {
  const view = render(<SharedHarness />);
  fireEvent.click(screen.getByRole("button", { name: "Shared" }));
  view.rerender(<SharedHarness active={false} />);
  expect(screen.queryByRole("dialog")).toBeNull();
  view.rerender(<SharedHarness />);
  expect(screen.queryByRole("dialog")).toBeNull();
});

it("reorders real tiers independently of the fallback switch",()=>{
 render(<SharedHarness/>);fireEvent.click(screen.getByRole("button",{name:"one"}));
 fireEvent.click(screen.getByRole("switch",{name:"resourceSelection.allowFallback"}));
 fireEvent.click(screen.getByRole("button",{name:"resourceSelection.addTier"}));
 fireEvent.keyDown(screen.getAllByRole("button",{name:"resourceSelection.dragTier"})[1],{key:"ArrowLeft"});
 close();fireEvent.click(screen.getByRole("button",{name:"one"}));
 expect(screen.getAllByRole("textbox",{name:"resourceSelection.tierName"}).map(input=>(input as HTMLInputElement).value)).toEqual(["resourceSelection.tierNumber","First"]);
 expect((screen.getByRole("switch",{name:"resourceSelection.allowFallback"}) as HTMLInputElement).checked).toBe(true);
 expect(screen.queryByRole("combobox",{name:"resourceSelection.tierCombination"})).toBeNull();
});
it("allows removing every tier and explains fallback on and off",()=>{
 render(<SharedHarness/>);fireEvent.click(screen.getByRole("button",{name:"one"}));
 fireEvent.click(screen.getByRole("button",{name:"resourceSelection.removeTier"}));
 expect(screen.queryByRole("textbox",{name:"resourceSelection.tierName"})).toBeNull();
 expect(screen.getByText("resourceSelection.noTiersReject")).toBeTruthy();
 fireEvent.click(screen.getByRole("switch",{name:"resourceSelection.allowFallback"}));
 expect(screen.getByText("resourceSelection.noTiersFallback")).toBeTruthy();
});

it("keeps an invalid unnamed draft accessible after the editor closes", () => {
  render(<SharedHarness />);
  fireEvent.click(screen.getByRole("button", { name: "Shared" }));
  fireEvent.change(screen.getByRole("textbox", { name: "resourceSelection.name" }), { target: { value: "" } });
  close();
  fireEvent.click(screen.getByRole("button", { name: "resourceSelection.ruleNumber" }));
  expect((screen.getByRole("textbox", { name: "resourceSelection.name" }) as HTMLInputElement).value).toBe("");
});

it("uses positive IDs in both global and tier references", () => {
  const changes = vi.fn();
  render(<ResourceSelectionEditor value={{...DEFAULT_RESOURCE_SELECTION,rules:[rule],activeGroup:"one",groups:[group("one")]}} onChange={changes} />);
  fireEvent.click(screen.getByRole("button",{name:"one"}));
  expect(screen.queryByRole("menuitem")).toBeNull();
  fireEvent.click(screen.getByRole("button",{name:"resourceSelection.removeReference"}));
  expect(changes.mock.calls.at(-1)![0].groups[0].tiers[0].rules).toEqual([]);
});

it("sorts pre-filter tags by the current library order without drag handles", () => {
 const value={...DEFAULT_RESOURCE_SELECTION,rules:[rule,{...rule,id:"other",name:"Other"}],activeGroup:"one",groups:[{...group("one"),require:["other","shared"]}]};
 const view=render(<ResourceSelectionEditor value={value} onChange={vi.fn()} />);fireEvent.click(screen.getByRole("button",{name:"one"}));
 const required=screen.getByRole("group",{name:"resourceSelection.require"});
 const names=()=>within(required).getAllByRole("listitem").map(item=>item.querySelector(".resource-rule-tags__name")?.textContent);
 expect(names()).toEqual(["Shared","Other"]);expect(required.querySelector("[draggable]")).toBeNull();
 view.rerender(<ResourceSelectionEditor value={{...value,rules:[...value.rules].reverse()}} onChange={vi.fn()} />);
 expect(names()).toEqual(["Other","Shared"]);
});

it("adds unlimited mixed conditions without exposing a rule combination", () => {
  render(<SharedHarness />);
  fireEvent.click(screen.getByRole("button", {name:"Shared"}));
  for (let i=0;i<10;i++) fireEvent.click(screen.getByRole("button",{name:"resourceSelection.addCondition"}));
  expect(screen.getAllByRole("combobox",{name:"resourceSelection.scope"})).toHaveLength(11);
  expect(within(screen.getByRole("dialog")).getAllByRole("combobox")).toHaveLength(11);
  fireEvent.change(screen.getAllByRole("combobox",{name:"resourceSelection.scope"})[1],{target:{value:"SIZE"}});
  fireEvent.change(screen.getByRole("textbox",{name:"resourceSelection.minimumGiB"}),{target:{value:"2.5"}});
  fireEvent.change(screen.getAllByRole("textbox",{name:"resourceSelection.expression"})[9],{target:{value:"/alpha|beta/"}});
  fireEvent.click(screen.getAllByRole("button",{name:"resourceSelection.removeCondition"})[0]);
  close();fireEvent.click(screen.getByRole("button",{name:"Shared"}));
  expect(screen.getAllByRole("combobox",{name:"resourceSelection.scope"})).toHaveLength(10);
  expect((screen.getByRole("textbox",{name:"resourceSelection.minimumGiB"}) as HTMLInputElement).value).toBe("2.5");
  expect((screen.getAllByRole("textbox",{name:"resourceSelection.expression"}).at(-1) as HTMLInputElement).value).toBe("/alpha|beta/");
});

it("creates a fallback-only group without requiring a rule library",()=>{
 function Harness(){const [value,setValue]=useState({...DEFAULT_RESOURCE_SELECTION});return <ResourceSelectionEditor value={value} onChange={setValue}/>;}
 render(<Harness/>);fireEvent.click(screen.getByRole("button",{name:"resourceSelection.addGroup"}));
 expect(screen.getByText("resourceSelection.noTiersReject")).toBeTruthy();
 fireEvent.click(screen.getByRole("switch",{name:"resourceSelection.allowFallback"}));
 expect(screen.getByText("resourceSelection.noTiersFallback")).toBeTruthy();
});

it("switches the active group from its card without opening an editor", () => {
  render(<SharedHarness />);
  expect(screen.queryByRole("combobox", { name: "resourceSelection.activeGroup" })).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "resourceSelection.useGroup" }));
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(screen.getByRole("button", { name: "resourceSelection.activeGroupNamed" }).hasAttribute("disabled")).toBe(true);
  expect(screen.getByRole("button", { name: "resourceSelection.activeGroupNamed" }).getAttribute("aria-pressed")).toBe("true");
});

it.each(["card", "editor"])("requires confirmation to delete a rule from the %s", entry => {
  const changes = vi.fn();
  function Harness() {
    const [value, setValue] = useState<typeof DEFAULT_RESOURCE_SELECTION>({ ...DEFAULT_RESOURCE_SELECTION, rules: [rule] });
    return <ResourceSelectionEditor value={value} onChange={next => { setValue(next); changes(next); }} />;
  }
  render(<Harness />);
  const begin = () => {
    if (entry === "editor") fireEvent.click(screen.getByRole("button", { name: "Shared" }));
    fireEvent.click(screen.getByRole("button", { name: entry === "editor" ? "common.delete" : "resourceSelection.deleteRuleNamed" }));
  };
  begin();
  expect(changes).not.toHaveBeenCalled();
  fireEvent.click(within(screen.getByRole("dialog", { name: "resourceSelection.confirmDeleteRule" })).getByRole("button", { name: "common.cancel" }));
  if (entry === "editor") close();
  expect(changes).not.toHaveBeenCalled();
  begin();
  fireEvent.click(within(screen.getByRole("dialog", { name: "resourceSelection.confirmDeleteRule" })).getByRole("button", { name: "common.delete" }));
  expect(changes.mock.calls.at(-1)![0].rules).toEqual([]);
  expect(screen.queryByRole("dialog")).toBeNull();
});

it.each(["card", "editor"])("requires confirmation to delete an active group from the %s", entry => {
  const changes = vi.fn();
  function Harness() {
    const [value, setValue] = useState<typeof DEFAULT_RESOURCE_SELECTION>({ ...DEFAULT_RESOURCE_SELECTION, rules: [rule], groups: [group("one"), group("two")], activeGroup: "one" });
    return <ResourceSelectionEditor value={value} onChange={next => { setValue(next); changes(next); }} />;
  }
  render(<Harness />);
  if (entry === "editor") fireEvent.click(screen.getByRole("button", { name: "one" }));
  fireEvent.click(screen.getAllByRole("button", { name: entry === "editor" ? "resourceSelection.deleteGroup" : "resourceSelection.deleteGroupNamed" })[0]);
  expect(changes).not.toHaveBeenCalled();
  const confirm = screen.getByRole("dialog", { name: "resourceSelection.confirmDeleteGroup" });
  expect(within(confirm).getByText("resourceSelection.deleteActiveNext")).toBeTruthy();
  fireEvent.click(within(confirm).getByRole("button", { name: "common.delete" }));
  expect(changes.mock.calls.at(-1)![0].activeGroup).toBe("two");
  expect(changes.mock.calls.at(-1)![0].rules).toEqual([rule]);
});

it("reorders library rules without changing any group references", () => {
  const changes = vi.fn();
  function Harness() {
    const [value, setValue] = useState<typeof DEFAULT_RESOURCE_SELECTION>({ ...DEFAULT_RESOURCE_SELECTION, rules: [rule, { ...rule, id: "other", name: "Other" }], groups: [group("one")], activeGroup: "one" });
    return <ResourceSelectionEditor value={value} onChange={next => { setValue(next); changes(next); }} />;
  }
  render(<Harness />);
  fireEvent.keyDown(screen.getAllByRole("button", { name: "resourceSelection.dragRule" })[0], { key: "ArrowRight" });
  expect(changes.mock.calls.at(-1)![0].rules.map((rule: {id: string}) => rule.id)).toEqual(["other", "shared"]);
  expect(changes.mock.calls.at(-1)![0].groups).toEqual([group("one")]);
  expect(screen.queryByRole("dialog")).toBeNull();
});

it("keeps admission and exclusion mutually exclusive and makes removed rules available again", () => {
  function Harness() {
    const [value, setValue] = useState<typeof DEFAULT_RESOURCE_SELECTION>({ ...DEFAULT_RESOURCE_SELECTION, rules: [rule, { ...rule, id: "other", name: "Other" }], groups: [{ ...group("one"), require: ["shared"], tiers: [] }], activeGroup: "one" });
    return <ResourceSelectionEditor value={value} onChange={setValue} />;
  }
  render(<Harness />);fireEvent.click(screen.getByRole("button", { name: "one" }));
  const admission = screen.getByRole("group", { name: "resourceSelection.require" });
  const excluded = screen.getByRole("group", { name: "resourceSelection.exclude" });
  const picker = within(excluded).getByRole("combobox");
  expect(within(picker).queryByRole("option", { name: "Shared" })).toBeNull();
  fireEvent.click(within(admission).getByRole("button", { name: "resourceSelection.removeReference" }));
  expect(within(picker).getByRole("option", { name: "Shared" })).toBeTruthy();
  fireEvent.change(picker, { target: { value: "shared" } });
  expect(within(within(admission).getByRole("combobox")).queryByRole("option", { name: "Shared" })).toBeNull();
});

it("opens either editor from card padding and keeps controls independent", () => {
  const view = render(<SharedHarness />);
  fireEvent.click(view.container.querySelector(".resource-selection__rule-tile")!);
  expect(screen.getByRole("dialog", { name: "Shared" })).toBeTruthy(); close();
  fireEvent.click(view.container.querySelector(".resource-selection__card-actions")!);
  expect(screen.getByRole("dialog", { name: "one" })).toBeTruthy(); close();
  fireEvent.click(screen.getAllByRole("button", { name: "resourceSelection.dragGroup" })[0]);
  expect(screen.queryByRole("dialog")).toBeNull();
  fireEvent.keyDown(screen.getAllByRole("button", { name: "resourceSelection.dragGroup" })[0], { key: "ArrowRight" });
  expect(screen.getAllByRole("button", { name: /^(one|two)$/ }).map(button => button.getAttribute("aria-label"))).toEqual(["two", "one"]);
  expect(screen.getByRole("button", { name: "resourceSelection.activeGroupNamed" }).getAttribute("aria-pressed")).toBe("true");
});

it("prevents priority rules from also being excluded", () => {
  render(<SharedHarness />);fireEvent.click(screen.getByRole("button",{name:"one"}));
  const excluded=screen.getByRole("group",{name:"resourceSelection.exclude"});
  expect(within(within(excluded).getByRole("combobox")).queryByRole("option",{name:"Shared"})).toBeNull();
});

it("adds an empty priority tier even without any library rules", () => {
  const changes = vi.fn();
  function Harness() {
    const [value, setValue] = useState<typeof DEFAULT_RESOURCE_SELECTION>({ ...DEFAULT_RESOURCE_SELECTION, rules: [], activeGroup: "one", groups: [{ ...group("one"), tiers: [] }] });
    return <ResourceSelectionEditor value={value} onChange={next => { setValue(next); changes(next); }} />;
  }
  render(<Harness />);fireEvent.click(screen.getByRole("button", { name: "one" }));
  fireEvent.click(screen.getByRole("button", { name: "resourceSelection.addTier" }));
  expect(changes.mock.calls.at(-1)![0].groups[0].tiers[0].rules).toEqual([]);
  expect(screen.queryByRole("button", {name:"resourceSelection.moveUp"})).toBeNull();
});

it("edits rule and group names directly in their dialog headings", () => {
  const view = render(<SharedHarness />);
  for (const [name, label, renamed] of [["Shared", "resourceSelection.name", "Renamed rule"], ["one", "resourceSelection.groupName", "Renamed group"]]) {
    fireEvent.click(screen.getByRole("button", {name}));
    const dialog = screen.getByRole("dialog", {name});
    const title = within(dialog).getByRole("textbox", {name:label});
    expect(title.closest(".drawer__head")).toBeTruthy();
    expect(within(dialog).queryByRole("heading", {name:"resourceSelection.editRule"})).toBeNull();
    fireEvent.change(title, {target:{value:renamed}});
    expect(dialog.getAttribute("aria-label")).toBe(renamed);
    fireEvent.keyDown(title, {key:"Enter"});
    expect(view.container.querySelector(".resource-selection")).toBeTruthy();
    close();expect(screen.getByRole("button", {name:renamed})).toBeTruthy();
  }
});

it("opens rule and group editors on the panel and keeps titles in the tab sequence", () => {
  render(<SharedHarness />);
  for (const [name, label] of [["Shared", "resourceSelection.name"], ["one", "resourceSelection.groupName"]]) {
    const opener = screen.getByRole("button", {name});opener.focus();fireEvent.click(opener);
    const dialog = screen.getByRole("dialog", {name});expect(document.activeElement).toBe(dialog);
    fireEvent.keyDown(document, {key:"Tab"});expect(document.activeElement).toBe(within(dialog).getByRole("textbox", {name:label}));
    fireEvent.keyDown(document, {key:"Escape"});expect(document.activeElement).toBe(opener);
  }
});
