# Phase 1: step-by-step guide

Phase 1 (Foundation) is done when every **Must** requirement in `AUTH`, `ACCT`, `PAIR` and the
`PWA` shell is **Implemented**, the Phase 1 open questions are answered, and the NFR-SEC and
NFR-OBS baseline checks pass
([01 §8](./requirements/01-product-requirements.md)).

Work top to bottom. Each step says what to change, where, and how you know it is done. Tick the
box only when the "Done when" line is true.

---

## How to finish any one requirement

A requirement is **Implemented** only when all five hold
([requirements README §4](./requirements/README.md)):

1. The Go endpoint and the web client both exist and talk to each other.
2. Every acceptance criterion (`.AC1`, `.AC2`, …) has an automated test, or a written manual check.
3. `[docs/API.md](./API.md)` and `apps/web/src/lib/api/types.ts` describe the same contract.
4. The relevant NFRs have been checked.
5. The requirement's status and the traceability matrix in
  [02](./requirements/02-functional-requirements.md) are updated in the same change.

Use the same loop every time:

1. Read the requirement and its acceptance criteria in 02.
2. Build: migration → entity/rules → repository → service → handler → wire in `internal/app`.
3. Move the endpoint in `docs/API.md` from "Planned endpoints" to the built section.
4. Wire the web client (`features/<module>/api.ts`, `hooks.ts`, the screen).
5. Run the checks below, then click through it in the browser with two accounts.
6. Update the status in 02.

Where each layer lives (`<module>` is `auth`, `user` or `couples`). Every checkbox below starts
with the file it changes, so you never have to guess.


| Layer                                | File                                                                 |
| ------------------------------------ | -------------------------------------------------------------------- |
| Migration                            | `apps/api/migrations/000NN_<name>.sql` (next free number: `00007`)   |
| Entity, errors, rules                | `apps/api/internal/modules/<module>/<module>.go`                     |
| Repository interface (what it needs) | `apps/api/internal/modules/<module>/service.go`, top of the file     |
| SQL                                  | `apps/api/internal/modules/<module>/repository_postgres.go`          |
| Use case                             | `apps/api/internal/modules/<module>/service.go`                      |
| Route + request decoding             | `apps/api/internal/modules/<module>/handler.go`                      |
| Response shape                       | `apps/api/internal/modules/<module>/dto.go` (couples), `user/dto.go` |
| Wiring (limiters, mailer, config)    | `apps/api/internal/app/app.go`, `apps/api/internal/config/config.go` |
| Web API call                         | `apps/web/src/features/<feature>/api.ts`                             |
| Web hook (TanStack Query)            | `apps/web/src/features/<feature>/hooks.ts`                           |
| Web form rules (zod)                 | `apps/web/src/lib/api/schemas.ts`                                    |
| Web response types                   | `apps/web/src/lib/api/types.ts`                                      |
| Web sheet / form component           | `apps/web/src/features/<feature>/components/*.tsx`                   |
| Web page                             | `apps/web/src/app/(group)/<route>/page.tsx`                          |
| URLs, public (signed-out) pages      | `apps/web/src/lib/routes.ts`, `apps/web/src/proxy.ts` (`PUBLIC`)     |
| Contract doc                         | `docs/API.md`                                                        |


Account, email, password and couple screens use the `couple` web feature
(`apps/web/src/features/couple/`); sign-in and sign-up use `features/auth/`.

Checks to run before calling anything done:

```powershell
npm run db
npm run migrate
$env:AMORAE_TEST_DATABASE_URL = "postgres://amorae:amorae@localhost:5434/amorae?sslmode=disable"
npm run test:api
npm run check
```

Without `AMORAE_TEST_DATABASE_URL`, every repository test is skipped and still reports `ok`. A
green run without it proves nothing about SQL.

---

## Where you are today


| Area | Implemented                                                                      | Not done                                                                                                            |
| ---- | -------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| AUTH | Register, login, logout, 30-day session, password rule, login rate limits        | Password reset, age check, recorded Terms acceptance, faith consent. Idle expiry and email verification are Should  |
| ACCT | View/edit profile, change email, sign-out clears local state                     | Change password, deletion with password, data export. Two-step email change and "sign out other devices" are Should |
| PAIR | Go routes for create, join, get, edit couple, role, onboarding                   | The gaps in Step 2, invite regeneration and limits, leave couple                                                    |
| PWA  | Install, iOS guide, offline page, offline banner, offline queue, clear on logout | Idempotency keys (a Must, but only needed once a queued create reaches Go), update prompt (Should)                  |


---

## Step 1 — Make what exists correct

These are bugs in code that is already written. Fix them before adding anything.

### 1.1 Account deletion: web and API disagree

- [x] **The API requires a body the web never sends.** `deleteMe` in
  `apps/api/internal/modules/user/handler.go` decodes `{ "confirm": "delete" }` and rejects a
  missing body as `invalid_json`. The web call in `apps/web/src/features/couple/api.ts` is
  `http.delete("/users/me")` with no body, so **deleting from Settings always fails today.**
  Step 5 replaces this body anyway; until then send `{ confirm: "delete" }`.
- [x] **A 204 must not have a body.** The handler ends with
  `httpx.Data(w, 204, "User deleted successfully")`. Use `httpx.NoContent(w)`, the same call
  `auth` logout uses.

**Done when:** Delete account in Settings removes the account and you land on the sign-in screen.

### 1.2 Apply the delete-cascade migration

- [x] Run `npm run migrate`, then `npm run migrate:status` and confirm `00005_delete_cascade` is
  applied.
- [x] Run the two `TestPostgresRepository_DeleteMe_*` tests with the database URL set. They were
  written but have never run against Postgres.

**Done when:** both tests pass with `AMORAE_TEST_DATABASE_URL` set.

---

## Step 2 — Finish pairing (FR-PAIR-001 to 007)

Every route is registered in `apps/api/internal/modules/couples/handler.go`, but three behaviours
are wrong.

### 2.1 Creating a second couple returns a 500 (FR-PAIR-001.AC2)

`PostgresRepository.Create` inserts into `couple_members` and returns the raw error. For a user
already in a couple, Postgres raises a unique violation on `couple_members_user_id_key`, and the
API answers 500 instead of 409.

- [x] `couples/repository_postgres.go`, `Create`: return `translateMemberWriteErr(err)` for the member insert, as `Join` does.
  Then the caller gets 409 `already_paired`. The transaction rolls back the couple row, so no
  orphan is left.

### 2.2 The waiting screen can show a dead code (FR-PAIR-002)

`GetForUser` loads the newest invite for the couple whatever its state, so after a partner joins,
the API still returns the used code as `invite_code`.

- [x] `couples/repository_postgres.go`, `GetForUser`: add an `at time.Time` parameter. Replace the
  invite query near the end of the function with:

```sql
SELECT code FROM couple_invitations
WHERE couple_id = $1 AND status = 'pending' AND expires_at > $2
ORDER BY created_at DESC LIMIT 1
```

  and only run it when `len(members) < 2`, so a full couple never gets a code.

- [x] `couples/service.go`: add `at time.Time` to `GetForUser` in the `Repository` interface, and
  make `GetMine` call `s.repo.GetForUser(ctx, userID, s.now())`. Use the service clock, not `now()`
  in SQL, so tests can move time.
- [x] Nothing changes on the web: `app/(onboarding)/invite/page.tsx` already shows `•••-•••` when
  `invite_code` is empty.

### 2.3 Join errors don't match the contract (FR-PAIR-003.AC2)

`docs/API.md` promises `400 validation_failed` with `fields.code` for a bad code. The code returns
`invite_invalid`, `invite_expired`, `invite_used` and `invite_revoked` as plain errors with no
`fields`, so the join form can't show the message under the input.

Recommended shape: keep the four specific codes (the UI can say "expired" vs "already used"), and
also put the message in `fields.code` so the form shows it under the input.

- [x] `couples/couples.go`, the `var (...)` block at the top: build `ErrInviteInvalid`,
  `ErrInviteExpired`, `ErrInviteUsed` and `ErrInviteRevoked` with the `inviteError` helper, which
  sets `Fields: {"code": message}` on top of `apperr.Invalid(...)`.
- [x] `couples/couples.go`: leave `ErrCoupleFull` as 409 `couple_full` (AC3).
- [x] `docs/API.md`: update the `POST /v1/couples/join` row to list the four codes and
  `fields.code`.
- [x] Nothing changes on the web: `app/(onboarding)/join/page.tsx` already shows
  `e.fields?.code ?? e.message` under the input.

### 2.4 Web client

`apps/web/src/features/couple/api.ts` has `couple`, `create`, `join` and `onboarding`, but nothing
for editing the couple or the role.

**Edit the couple and your role** (FR-PAIR-005, FR-PAIR-006 stay Partial until a screen calls them):

- [x] `features/couple/api.ts`: add
  `updateCouple: (patch: { name?: string; relationship_start_date?: string }) => http.patch<Couple>("/couples/me", patch)`
  and `updateRole: (role: string) => http.patch<Couple>("/couples/role", { role })`, each ending
  `.then((r) => r.data)` like `onboarding`.
- [x] `features/couple/hooks.ts`: add `useUpdateCouple` and `useUpdateRole`. On success, write the
  returned couple into the cache with `qc.setQueryData(keys.couple, couple)`, and set
  `meta: { handlesError: true }` so field errors show on the form, not as a toast.
- [x] `lib/api/schemas.ts`: add `coupleSchema` (name, optional date) and `roleSchema` (1–32
  characters, the same rule as `ValidateRole` in `couples/couples.go`).
- [x] `features/couple/components/edit-couple-sheet.tsx`: new sheet, built like
  `change-email-sheet.tsx`, with name, start date and your role.
- [x] `app/(app)/settings/profile/page.tsx`: open the sheet from the couple row, the same way
  it opens `ChangeEmailSheet`.

**Invite link:**

- [x] `app/(onboarding)/invite/page.tsx`: next to `ABC-123`, offer a share/copy link built with
  `window.location.origin + routes.join({ code })`. `routes.join` already takes a `code`.
- [x] `app/(onboarding)/join/page.tsx`: it does **not** read the URL today. Read it with
  `useSearchParams().get("code")` and use `formatInviteCode(...)` of it as the form's
  `defaultValues.code`. `useSearchParams` needs a `<Suspense>` boundary around the component in the
  App Router.

**Contract:**

- [x] `lib/api/types.ts`, the `Couple` type: compare field by field with `MineDTO` in
  `couples/dto.go` (`id`, `name`, `me`, `partner`, `invite_code`, `started_on`, `onboarding`) and
  fix any difference.
- [x] `docs/API.md`: move the six couples endpoints out of "Planned endpoints".

**Done when:** in two browsers (or one normal window and one private window), account A creates a
couple and sees a code, account B joins with it, both see each other's name, the code disappears
from A's screen, and a third account gets "couple full".

---

## Step 3 — Invite lifetime and guess limits (FR-PAIR-004, Q-08)

- [ ] `docs/requirements/05-decisions-and-open-questions.md`: add a `DEC` accepting Q-08's
  recommendation (7 days, one active code per couple, regenerating invalidates the old one, join
  attempts limited per user and per IP), and set Q-08 to **Resolved → DEC-NN**.

**Expiry** (AC1) is already enforced in `couples/repository_postgres.go`, `Join`
(`!at.Before(expiresAt)`). Nothing to build.

**Regenerate the code** with `POST /couples/invite`:

- [x] `couples/service.go`, `Repository` interface: add
  `ReplaceInvite(ctx context.Context, userID uuid.UUID, code string, expiresAt, at time.Time) error`.
- [x] `couples/repository_postgres.go`: implement `ReplaceInvite` inside `r.db.InTx`:
  1. find the caller's couple and lock it (`SELECT ... FOR UPDATE`, as `Join` does)
  2. count members; 2 or more → `ErrCoupleFull`
  3. `UPDATE couple_invitations SET status = 'revoked' WHERE couple_id = $1 AND status = 'pending'`
  4. insert the new pending invite
  5. a `23505` on `couple_invitations_code_key` (the `UNIQUE` on `code` in `00001_init.sql`)
    means the code is taken: return a sentinel such as `errCodeTaken`
- [x] `couples/service.go`: add `RegenerateInvite(ctx, userID) (Mine, error)`. Generate a code
  with `s.newInviteCode()`, call `ReplaceInvite` with `s.now().Add(inviteTTL)`, retry up to three
  times on `errCodeTaken`, then return `s.GetMine(ctx, userID)`.
- [x] `couples/handler.go`: add `"POST /couples/invite"` to `service`, to `RegisterRoutes` with
  `HandleAuthed`, and a `regenerateInvite` method that answers like `join`.

**Guess limits** (AC2), both answering 429 with `Retry-After`:

- [x] `couples/service.go`: declare `type AttemptLimiter interface { Allow(key string) (bool,
  time.Duration) }`(the same shape as auth's), add it to`Service`and`NewService`, and at the top of` Join`return`apperr.RateLimited(retryAfter)`when`Allow("join:" + userID.String())` fails.
- [x] `internal/app/app.go`: build `joinAttempts := ratelimit.New(10, 15*time.Minute, time.Now)`
  next to `loginAttempts` and pass it to `couples.NewService`.
- [x] `internal/app/app.go`: per-IP limit. Change `couplesHandler.RegisterRoutes(v1)` to
  `couplesHandler.RegisterRoutes(v1.With(middleware.RateLimit(coupleRequests, "couples")))` with
  `coupleRequests := ratelimit.New(60, time.Minute, time.Now)`. This covers every couples route,
  which is fine at that rate.

**Web:**

- [x] `features/couple/api.ts`: `regenerateInvite: () => http.post<Couple>("/couples/invite", {})`.
- [x] `features/couple/hooks.ts`: `useRegenerateInvite`, writing the result into `keys.couple`.
- [x] `app/(onboarding)/invite/page.tsx`: a "New code" button under the code.
- [x] `docs/API.md`: add `POST /v1/couples/invite`.

**Done when:** an old code stops working the moment a new one is issued, and the 11th wrong guess
in 15 minutes gets 429.

---

## Step 4 — Change password (FR-ACCT-004)

Nothing blocks this. It is `ChangeEmail` again, plus signing out other devices, which already
exists as `SignOutOtherSessions`.

API (all paths under `apps/api/internal/modules/`):

- [x] `user/repository_postgres.go`: add `UpdatePasswordHash(ctx, id uuid.UUID, hash string, at
  time.Time) error`,` UPDATE users SET password_hash = $2, updated_at = $3 WHERE id = $1`, returning` ErrNotFound`on zero rows (copy`UpdateEmail`).
- [x] `auth/service.go`, `UserRepository` interface: add `UpdatePasswordHash`. The session side
  needs nothing new: `SessionRepository.DeleteOthers` already deletes every session but one.
- [x] `auth/service_test.go`: add `UpdatePasswordHash` to the fake user repository, or the package
  stops compiling.
- [x] `auth/service.go`: add `ChangePasswordInput{ CurrentPassword, NewPassword string }` and
  `ChangePassword(ctx, userID, currentToken string, input) error`, below `ChangeEmail`:
  1. `s.attempts.Allow("reauth:" + userID.String())`, else `apperr.RateLimited`
  2. `passwordProblem(input.NewPassword)` → 400 `fields.new_password`
  3. empty current password → 400 `fields.current_password`
  4. `s.users.GetByID`, then `s.hasher.Compare`; on failure return `errWrongCurrentPassword`
  5. `s.hasher.Hash(input.NewPassword)`, then `s.users.UpdatePasswordHash(..., s.now())`
  6. `s.sessions.DeleteOthers(ctx, userID, hashToken(currentToken))` (AC1)
- [x] `auth/handler.go`: add `ChangePassword` to the `service` interface; register
  `r.HandleAuthed("PUT /users/me/password", ...)` next to `PUT /users/me/email`; the method decodes
  `{ current_password, new_password }`, reads the caller's token with `bearerToken(r)` (as `logout`
  does), and answers `httpx.NoContent(w)`.
- [x] `docs/API.md`: add `PUT /v1/users/me/password`.

Web (all paths under `apps/web/src/`):

- [x] `lib/api/schemas.ts`: `changePasswordSchema` with `current_password` (required) and
  `new_password` using the existing password rule, plus `ChangePasswordInput`.
- [x] `features/couple/api.ts`: `changePassword: (input: ChangePasswordInput) =>
  http.put("/users/me/password", input).then(() => undefined)`.
- [x] `features/couple/hooks.ts`: `useChangePassword` with `meta: { handlesError: true }`.
- [x] `features/couple/components/change-password-sheet.tsx`: copy `change-email-sheet.tsx`; map
  `fields.current_password` and `fields.new_password` onto their inputs.
- [x] `app/(app)/settings/profile/page.tsx`: a "Change password" row that opens the sheet, next to
  where `ChangeEmailSheet` is opened.

**Done when:** two logged-in devices, change the password on one, the other is signed out on its
next request.

---

## Step 5 — Account deletion with password (FR-ACCT-005)

Step 1 made deletion work; this makes it meet the requirement. AC2 says deletion needs the
current password, and only `auth` has the password hasher, so move the check there.

The body becomes `{ "confirm": "delete", "current_password": "…" }`. Keep the typed word; the
password is the real check.

API (all paths under `apps/api/internal/modules/`):

- [x] `auth/service.go`, `UserRepository` interface: add `DeleteMe(ctx, id uuid.UUID) error`.
  `user.PostgresRepository` already implements it, so `app.go` needs no change.
- [x] `auth/service_test.go`: add `DeleteMe` to the fake user repository.
- [x] `auth/service.go`: add `DeleteAccount(ctx, userID, currentPassword string) error`: the
  `"reauth:"` limit, empty password → 400 `fields.current_password`, `GetByID` + `Compare` →
  `errWrongCurrentPassword`, then `s.users.DeleteMe`. Sessions go with the user through the
  cascade.
- [x] `auth/handler.go`: add `DeleteAccount` to `service`; register
  `r.HandleAuthed("DELETE /users/me", ...)`. Move `isDeleteConfirmation` and
  `deleteConfirmation` here from `user/handler.go`, check the word first, then call the service and
  answer `httpx.NoContent(w)`.
- [x] `user/handler.go`: remove the `DELETE /users/me` route, `deleteMe`, and
  `DeleteProfile` from its `service` interface. Two routes with the same pattern panic at startup.
- [x] `user/handler_test.go`: move the `TestHandler_DeleteMe_*` cases to `auth/handler_test.go`
  or delete them, since the method they call is gone.
- [x] `user/service.go`: remove `DeleteProfile` once nothing calls it.

Web (all paths under `apps/web/src/`):

- [x] `lib/api/schemas.ts`: add `current_password: z.string().min(1, "Enter your current
  password.")`to`deleteAccountSchema`.
- [x] `features/couple/components/delete-account-sheet.tsx`: add a password `Field`
  (`type="password"`, `autoComplete="current-password"`), and map `fields.current_password` onto it
  in `onError` the way `fields.confirm` is mapped.
- [x] `app/(app)/settings/page.tsx`: nothing to change; `onDeleted={logout}` already clears local
  state and signs out.

**Done when:** the account is gone only after the correct password, and a partner who remains
keeps their couple.

---

## Step 6 — Answer the Phase 1 questions

Phase 1 cannot exit until each of these is **Resolved → DEC-NN** in
[05](./requirements/05-decisions-and-open-questions.md). For each: add a `DEC` row with the
decision and the reason, set the question's status, and add a line to 05's change log. Most have a
recommendation already; accepting it is usually enough.


| Question | Decide                                                                                         | What it unlocks                        |
| -------- | ---------------------------------------------------------------------------------------------- | -------------------------------------- |
| Q-03     | Idle expiry of 14 days plus the 30-day limit; password needed for password change and deletion | FR-AUTH-005 (Should) and Steps 4–5     |
| Q-04     | Email provider, sent only through a `Mailer` interface                                         | Step 7                                 |
| Q-05     | Verify on sign-up without blocking; two-step email change                                      | FR-AUTH-009, FR-ACCT-003 (both Should) |
| Q-08     | Recorded in Step 3                                                                             | FR-PAIR-004                            |
| Q-09     | Leaving dissolves the couple, 30 days read-only plus export, then shared content is deleted    | Step 9                                 |
| Q-14     | First-party counts only: SQL and Prometheus counters, no third-party trackers                  | Step 11                                |
| Q-15     | Vitest and Testing Library for units, Playwright for end-to-end                                | Nothing to build in Phase 1            |
| Q-20     | An unticked faith-consent checkbox at sign-up, stored with time and policy version             | Step 8                                 |
| Q-21     | Light theme only for MVP                                                                       | Nothing to build                       |


Q-10 (minimum age) is a launch-gate question, but FR-AUTH-010 is a Phase 1 Must, so decide
**18** now.

---

## Step 7 — Password reset (FR-AUTH-008)

Blocked by Q-04. Both mailers already exist in `apps/api/internal/platform/mailer/`: `log.go`
(writes the mail to the log, for development) and `resend.go` (sends through Resend). What's
missing is a consumer and the choice between them.

**Mailer wiring:**

- [x] `apps/api/internal/modules/auth/service.go`: declare
  `type Mailer interface { Send(ctx context.Context, to, subject, body string) error }` next to the
  other interfaces, add it to `Service` and `NewService`.
- [x] `apps/api/internal/config/config.go`: add `AppURL` (the web origin, e.g.
  `http://localhost:3000`) so the email can contain a full link.
- [x] `apps/api/internal/app/app.go`, the `// --- Mailer ---` block: use `mailer.NewLog(log)` when
  `cfg.RESEND_API_KEY` is empty, otherwise `mailer.NewResend(...)`. Store it as the `auth.Mailer`
  interface (not `*mailer.Resend` in the `App` struct), and pass it to `auth.NewService`.

**Storage:**

- [x] `apps/api/migrations/00007_password_resets.sql` (`00006` is already
  `00006_session_device.sql`):

```sql
CREATE TABLE password_resets (
    token_hash BYTEA PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX password_resets_user_id_idx ON password_resets (user_id);
```

  Store only the SHA-256 hash of the token, exactly like sessions (`hashToken` in
  `auth/token.go`).

- [x] `apps/api/internal/modules/auth/service.go`: declare `PasswordResetRepository` with
  `Create(ctx, hash []byte, userID uuid.UUID, expiresAt time.Time) error` and
  `Consume(ctx, hash []byte, at time.Time) (uuid.UUID, error)`.
- [x] `apps/api/internal/modules/auth/repository_postgres.go`: implement it. `Consume` runs
  `SELECT user_id ... WHERE token_hash = $1 AND used_at IS NULL AND expires_at > $2 FOR UPDATE`,
  then `UPDATE ... SET used_at = $2`; no row → a sentinel the service turns into 400
  `fields.token`.
- [x] `apps/api/internal/modules/auth/repository_postgres.go` (the session repository) and the
  `SessionRepository` interface in `auth/service.go`: add `DeleteAllForUser(ctx, userID)`
  (`DELETE FROM sessions WHERE user_id = $1`). Reset signs out **every** device, including the one
  resetting. Add it to the fake in `auth/service_test.go` too.
- [x] `apps/api/internal/app/app.go`: build the reset repository and pass it, the mailer and
  `cfg.AppURL` to `auth.NewService`. The service stores the URL as a plain string; it never reads
  config itself.

**Use cases**, in `apps/api/internal/modules/auth/service.go`:

- [x] `ForgotPassword(ctx, email) error`: always return `nil` for an unknown email (AC1: never
  reveal which emails exist). For a known one: limit with `s.attempts.Allow("forgot:" + email)`,
  create a token with `s.newToken`, store its hash with a 1-hour expiry, and
  `s.mailer.Send(...)` a link to `s.appURL + "/reset?token=" + token`.
- [x] `ResetPassword(ctx, token, newPassword string) error`: inside `s.tx.InTx`: check
  `passwordProblem`, `Consume(hashToken(token), s.now())`, hash and `UpdatePasswordHash` (from
  Step 4), then `DeleteAllForUser` (AC2).

**Routes**, in `apps/api/internal/modules/auth/handler.go` (public, so `r.Handle`, not
`HandleAuthed`; the per-IP `/auth/*` limit in `app.go` already covers them):

- [x] `POST /auth/password/forgot` `{ email }` → always `httpx.NoContent(w)`.
- [x] `POST /auth/password/reset` `{ token, new_password }` → `httpx.NoContent(w)`.
- [x] `docs/API.md`: add both.

**Web** (all paths under `apps/web/src/`):

- [x] `lib/routes.ts`: add `forgot: "/forgot"`. The reset link is built by the API from `APP_URL`, so no `reset` route helper is needed.
- [x] `proxy.ts`: add `"/forgot"` and `"/reset"` to `PUBLIC`, or signed-out people get sent to
  `/login`.
- [x] `lib/api/schemas.ts`: `forgotSchema` (email) and `resetSchema` (new password, same rule).
- [x] `features/couple/api.ts`: `forgotPassword(email)` and `resetPassword(token, new_password)`, called from the browser through the BFF (`features/auth/api.ts` is server-only).
- [x] `app/(auth)/forgot/page.tsx`: email form; after submit, always show "If that email has an
  account, we sent a link".
- [x] `app/(auth)/reset/page.tsx`: reads `token` from `searchParams` (as `login/page.tsx` does),
  new-password form, then links to `routes.login()`.
- [x] `features/auth/components/login-form.tsx`: a "Forgot password?" link to `routes.forgot`.

**Done when:** in development, the reset link from the API log sets a new password and signs out
every device.

---

## Step 8 — What sign-up must record (FR-AUTH-010, 011, 012)

All three change `POST /auth/register`, so build them together.

- [x] `apps/api/migrations/00008_consents.sql`: one row per event, never overwritten:

```sql
CREATE TABLE user_consents (
    id             UUID PRIMARY KEY,
    user_id        UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind           TEXT NOT NULL,  -- 'terms', 'privacy', 'faith_content', 'age_18'
    policy_version TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX user_consents_user_id_idx ON user_consents (user_id);
```

API:

- [x] `apps/api/internal/config/config.go`: add `TermsVersion`, `PrivacyVersion` and
  `FaithPolicyVersion`. The versions come from config, never the request, so a client can't claim
  one.
- [x] `apps/api/internal/modules/auth/service.go`: declare `ConsentRepository` with
  `Record(ctx, userID uuid.UUID, kind, version string, at time.Time) error`, and add it plus the
  versions to `Service` and `NewService`.
- [x] `apps/api/internal/modules/auth/repository_postgres.go`: implement `Record` as one `INSERT`
  that uses `r.db.Q(ctx)`, so it joins the caller's transaction.
- [x] `apps/api/internal/app/app.go`: build the consent repository and pass it and the config
  versions to `auth.NewService`.
- [x] `apps/api/internal/modules/auth/service.go`, `RegisterInput`: add `AgeConfirmed`,
  `AcceptedTerms` and `FaithConsent` (bools). In `Register`:
  - `AgeConfirmed` false → 400 `fields.age_confirmed` (FR-AUTH-010)
  - `AcceptedTerms` false → 400 `fields.accepted_terms`
  - inside the `s.tx.InTx` that already creates the user and session, record `terms`, `privacy`
  and `age_18` (FR-AUTH-011.AC2)
  - record `faith_content` only when `FaithConsent` is true (FR-AUTH-012). Accepting the Terms
  must never imply it.
- [x] `apps/api/internal/modules/auth/handler.go`, `registerRequest`: add `age_confirmed`,
  `accepted_terms` and `faith_consent`, and pass them into `RegisterInput`.
- [x] `docs/API.md`: update the `POST /v1/auth/register` body.

Web (all paths under `apps/web/src/`):

- [x] `lib/api/schemas.ts`, the register schema: `age_confirmed` and `accepted_terms` must be
  `true` (use `z.literal(true, ...)`), `faith_consent` is a plain boolean defaulting to `false`.
- [x] `features/auth/actions.ts`, `registerAction`: send the three new fields in the body.
- [x] `features/auth/components/register-form.tsx`: three checkboxes, all unticked by default,
  linking to `routes.terms` and `routes.privacy`. The age and Terms boxes block submit; the faith
  box is optional and explained in one sentence. Show `fields.age_confirmed` and
  `fields.accepted_terms` under their boxes.
- [ ] `docs/requirements/05-decisions-and-open-questions.md`, Q-20's decision: record what happens when a user has not consented to faith content and later opens Prayers (ask
  then, or block). Phase 2 needs that answer.

---

## Step 9 — Leave couple (FR-PAIR-008)

Blocked by Q-09. This is separate from deleting an account: both people keep their accounts.

- [ ] `docs/requirements/05-decisions-and-open-questions.md`, Q-09: decide **pairing again**
  first. `couple_members` has `UNIQUE (user_id)`, so while the old membership row exists, neither
  person can create or join a new couple. Either they wait 30 days, or leaving moves the old
  membership out of the way. The rest of this step depends on the answer.

API (all paths under `apps/api/`):

- [ ] `migrations/00009_couple_dissolved.sql`: `ALTER TABLE couples ADD COLUMN dissolved_at
  TIMESTAMPTZ NULL;` (Down drops it).
- [ ] `internal/modules/couples/couples.go`: add `DissolvedAt *time.Time` to `COUPLES`, and
  `ErrCoupleDissolved = apperr.Conflict("couple_dissolved", "…")` to the error block.
- [ ] `internal/modules/couples/repository_postgres.go`: select `c.dissolved_at` in `GetForUser`;
  add `Leave(ctx, coupleID uuid.UUID, at time.Time) error`, which in one `InTx` sets
  `dissolved_at` and revokes pending invites.
- [ ] `internal/modules/couples/service.go`: add `Leave` to `Repository`; add a helper
  `requireActive(mine Mine) error` returning `ErrCoupleDissolved`, and call it at the top of
  `UpdateCouples`, `UpdateRole`, `UpdateOnboarding` and `RegenerateInvite`. Later modules (prayers,
  together) call the same helper before every write. Add `Leave(ctx, userID) (Mine, error)`.
- [ ] `internal/modules/couples/handler.go`: `POST /couples/leave` `{ confirm }`, checking the
  typed word the way `DELETE /users/me` does.
- [ ] `internal/modules/couples/dto.go`: add `DissolvedAt *time.Time \`json:"dissolved_at"`
  to `MineDTO` and fill it in `ToMineDTO`.
- [ ] `scripts/purge-dissolved-couples.sql`: the hand-run purge until the worker exists (Q-16,
  Phase 2): `DELETE FROM couples WHERE dissolved_at < now() - interval '30 days';`. The cascade from
  Step 1.2 removes members and invites.
- [ ] `docs/API.md`: add `POST /v1/couples/leave` and the `dissolved_at` field.

Web (all paths under `apps/web/src/`):

- [ ] `lib/api/types.ts`: `dissolved_at: string | null` on `Couple`.
- [ ] `features/couple/api.ts` and `hooks.ts`: `leave(confirm)` and `useLeaveCouple`.
- [ ] `features/couple/components/leave-couple-sheet.tsx`: copy `delete-account-sheet.tsx`, with
  the word `leave`.
- [ ] `app/(app)/settings/page.tsx`: a "Leave couple" row that opens the sheet.
- [ ] `components/layout/app-shell.tsx`: a read-only banner when `couple.dissolved_at` is set, with
  the export link from Step 10.

**Done when:** after leaving, both partners can read the couple but not change it, and the purge
query removes it once 30 days have passed.

---

## Step 10 — Data export (FR-ACCT-006)

The file has the API's own field names:

- `user`: the profile DTO (never the password hash)
- `consents`: the user's consent rows
- `couple`: the couple, members by display name and role only (never the partner's email or
credentials, AC2), and invitations

The export reads from three modules, so it gets its own module instead of living in one of them.
Each section comes from that module's existing read methods, not one giant SQL query, so Phase 2–4
modules add their section without touching the others.

API (all paths under `apps/api/internal/`):

- [x] `modules/auth/repository_postgres.go`: add `ListConsents(ctx, userID)` to the consent
  repository from Step 8.
- [x] `modules/export/handler.go`: new package. Declare three small interfaces (a user reader, a
  couple reader, a consent reader), register `r.HandleAuthed("GET /users/me/export", ...)`, set
  `Content-Disposition: attachment; filename="amorae-export.json"`, and write the three sections
  with `httpx.Data`.
- [x] `app/app.go`: build `export.NewHandler(userSvc, couplesSvc, consentRepo)` and register it on
  `v1`.
- [x] `docs/API.md`: add `GET /v1/users/me/export`.

Web (all paths under `apps/web/src/`):

- [x] `app/(app)/settings/page.tsx`: a "Download my data" row that is a plain link to
  `/api/v1/users/me/export` with the `download` attribute. The BFF route
  `app/api/v1/[...path]/route.ts` forwards it with the session, so no `api.ts` call is needed.
- [ ] `components/layout/app-shell.tsx`: the same link in the dissolved-couple banner (Step 9).

**Done when:** the downloaded file opens as valid JSON and contains everything you can see in the app.

---

## Step 11 — Metrics and baseline checks

**Metrics (Q-14)**, counts only, no content. The `/metrics` listener already exists.

- [x] `apps/api/internal/platform/metrics/metrics.go`: register three counters on the private
  registry (`amorae_signups_total`, `amorae_couples_created_total`, `amorae_couples_paired_total`)
  with methods `SignedUp()`, `CoupleCreated()`, `CouplePaired()`.
- [x] `apps/api/internal/modules/auth/service.go`: declare `type Events interface { SignedUp() }`,
  add it to `NewService`, call it after `Register` commits.
- [x] `apps/api/internal/modules/couples/service.go`: the same with `CoupleCreated()` after
  `Create` and `CouplePaired()` after `Join`.
- [x] `apps/api/internal/app/app.go`: create `m := metrics.New()` before the services (it is
  created in the HTTP section today) and pass `m` to both.
- [x] Fakes in `auth/service_test.go`: a no-op `Events`.

**NFR baselines:**

- [ ] `docs/requirements/03-non-functional-requirements.md`: go through the NFR-SEC and NFR-OBS
  items and tick each one that applies to Phase 1. Request ids, access logs, `/metrics`, rate
  limits and the BFF origin checks are already in place.
- [ ] `apps/api/internal/platform/mailer/log.go` and every new handler: confirm nothing logs a
  password, token or email body. The log mailer prints the body on purpose, and that body will
  contain the reset token, so it must only run in development.

---

## Step 12 — Close out the phase

- [ ] Every Phase 1 Must in 02 says **Implemented**: FR-AUTH-001–004, 006–008, 010–012;
  FR-ACCT-001, 002, 004–007; FR-PAIR-001–005, 007, 008; FR-PWA-001–009.
- [ ] FR-PWA-009 (idempotency keys) is a Must, but no queued write reaches a real endpoint in
  Phase 1. Either build the client key and an `Idempotency-Key` check now, or record in 02 that it
  moves to Phase 3, where event creation is the first non-idempotent queued write.
- [ ] Should items are either Implemented or explicitly moved to a later phase in 02:
  FR-AUTH-005, 009; FR-ACCT-003, 008; FR-PAIR-006; FR-PWA-010, 011, 013.
- [ ] Q-03, Q-04, Q-05, Q-08, Q-09, Q-14, Q-15, Q-20 and Q-21 are Resolved in 05.
- [ ] `API.md` has no Phase 1 endpoint left under "Planned endpoints".
- [ ] `npm run check` and `npm run test:api` (with the database URL) pass.
- [ ] Update "Current status" in [01 §8.1](./requirements/01-product-requirements.md).

Then start Phase 2 (prayer), after Q-07 and Q-16 are answered.

---

## Suggested order


| #   | Step                                     | Blocked by                     |
| --- | ---------------------------------------- | ------------------------------ |
| 1   | Fix deletion bugs, apply migration 00005 | Nothing                        |
| 2   | Finish pairing                           | Nothing                        |
| 3   | Invite regeneration and guess limits     | Q-08                           |
| 4   | Change password                          | Nothing                        |
| 5   | Deletion with password                   | Step 4 (shared session code)   |
| 6   | Answer the Phase 1 questions             | You                            |
| 7   | Password reset                           | Q-04                           |
| 8   | Sign-up consents and age                 | Q-10, Q-20                     |
| 9   | Leave couple                             | Q-09                           |
| 10  | Data export                              | Step 9 (the promise to export) |
| 11  | Metrics and NFR checks                   | Q-14                           |
| 12  | Close out                                | Everything above               |


Steps 1, 2, 4 and 6 can start today.
