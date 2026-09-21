// iOS ignores the manifest for launch screens: it only shows an
// apple-touch-startup-image whose media query matches the device exactly,
// and falls back to a blank screen otherwise. One entry per portrait size.
// When Apple ships a new screen size, add its PNG under public/splash and a
// line here.

interface Device {
  width: number; // CSS pixels
  height: number;
  ratio: number; // device pixel ratio
  models: string; // for humans, not used at runtime
}

const devices: Device[] = [
  { width: 440, height: 956, ratio: 3, models: "iPhone 16 Pro Max, 17 Pro Max" },
  { width: 402, height: 874, ratio: 3, models: "iPhone 16 Pro, 17, 17 Pro" },
  { width: 430, height: 932, ratio: 3, models: "iPhone 14 Pro Max, 15 Plus, 15 Pro Max, 16 Plus" },
  { width: 393, height: 852, ratio: 3, models: "iPhone 14 Pro, 15, 15 Pro, 16" },
  { width: 428, height: 926, ratio: 3, models: "iPhone 12 Pro Max, 13 Pro Max, 14 Plus" },
  { width: 390, height: 844, ratio: 3, models: "iPhone 12, 13, 14" },
  { width: 375, height: 812, ratio: 3, models: "iPhone X, XS, 11 Pro, 12 mini, 13 mini" },
  { width: 414, height: 896, ratio: 3, models: "iPhone XS Max, 11 Pro Max" },
  { width: 414, height: 896, ratio: 2, models: "iPhone XR, 11" },
  { width: 414, height: 736, ratio: 3, models: "iPhone 8 Plus" },
  { width: 375, height: 667, ratio: 2, models: "iPhone 8, SE 2 and 3" },
  { width: 320, height: 568, ratio: 2, models: "iPhone SE 1" },
  { width: 768, height: 1024, ratio: 2, models: "iPad mini, iPad 9.7" },
  { width: 834, height: 1194, ratio: 2, models: "iPad Pro 11" },
  { width: 1024, height: 1366, ratio: 2, models: "iPad Pro 12.9" },
];

export const startupImages = devices.map(({ width, height, ratio }) => ({
  url: `/splash/splash-${width * ratio}x${height * ratio}.png`,
  media:
    `screen and (device-width: ${width}px) and (device-height: ${height}px) ` +
    `and (-webkit-device-pixel-ratio: ${ratio}) and (orientation: portrait)`,
}));
