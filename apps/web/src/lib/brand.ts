// Brand constants shared by everything that can't read CSS variables: the
// web app manifest, the <meta name="theme-color"> tags, and the Safari
// pinned-tab colour. globals.css holds the same palette as design tokens;
// keep the two in step (docs/BRAND.md is the source of truth).
export const brand = {
  name: "Amorae",
  tagline: "Two hearts, one faith.",
  description: "A private space for the life you are building together.",
  colors: {
    plum: "#5B2A4A",
    background: "#F7F4EF",
    ink: "#24201F",
    softWhite: "#FFFDFA",
    // The dark-mode page background, used for the dark theme-color so the
    // browser chrome matches the page instead of flashing the light colour.
    darkBackground: "#1A1716",
  },
} as const;
