# Deploying Amorae for nothing

Written for the actual situation: two people, no revenue, and a hard budget of
zero. Everything below is chosen for that, and where zero costs you something
other than money it says so.

Prices and free-tier limits were checked on 2026-09-24 and drift constantly.
Check them again before committing to one.

## What has to run

Five things, and the fifth is what makes this harder than it looks.

| | What | Needs |
|---|---|---|
| 1 | **Postgres** | To survive restarts |
| 2 | **`/app/migrate`** | To run once, before the API, on each deploy |
| 3 | **`/app/api`** | To answer HTTP |
| 4 | **web** (Next.js) | To answer HTTP, server-side |
| 5 | **`/app/worker`** | **To tick every five minutes, forever** |

Free hosting is built around request and response. Most free tiers sleep a
service after some minutes of inactivity, and a sleeping worker sends nothing
— which is why "why did no notification arrive" is the question this document
exists to prevent. Every plan below is judged first on what it does about (5).

The good news: `Worker.Tick` is one idempotent pass. Every send is claimed by a
row before it goes out, so running it late, twice, or from two places at once
still sends each notification exactly once. That allows more freedom about
*how* it gets called than a stateful scheduler would.

> **Chosen for this deployment: [Option C](#option-c--split-across-managed-free-tiers-chosen).**
> A and B are kept below because they are better answers if the friction each
> describes turns out not to apply to you — and because C's costs (a cold
> start, a reminder up to five minutes late) are real ones you may tire of.

## Option A — one Always Free VM

Oracle Cloud's Always Free tier includes ARM (Ampere) instances that do not
expire and are not a trial. One of them runs this repository as it stands:

```
git clone … && cd amorae
cp .env.example .env     # fill in the secrets below
docker compose up -d --build
```

That is the whole deployment. `docker-compose.yml` already defines Postgres,
the one-shot migration, the API, the web app **and the worker** — so (5) stops
being a problem by being an ordinary long-running container.

**What it costs instead of money**

- Oracle asks for a card to verify the account. It is not charged on Always
  Free, but Nigerian cards are refused often enough that it is worth trying
  before planning around it.
- ARM capacity is frequently unavailable in popular regions. "Out of host
  capacity" on creation is common and can last days. The AMD micro instances
  are almost always available but have 1 GB of RAM, which is tight for
  Postgres plus three services — add swap, or split across the two micro
  instances you are allowed.
- You are the sysadmin: updates, TLS, backups. The next sections are that
  work.

**TLS is not optional** — service workers and push require HTTPS. Cheapest
path is Cloudflare in front: point the domain's nameservers at Cloudflare
(free), proxy the record, and it terminates TLS. Otherwise put Caddy on the
box and let it fetch a Let's Encrypt certificate.

**Before it is reachable from outside**, change the parts of
`docker-compose.yml` that were written for a laptop:

- `x-bff-secret` is a dev value in the file. Replace it with
  `openssl rand -hex 32`, supplied from the environment.
- Postgres has the password `amorae`. Change it. Keep its port bound to
  `127.0.0.1` — it should be reachable only by the other containers.
- `COOKIE_SECURE: "true"` on web, once TLS is in front.

## Option B — a machine you already own, plus a Cloudflare Tunnel

If there is any machine at home that can stay on — an old laptop, a Pi — this
is genuinely zero, with no signup, no card and no capacity lottery.

The same `docker compose up -d`, then a Cloudflare Tunnel (free) publishes it
at a real HTTPS hostname with no public IP, no static address and no port
opened on the router.

For two users this is a serious answer, not a hack. What you trade is uptime:
the app is up while that machine and your home internet are up. Given who is
using it, that may be perfectly fine.

## Option C — split across managed free tiers (chosen)

The most moving parts of the three, and the one that needed a code change.
That change is now made: `POST /internal/tick` runs one pass of the worker on
request, so nothing here has to stay awake.

| | Where | Note |
|---|---|---|
| web | **Vercel Hobby** | Free for personal projects, does not sleep. Set the project's root directory to `apps/web` |
| Postgres | **Neon** or **Supabase** free tier | `DATABASE_URL` needs `?sslmode=require` |
| API | a free container host | `apps/api/Dockerfile` builds it. These sleep on inactivity; the cron below keeps it awake |
| worker | **nothing** | It is the cron, calling the API |

### The order to do it in

1. **Postgres first**, because everything else needs its URL. Create the
   database, copy the connection string.

2. **Migrate.** Nothing runs migrations for you here — that is `migrate`'s own
   container in compose, which this option does not use. From your laptop:

   ```
   cd apps/api && DATABASE_URL='postgres://…?sslmode=require' go run ./cmd/migrate up
   ```

   Repeat this on any deploy that adds a migration, *before* the new API goes
   live.

3. **The API.** Deploy `apps/api/Dockerfile`, command `/app/api`. It needs
   every secret in the table below, `APP_ENV=production`, and `TICK_SECRET`.
   Note the public URL it gets.

4. **The web app** on Vercel, root directory `apps/web`. `API_URL` is that
   public API URL. `BFF_SECRET` must be character-for-character what the API
   has. `COOKIE_SECURE=true`. And `NEXT_PUBLIC_VAPID_PUBLIC_KEY`, which Vercel
   supplies at build time — which is exactly when it is needed.

5. **The cron.** cron-job.org, free, every five minutes:

   ```
   POST https://your-api-host/internal/tick
   Authorization: Bearer <TICK_SECRET>
   ```

   A GitHub Actions schedule works too if the repository is public. On a
   private one the monthly Actions minutes will not cover a five-minute cron,
   and GitHub's scheduler is unreliable to the minute anyway — which does not
   matter here, since a late tick sends the same notifications.

### What to expect from it

- **The first request after a quiet spell is slow.** A free API host sleeps;
  the five-minute cron mostly prevents that, but a cold start of thirty
  seconds or so is the price of the tier.
- **A reminder can be up to five minutes late.** "Ten minutes before" may
  arrive six minutes before. That is inherent to driving the worker from
  outside and is why the grace windows exist.
- **`{"status":"already running"}`** in the cron's log is not an error. A pass
  overran its five minutes and the next one declined to pile on.
- **Watch the cron's own log** for a week. It is the only thing that will tell
  you the worker has stopped, and silence from a notification system looks
  exactly like having nothing to say.

## The domain

**You do not need one.** Every option above gives an HTTPS hostname for free —
the host's own subdomain, or the tunnel's. Push, service workers and
installing the PWA all work on those. Buy a domain when you want it to look
like something, not to make it work.

When you do, cheapest in naira, from WhoGoHost with the currency set to NGN
(checked 2026-09-24):

| | Per year |
|---|---|
| **`.com.ng`** | **₦3,000** |
| `.site` | ₦4,500 |
| `.online` | ₦5,000 |
| `.ng` | ₦9,200 |
| `.com` | ₦14,999 |

`.com.ng` is the obvious one: a fifth of a `.com`, and it reads as Nigerian
rather than as a compromise. Qservers and Truehost are the other local
registrars worth a price check.

Whatever you buy, put it behind Cloudflare's free plan for TLS and DNS.

## Secrets it will not start without

`internal/config` refuses to boot in production without these, deliberately —
a missing key should stop a deploy rather than surface later as silence.

| | Where from |
|---|---|
| `DATABASE_URL` | Your Postgres |
| `APP_URL` | `https://…`, the real one. It signs the links in emails |
| `BFF_SECRET` | `openssl rand -hex 32`, the same value for api and web |
| `VAPID_PUBLIC_KEY` / `VAPID_PRIVATE_KEY` / `VAPID_SUBJECT` | You have these. The subject is a `mailto:` or `https:` URL |
| `RESEND_API_KEY` / `MAIL_FROM` | Resend's free tier. `MAIL_FROM` needs a domain verified with them |

**The one that will not shout at you.** `NEXT_PUBLIC_VAPID_PUBLIC_KEY` is
inlined into the browser bundle when the web image is *built*, not read when
it runs — so setting it only on the running container is too late and it is
simply absent. The app then asks for notification permission, is granted it,
and never subscribes anybody. `docker-compose.yml` now passes it as a build
argument from `VAPID_PUBLIC_KEY`, so exporting that before
`docker compose build` handles it. Anywhere else, remember it.

## Backups

The point of this app is that it is a record. A free VM that dies takes it
with it.

```
docker compose exec -T db pg_dump -U amorae amorae | gzip > amorae-$(date +%F).sql.gz
```

Nightly via cron, and off the machine — Backblaze B2's free tier, or a copy
pulled to your laptop weekly. Untested backups are not backups: restore one
into a throwaway container once, so you know the command works before you need
it in a hurry.

## What will eventually cost money

Not soon, and not for two people, but so it is not a surprise:

- **Photos** (FR-MEM-003) need object storage. Backblaze B2 and Cloudflare R2
  both have free allowances two people will not exhaust for years.
- **Email** beyond Resend's free allowance, which is far past what two
  accounts send.
- **Postgres beyond a free tier**, on Option C. On A or B it is a file on your
  own disk.

If zero ever costs more in evenings than it saves in money, the honest
alternative is a small VPS. Hetzner's cheapest is a few euros a month and
removes the capacity lottery, the ARM availability and the tunnel in one go.
