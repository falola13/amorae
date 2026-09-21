// iOS startup images. One per device size; Safari picks the one whose media query matches exactly.
const DEVICES: [number, number, number][] = [
  [440, 956, 3], [402, 874, 3], [430, 932, 3], [393, 852, 3], [428, 926, 3], [390, 844, 3], [375, 812, 3], [414, 896, 3], [414, 896, 2], [414, 736, 3], [375, 667, 2], [320, 568, 2],
  [768, 1024, 2], [834, 1194, 2], [1024, 1366, 2],
];
export const SPLASH_LINKS = DEVICES.map(([w, h, dpr]) => ({
  media: `screen and (device-width: ${w}px) and (device-height: ${h}px) and (-webkit-device-pixel-ratio: ${dpr}) and (orientation: portrait)`,
  href: `/splash/splash-${w * dpr}x${h * dpr}.png`,
}));
