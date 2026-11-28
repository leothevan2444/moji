// @vitest-environment jsdom
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { faUsers, faHeart, faBookmark } from "@fortawesome/free-solid-svg-icons";
import i18n from "../../i18n/i18n";
import { FilterCountHeading } from "./FilterCountHeading";

beforeEach(async () => { await i18n.changeLanguage("en"); });
afterEach(cleanup);
it("supports keyboard exploration without changing filters, zero counts and unknown counts", async () => {
  const onChange = vi.fn(), user = userEvent.setup();
  render(<FilterCountHeading title="Performers" kind="performers" value="all" onChange={onChange} items={[
    {id:"all",label:"All performers",icon:faUsers,count:12},
    {id:"favorite",label:"Stash favorites",icon:faHeart,count:0},
    {id:"subscribed",label:"Subscribed",icon:faBookmark,count:null},
  ]} />);
  const group = within(screen.getByRole("toolbar", {name:"Filter performers by scope"}));
  const all = group.getByRole("button", {name:"All performers 12"});
  const favorite = group.getByRole("button", {name:"Stash favorites 0"});
  const unknown = group.getByRole("button", {name:"Subscribed —"});
  expect(unknown.title).toBe("Show only Subscribed · Count unavailable");
  expect(favorite.title).toBe("Show only Stash favorites · 0 performers");
  all.focus();
  await user.keyboard("{ArrowRight}");
  expect(document.activeElement).toBe(favorite);
  expect(all.getAttribute("aria-pressed")).toBe("true");
  expect(onChange).not.toHaveBeenCalled();
  await user.keyboard("{Enter}");
  expect(onChange).toHaveBeenCalledExactlyOnceWith("favorite");
  await user.keyboard("{End}");
  expect(document.activeElement).toBe(unknown);
  await user.keyboard("{Home}");
  expect(document.activeElement).toBe(all);
  await user.keyboard(" ");
  expect(onChange).toHaveBeenCalledTimes(1);
});
