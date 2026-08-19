// web/src/shell/sidebar-state.ts
export const SIDEBAR_COLLAPSED_KEY = 'pa.sidebarCollapsed';
export const readSidebarCollapsed = (storage: Storage) => storage.getItem(SIDEBAR_COLLAPSED_KEY) === 'true';
export const writeSidebarCollapsed = (storage: Storage, value: boolean) =>
  storage.setItem(SIDEBAR_COLLAPSED_KEY, String(value));
