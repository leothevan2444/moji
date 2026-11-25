// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "../../../i18n/i18n";
import { Pagination } from "./Pagination";

afterEach(cleanup);
beforeEach(async () => { await i18n.changeLanguage("en"); });
const base = {label:"Scenes",position:"top" as const,page:3,pageSize:24,totalCount:120,totalPages:5,onPageChange:vi.fn(),onPageSizeChange:vi.fn()};

describe("shared pagination", () => {
  it("keeps unknown totals truthful, bounds jumps to loaded pages, and allows loading the next page", () => {
    const onPageChange = vi.fn();
    const view = render(<Pagination {...base} totalPages={null} totalCount={64} hasNextPage onPageChange={onPageChange} />);
    expect(screen.getByRole("status").textContent).toBe("Page 3 / —, 64 loaded");
    expect((screen.getByRole("button",{name:"Last page"}) as HTMLButtonElement).disabled).toBe(true);
    const input = screen.getByRole("textbox",{name:"Page number"});
    fireEvent.change(input,{target:{value:"4"}});fireEvent.submit(input.closest("form")!);
    expect(onPageChange).not.toHaveBeenCalled();
    expect((input as HTMLInputElement).validationMessage).toContain("1 to 3");
    fireEvent.change(input,{target:{value:"2"}});fireEvent.submit(input.closest("form")!);
    expect(onPageChange).toHaveBeenLastCalledWith(2);
    fireEvent.click(screen.getByRole("button",{name:"Previous"}));
    expect(onPageChange).toHaveBeenLastCalledWith(2);
    fireEvent.click(screen.getByRole("button",{name:"Next"}));
    expect(onPageChange).toHaveBeenLastCalledWith(4);
    view.rerender(<Pagination {...base} onPageChange={onPageChange} />);
    expect(screen.getByRole("status").textContent).toBe("Page 3 / 5, 120 items");
    fireEvent.click(screen.getByRole("button",{name:"Last page"}));
    expect(onPageChange).toHaveBeenLastCalledWith(5);
  });

  it("keeps page size usable on empty results and blocks navigation while a request is pending", () => {
    const onPageChange=vi.fn(), onPageSizeChange=vi.fn();
    const view=render(<Pagination {...base} page={1} totalCount={0} totalPages={0} onPageChange={onPageChange} onPageSizeChange={onPageSizeChange} />);
    expect(screen.getByRole("status").textContent).toBe("Page 0 / 0, 0 items");
    for(const name of ["First page","Previous","Next","Last page"]) expect((screen.getByRole("button",{name}) as HTMLButtonElement).disabled).toBe(true);
    fireEvent.change(screen.getByRole("combobox",{name:"Page size"}),{target:{value:"96"}});
    expect(onPageSizeChange).toHaveBeenCalledWith(96);
    view.rerender(<Pagination {...base} loading onPageChange={onPageChange} />);
    const input=screen.getByRole("textbox",{name:"Page number"});
    fireEvent.change(input,{target:{value:"4"}});fireEvent.submit(input.closest("form")!);
    expect(onPageChange).not.toHaveBeenCalled();
    expect((input as HTMLInputElement).disabled).toBe(true);
  });

  it("synchronizes the two controlled instances after jumping and changing page size", () => {
    function Pair(){
      const [page,setPage]=useState(3),[pageSize,setSize]=useState(24);
      return <>{(["top","bottom"] as const).map(position=><Pagination {...base} key={position} position={position} page={page} pageSize={pageSize}
        onPageChange={setPage} onPageSizeChange={size=>{setSize(size);setPage(1);}} />)}</>;
    }
    render(<Pair/>);
    const top=within(screen.getByRole("navigation",{name:"Scenes pagination (top)"}));
    const bottom=within(screen.getByRole("navigation",{name:"Scenes pagination (bottom)"}));
    const input=top.getByRole("textbox",{name:"Page number"});
    for(const value of ["", "0", "6", "2.5", "1e1", "-1"]) {
      fireEvent.change(input,{target:{value}});fireEvent.submit(input.closest("form")!);
      expect((bottom.getByRole("textbox") as HTMLInputElement).value).toBe("3");
    }
    fireEvent.change(input,{target:{value:"4"}});fireEvent.submit(input.closest("form")!);
    expect((bottom.getByRole("textbox") as HTMLInputElement).value).toBe("4");
    fireEvent.change(bottom.getByRole("combobox",{name:"Page size"}),{target:{value:"48"}});
    expect((top.getByRole("combobox",{name:"Page size"}) as HTMLSelectElement).value).toBe("48");
    expect((top.getByRole("textbox") as HTMLInputElement).value).toBe("1");
  });

  it("focuses options, dismisses with Escape, and restores focus to the trigger", async () => {
    render(<Pagination {...base}/>);
    const user=userEvent.setup(), trigger=screen.getByRole("button",{name:"More pagination options"});
    await user.click(trigger);
    const panel=screen.getByRole("dialog",{name:"More pagination options"});
    expect(document.activeElement).toBe(within(panel).getByRole("combobox",{name:"Page size"}));
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(document.activeElement).toBe(trigger);
  });
});
