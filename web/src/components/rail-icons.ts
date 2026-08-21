/** Inline SVG path data (24 viewBox) for project rail chrome. */
export type RailIconName =
  | 'config'
  | 'files'
  | 'expand-workspace'
  | 'collapse-canvas'
  | 'show-canvas'

const icons: Record<RailIconName, string> = {
  config:
    'M4 7h10m4 0h2M4 17h2m4 0h10M14 4v6M6 14v6',
  files:
    'M4 7.5A2.5 2.5 0 0 1 6.5 5h3.2c.5 0 1 .2 1.3.6L12 7h5.5A2.5 2.5 0 0 1 20 9.5v7A2.5 2.5 0 0 1 17.5 19h-11A2.5 2.5 0 0 1 4 16.5v-9Z',
  'expand-workspace':
    'M9 4H4v5M4 4l6 6m5-6h5v5m0-5-6 6M9 20H4v-5m0 5 6-6m5 6h5v-5m0 5-6-6',
  'collapse-canvas':
    'M5 5h14a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2Zm10 2v10h4V7h-4Zm-2.5 2.5L10 12l2.5 2.5',
  'show-canvas':
    'M5 5h14a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2Zm10 2v10h4V7h-4Zm-5.5 2.5L12 12l-2.5 2.5',
}

export function railIconPath(name: RailIconName): string {
  return icons[name]
}
