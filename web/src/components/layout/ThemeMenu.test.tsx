// @vitest-environment jsdom
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it } from "vitest";
import i18n from "../../i18n/i18n";
import { useTheme } from "../../hooks/useTheme";
import { ThemeMenu } from "./ThemeMenu";

function Harness() { const theme = useTheme(); return <><ThemeMenu {...theme} /><button>Next control</button></>; }
beforeEach(async () => { localStorage.clear(); await i18n.changeLanguage("en"); });
afterEach(cleanup);

it("opens at the selection, wraps arrows, selects and returns focus", async () => {
  const user = userEvent.setup();
  render(<Harness />);
  const trigger = screen.getByRole("button", { name: /Theme/ });
  trigger.focus();
  await user.keyboard("{ArrowDown}");
  const items = screen.getAllByRole("menuitemradio");
  expect(document.activeElement).toBe(screen.getByRole("menuitemradio", {checked:true}));
  await user.keyboard("{Home}{ArrowUp}");
  expect(document.activeElement).toBe(items[2]);
  await user.keyboard("{End}{ArrowDown}");
  expect(document.activeElement).toBe(items[0]);
  await user.keyboard("{Enter}");
  expect(screen.queryByRole("menu")).toBeNull();
  expect(document.activeElement).toBe(trigger);
  expect(localStorage.getItem("moji:theme")).toBe("light");
  await user.click(trigger);
  await user.keyboard("{Escape}");
  expect(document.activeElement).toBe(trigger);
  await user.click(trigger);
  await user.tab();
  expect(document.activeElement).toBe(screen.getByRole("button", {name:"Next control"}));
  expect(screen.queryByRole("menu")).toBeNull();
});
