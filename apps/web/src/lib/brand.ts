// Brand constants shared by everything that can't read CSS variables: the
// web app manifest, the <meta name="theme-color"> tags, and the Safari
// pinned-tab colour. globals.css holds the full token set; keep the two in
// step (docs/BRAND.md is the source of truth).
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
