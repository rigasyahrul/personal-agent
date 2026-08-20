// web/src/shell/nav-icons.ts
/** Inline SVG path data (24 viewBox) for shell nav — one consistent set. */
export type NavIconName =
  | 'home'
  | 'projects'
  | 'sessions'
  | 'vaults'
  | 'review'
  | 'settings'
  | 'panel-left'
  | 'panel-left-close';

const icons: Record<NavIconName, string> = {
  home: 'M4 10.5 12 4l8 6.5V20a1 1 0 0 1-1 1h-5v-6H10v6H5a1 1 0 0 1-1-1v-9.5Z',
  projects:
    'M4 7.5A2.5 2.5 0 0 1 6.5 5h3.2c.5 0 1 .2 1.3.6L12 7h5.5A2.5 2.5 0 0 1 20 9.5v7A2.5 2.5 0 0 1 17.5 19h-11A2.5 2.5 0 0 1 4 16.5v-9Z',
  sessions:
    'M5 6.5A2.5 2.5 0 0 1 7.5 4h9A2.5 2.5 0 0 1 19 6.5v7A2.5 2.5 0 0 1 16.5 16H10l-3.8 3.2c-.5.4-1.2 0-1.2-.7V16h-.5A2.5 2.5 0 0 1 5 13.5v-7Z',
  vaults:
    'M6.5 4h11A2.5 2.5 0 0 1 20 6.5v11a2.5 2.5 0 0 1-2.5 2.5h-11A2.5 2.5 0 0 1 4 17.5v-11A2.5 2.5 0 0 1 6.5 4Zm2 6.5h7v5h-7v-5Zm1.5-3a1.5 1.5 0 1 0 3 0 1.5 1.5 0 0 0-3 0Z',
  review:
    'M8 5h10a1 1 0 0 1 1 1v12.2a.8.8 0 0 1-1.3.6L14 16H8a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1Zm-3 3h1v9.5A2.5 2.5 0 0 0 8.5 20H16v1H8.5A3.5 3.5 0 0 1 5 17.5V8Z',
  settings:
    'M12 8.5a3.5 3.5 0 1 1 0 7 3.5 3.5 0 0 1 0-7Zm8.1 2.6-1.4-.3a6.7 6.7 0 0 0-.6-1.4l.8-1.2-1.5-1.5-1.2.8c-.4-.3-.9-.5-1.4-.6l-.3-1.4h-2.2l-.3 1.4c-.5.1-1 .3-1.4.6l-1.2-.8-1.5 1.5.8 1.2c-.3.4-.5.9-.6 1.4l-1.4.3v2.2l1.4.3c.1.5.3 1 .6 1.4l-.8 1.2 1.5 1.5 1.2-.8c.4.3.9.5 1.4.6l.3 1.4h2.2l.3-1.4c.5-.1 1-.3 1.4-.6l1.2.8 1.5-1.5-.8-1.2c.3-.4.5-.9.6-1.4l1.4-.3v-2.2Z',
  'panel-left':
    'M5 5h14a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2Zm0 2v10h4V7H5Zm6 0v10h8V7h-8Z',
  'panel-left-close':
    'M5 5h14a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2Zm0 2v10h4V7H5Zm9.3 2.3 1.4 1.4-1.8 1.8 1.8 1.8-1.4 1.4-1.8-1.8-1.8 1.8-1.4-1.4 1.8-1.8-1.8-1.8 1.4-1.4 1.8 1.8 1.8-1.8Z',
};

export function navIconPath(name: NavIconName): string {
  return icons[name];
}

export function iconForLabel(label: string): NavIconName {
  const key = label.toLowerCase() as NavIconName;
  if (key in icons) return key;
  return 'home';
}
