// For things that can't read CSS variables (manifest, theme-color meta, pinned
// tab). Keep in step with globals.css's full token set (docs/BRAND.md is the source of truth).
export const brand = {
  name: "Amorae",
  tagline: "Two hearts, one faith.",
  description: "A private space for the life you are building together.",
  colors: {
    plum: "#5B2A4A",
    // The mark and accent on dark backgrounds, per docs/BRAND.md.
    plumDark: "#E0B4CD",
    background: "#F7F4EF",
    backgroundDark: "#16130F",
    ink: "#24201F",
    softWhite: "#FFFDFA",
  },
} as const;
