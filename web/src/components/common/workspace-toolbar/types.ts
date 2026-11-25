import type { IconDefinition } from "@fortawesome/free-solid-svg-icons";

export interface ToolbarAction {
  id: string;
  label: string;
  title?: string;
  icon: IconDefinition;
  onClick?: () => void;
  href?: string;
  disabled?: boolean;
  busy?: boolean;
  spinning?: boolean;
  primary?: boolean;
  danger?: boolean;
  showLabel?: boolean;
}
export interface ToolbarFilter {
  id: string;
  label: string;
  value: string;
  options: readonly { value: string; label: string }[];
  onChange: (value: string) => void;
}
export interface ToolbarContent {
  search?: { label: string; value: string; onChange: (value: string) => void };
  filters?: readonly ToolbarFilter[];
  actions: readonly ToolbarAction[];
  summary?: string;
}
