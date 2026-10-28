export interface NavItem {
  key: string;
  label: string;
  href: string;
  sheet: string;
}

/** The main sections of the public site, with their drawing-sheet numbers. */
export const navItems: NavItem[] = [
  { key: "residences", label: "Residences", href: "/residences", sheet: "A-200" },
  { key: "explore", label: "Explore in 3D", href: "/explore", sheet: "A-100" },
  { key: "building", label: "The building", href: "/building", sheet: "A-300" },
  { key: "location", label: "Location", href: "/location", sheet: "A-400" },
  { key: "progress", label: "Progress", href: "/progress", sheet: "A-500" },
  { key: "buying", label: "Buying", href: "/buying", sheet: "A-600" },
];
