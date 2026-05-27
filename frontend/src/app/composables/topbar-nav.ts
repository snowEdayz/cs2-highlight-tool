/** Route name constants for use with vue-router */
export const ROUTE_NAMES = {
  IMPORT: "import-actions",
  IMPORT_WANMEI: "import-wanmei",
  IMPORT_5E: "import-5e",
  CLIPS: "clips",
  PRODUCE: "produce",
  EDIT: "edit",
  SETTINGS: "settings",
} as const;

export type RouteName = (typeof ROUTE_NAMES)[keyof typeof ROUTE_NAMES];

export interface NavItem {
  label: string;
  routeName: RouteName;
  icon?: string;
}

/** Main navigation items for the top bar */
export const mainNavItems: NavItem[] = [
  { label: "import", routeName: ROUTE_NAMES.IMPORT },
  { label: "clips", routeName: ROUTE_NAMES.CLIPS },
  { label: "produce", routeName: ROUTE_NAMES.PRODUCE },
  { label: "edit", routeName: ROUTE_NAMES.EDIT },
];
