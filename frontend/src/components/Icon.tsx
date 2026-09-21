import type { SVGProps } from 'react';

// Outline icon set from the Amorae design system. 24 grid, 1.6 stroke, round caps.
const PATHS: Record<string, string> = {
  home: 'M3.5 10.5 12 3.5l8.5 7V19a1.5 1.5 0 0 1-1.5 1.5h-4.5v-6h-5v6H5A1.5 1.5 0 0 1 3.5 19z',
  together: 'M4.64 16.25A8.5 8.5 0 0 1 12 3.5a3.5 3.5 0 0 1 0 7M19.36 7.75A8.5 8.5 0 0 1 12 20.5a3.5 3.5 0 0 1 0-7',
  book: 'M12 7.5C10.5 5.8 8 5 3.5 5v13c4.5 0 7 .8 8.5 2.5 1.5-1.7 4-2.5 8.5-2.5V5C16 5 13.5 5.8 12 7.5zM12 7.5v13',
  clock: 'M12 7.5V12l3 2',
  sliders: 'M4 7h9M18 7h2M4 17h2M11 17h9',
  check: 'M5 12.5l4.5 4.5L19 7.5',
  right: 'M9.5 6l6 6-6 6',
  left: 'M14.5 6l-6 6 6 6',
  down: 'M12 5v14M6 13l6 6 6-6',
  plus: 'M12 5v14M5 12h14',
  grip: 'M9 6.5h.01M15 6.5h.01M9 12h.01M15 12h.01M9 17.5h.01M15 17.5h.01',
  bell: 'M6 9.5a6 6 0 0 1 12 0c0 5.5 2 7 2 7H4s2-1.5 2-7zM10 19.5a2 2 0 0 0 4 0',
  share: 'M12 3.5v11M8 7.5l4-4 4 4M7.5 11H6a1.5 1.5 0 0 0-1.5 1.5v6A1.5 1.5 0 0 0 6 20h12a1.5 1.5 0 0 0 1.5-1.5v-6A1.5 1.5 0 0 0 18 11h-1.5',
  addsq: 'M12 8.5v7M8.5 12h7',
  offline: 'M4 4l16 16M8.5 16a5 5 0 0 1 5.5-1M5.5 12.5a9 9 0 0 1 4-2.3M14 10.1a9 9 0 0 1 4.5 2.4M2.5 9a13.5 13.5 0 0 1 3.7-2.5M10.5 5.1A13.5 13.5 0 0 1 21.5 9M12 19.5h.01',
  sync: 'M19.5 11a7.5 7.5 0 0 0-13.7-3.5M4.5 4.5v3.5H8M4.5 13a7.5 7.5 0 0 0 13.7 3.5M19.5 19.5V16H16',
  copy: 'M15 5.5V5A1.5 1.5 0 0 0 13.5 3.5H5A1.5 1.5 0 0 0 3.5 5v8.5A1.5 1.5 0 0 0 5 15h.5',
  x: 'M6.5 6.5l11 11M17.5 6.5l-11 11',
  user: 'M4.5 20a7.5 7.5 0 0 1 15 0',
  users: 'M2.5 19.5a6.5 6.5 0 0 1 13 0M17.5 15.2a4.5 4.5 0 0 1 4 4.3',
  globe: 'M3.5 12h17M12 3.5c3 3 3 14 0 17M12 3.5c-3 3-3 14 0 17',
  lock: 'M8.5 10.5V8a3.5 3.5 0 0 1 7 0v2.5',
  logout: 'M9.5 4H6.5A1.5 1.5 0 0 0 5 5.5v13A1.5 1.5 0 0 0 6.5 20h3M15 8l4 4-4 4M19 12H10',
  trash: 'M4.5 7h15M9.5 7V4.5h5V7M6.5 7l.8 12.5h9.4L17.5 7',
  pencil: 'M4.5 19.5l1-4L16 5l3 3L8.5 18.5z',
  moon: 'M19.5 14.5A8 8 0 1 1 9.5 4.5a6.5 6.5 0 0 0 10 10z',
  alert: 'M12 8v4.5M12 16h.01',
  phone: 'M11 18h2',
  calendar: 'M4 10h16M8.5 3.5v3M15.5 3.5v3',
  pin: 'M12 21s6.5-5.6 6.5-10.5a6.5 6.5 0 0 0-13 0C5.5 15.4 12 21 12 21z',
  target: 'M12 12h.01',
  flag: 'M6 21V4M6 4.5h11.5l-2.5 4 2.5 4H6',
  image: 'M4 16l4.5-4.5 4 4 2.5-2.5 5 5',
  note: 'M5 4.5h14A1.5 1.5 0 0 1 20.5 6v9a1.5 1.5 0 0 1-1.5 1.5h-8L7 20v-3.5H5A1.5 1.5 0 0 1 3.5 15V6A1.5 1.5 0 0 1 5 4.5z',
  bookmark: 'M7 4h10v16.5l-5-4-5 4z',
};

// Extra primitives (circles and rects) some icons need.
const EXTRA: Record<string, React.ReactNode> = {
  clock: <circle cx="12" cy="12" r="8.5" />,
  sliders: (<><circle cx="15.5" cy="7" r="2.5" /><circle cx="8.5" cy="17" r="2.5" /></>),
  addsq: <rect x="4" y="4" width="16" height="16" rx="4" />,
  copy: <rect x="9" y="9" width="11" height="11" rx="2.5" />,
  user: <circle cx="12" cy="8.5" r="3.75" />,
  users: (<><circle cx="9" cy="9" r="3.25" /><circle cx="17" cy="10" r="2.75" /></>),
  globe: <circle cx="12" cy="12" r="8.5" />,
  lock: <rect x="5" y="10.5" width="14" height="10" rx="2.5" />,
  alert: <circle cx="12" cy="12" r="8.5" />,
  phone: <rect x="7" y="3" width="10" height="18" rx="2.5" />,
  calendar: <rect x="4" y="5" width="16" height="15" rx="2.5" />,
  pin: <circle cx="12" cy="10.5" r="2.25" />,
  target: (<><circle cx="12" cy="12" r="8.5" /><circle cx="12" cy="12" r="4.5" /></>),
  image: (<><rect x="4" y="5" width="16" height="14" rx="2.5" /><circle cx="15" cy="9.5" r="1.25" /></>),
};

export type IconName = keyof typeof PATHS;

export function Icon({ name, size = 24, strokeWidth = 1.6, className, ...rest }: { name: IconName; size?: number; strokeWidth?: number } & SVGProps<SVGSVGElement>) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden="true" fill="none" stroke="currentColor" strokeWidth={strokeWidth} strokeLinecap="round" strokeLinejoin="round" className={`shrink-0 ${className ?? ''}`} {...rest}>
      {EXTRA[name]}
      <path d={PATHS[name]} />
    </svg>
  );
}

/** The Amorae mark: two paths around one shared centre. Stroke gets heavier as it gets smaller. */
export function Mark({ size = 40, className }: { size?: number; className?: string }) {
  const sw = size >= 96 ? 1.5 : size >= 48 ? 1.7 : size >= 32 ? 2 : size >= 24 ? 2.2 : 2.6;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden="true" fill="none" stroke="currentColor" strokeWidth={sw} strokeLinecap="round" className={`shrink-0 ${className ?? ''}`}>
      <path d={PATHS.together} />
    </svg>
  );
}
