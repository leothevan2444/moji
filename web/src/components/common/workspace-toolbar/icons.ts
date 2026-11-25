import {
  faBinoculars,
  faClipboardList,
  faClipboardQuestion,
  faDownload,
  faEllipsis,
  faEraser,
  faListCheck,
  faRotate,
  faRotateRight,
  faSquareCheck,
  faTrashCan,
  faUserMinus,
  faUserPlus,
  faXmark,
} from "@fortawesome/free-solid-svg-icons";

// Shared action meanings across the task, performer and scene toolbars.
export const workspaceToolbarIcons = {
  enterSelection: faListCheck,
  exitSelection: faXmark,
  selectPage: faSquareCheck,
  clearSelection: faEraser,
  refresh: faRotate,
  queue: faClipboardList,
  retry: faRotateRight,
  delete: faTrashCan,
  subscribe: faUserPlus,
  unsubscribe: faUserMinus,
  check: faBinoculars,
  review: faClipboardQuestion,
  createDownload: faDownload,
  more: faEllipsis,
} as const;
