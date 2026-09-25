// Which deploy is serving right now. The value is baked into this build (see
// next.config.ts), so an app loaded from an older deploy gets a different
// answer here — that difference is the whole signal (lib/pwa/update.ts).
export const dynamic = "force-dynamic";

export function GET() {
  return Response.json(
    { build: process.env.NEXT_PUBLIC_BUILD_ID ?? null },
    { headers: { "Cache-Control": "no-store" } },
  );
}
