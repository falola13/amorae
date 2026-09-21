import type { Metadata, Viewport } from 'next';
import '@fontsource-variable/manrope';
import './globals.css';
import { Providers } from '@/components/Providers';
import { SPLASH_LINKS } from '@/lib/splash';

export const metadata: Metadata = {
  title: 'Amorae',
  description: 'A private space for the life you are building together.',
  applicationName: 'Amorae',
  manifest: '/manifest.webmanifest',
  appleWebApp: { capable: true, title: 'Amorae', statusBarStyle: 'default' },
  icons: {
    icon: [{ url: '/icons/favicon.ico', sizes: '48x48' }, { url: '/icons/favicon.svg', type: 'image/svg+xml' }],
    apple: [{ url: '/icons/apple-touch-icon.png' }, { url: '/icons/apple-touch-icon-167.png', sizes: '167x167' }, { url: '/icons/apple-touch-icon-152.png', sizes: '152x152' }, { url: '/icons/apple-touch-icon-120.png', sizes: '120x120' }],
    other: [{ rel: 'mask-icon', url: '/icons/safari-pinned-tab.svg', color: '#5B2A4A' }],
  },
};

export const viewport: Viewport = { themeColor: '#F7F4EF', width: 'device-width', initialScale: 1, viewportFit: 'cover' };

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <head>
        {SPLASH_LINKS.map((l) => <link key={l.href} rel="apple-touch-startup-image" media={l.media} href={l.href} />)}
      </head>
      <body>
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
