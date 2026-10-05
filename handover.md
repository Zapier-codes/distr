# distr — Session Handover

This file tracks the task board for this repo (Zapier-codes/distr, a fork of the upstream Distr project run as the public request storefront). It did not exist before 2026-10-05. Nothing in Task G is built; every status below is read from the code, not run.

Conventions for the code itself live in `CLAUDE.md`: build, test and lint through `mise` only, Go on chi/v5, the next migration number is `149` (latest is `148_product_payment`).

## Handoff process — ONE combined patch, apply with `git am` + `git push`

**Standing rule for every session (same process as the Zealot repo's `handover.md`):**

1. Session does the work on a branch, never pushes directly to `main`.
2. **Deliver exactly ONE `.patch` file, a single combined commit.** Everything the session changed goes into it: code, tests, migrations **and** the `handover.md` update (task board, status marks, session log). Squash the session's work into one commit on top of `main`, then run `git format-patch -1 HEAD`. Never hand over a numbered series (`0001…000n`), never a separate "docs" patch, never several files the operator has to order. If the session did several slices, they all ride in this one commit; list the slices in the commit body.
3. **The operator's checkout is already in place and up to date, so applying is just this, with no `checkout`, no `pull`, no `status` dance:**
   ```
   cd ~/distr
   git am ~/storage/downloads/<the-one-patch>.patch
   git push
   ```
   The session names the exact file it is handing over. Before handing it over the session must confirm the patch applies cleanly (`git apply --check`, or a throwaway `git am`) against the `main` tip it cloned, and say which commit that was. `git am` fails loudly if the tree has diverged since the patch was generated; if that happens, don't force it: the operator reports back and the next session rebases the patch instead of resolving conflicts blind. If a *previous* patch was only partly applied, say so before generating a new one, because a combined patch built on the wrong base will not apply.
4. **After the push, check the right workflows.** A push to `main` that touches anything outside `website/**`, `docker-autoheal/**` and `sdk/**` starts, among others, **`Publish to GHCR`** (`.github/workflows/publish-ghcr.yaml`, the Zapier-codes pipeline: build the community image, push `ghcr.io/zapier-codes/distr`, then call Render's deploy hook) and `Build Distr` (the upstream project's pipeline, left untouched). **`handover.md` is not in those ignore lists, so even a docs-only patch starts a build and a Render deploy.** Judge the deploy by `Publish to GHCR`, not by whichever run sits on top of the Actions list.
5. Build and behaviour are verified after that push (CI or a manual smoke test), not claimed as verified by the session itself. Say plainly what the sandbox could not run (here: no `mise` run unless the session actually ran it) and mark such work **code-complete, not run**.

## Task-splitting formula (TSF)

Use it for any task bigger than one obvious change, and write the slice table into the task's board entry *before* any code is written. Each slice must be small enough for the operator to verify quickly and revert alone.

**Slice test, all five must pass:**

1. **One behaviour.** The goal fits in one sentence with no "and".
2. **Small.** Roughly 6 files or fewer and 300 changed lines or fewer, ideally one layer (migration, `internal/db`, `handlers`, frontend). Over that, split again.
3. **Bootable after every slice.** Never leave a handler calling a function a previous slice deleted. Removals land after their replacement exists. A migration ships with its `.down.sql`.
4. **Checkable.** Has a quick check for the operator and a machine check where the sandbox allows (`mise run test:go`, `mise run lint:go`, `mise run lint:migrations`, `mise run test:frontend`). Anything the sandbox cannot run is written down as "not verified".
5. **Reversible.** Each slice lists what to revert. Money-moving slices say what happens to payments already in flight.

**Ordering formula** (top to bottom, ties broken by lowest risk first): 1. owner decisions resolved, so nothing is built on a guess; 2. foundation before UI (migration, db, handler before frontend); 3. replacement before removal; 4. independent low-risk slices before cross-cutting ones; 5. purely additive slices last; 6. the handover update (status marks, session log) is always the last edit of a session and goes into the same single patch.

**Slice card** (one per slice): `ID · Goal · Depends on · Files (predicted) · Acceptance check · Verify · Risk`.

**Naming and delivery:** Task `G` has slices `g.i`, `g.ii`, and so on. Slices are planning and verification units, not separate patches. Branch `feat/task-g-<slug>` (or `docs/task-g-<slug>`) off `main`; the session's slices are squashed into one commit `type(task-g): <slices done>` and delivered as one patch per the Handoff process above. A session takes 1 to 3 slices, never more than it can honestly verify. Commit messages follow the repo's semantic-PR style (`feat`, `fix`, `docs`, with a scope), which `semantic-pr.yaml` checks on pull requests.

## 1. What distr is in this repo today

A public request storefront, not yet a marketplace of third-party apps.

- `ProductService` is a storefront entry (`app` or `website`), seeded with two rows, `branded-app-store` and `branded-web-store`. It has an optional one-time price (`price_minor`, `price_currency`). It has no owner and no developer column, and the API only lists and reads entries (no create or update route).
- A visitor submits a `ProductRequest` (branding form) with no account. That creates a `TenantConfig` in `awaiting_gate`.
- Free/paid gate: `f.xii` free tier (`FreeTierClaim`, unique on the salted hash of a browser device fingerprint, so one free product per device across the whole storefront, ever). Anything the free tier did not cover goes to `f.xiii`.
- `f.xiii` payment: `internal/productpayment` creates a B-Pay payment link (`internal/bpay`, `POST /payments` with `payment_link: true`), stores a `ProductPayment`, and on B-Pay's signed webhook (HMAC-SHA512, `X-Webhook-Signature-512`) re-fetches the payment, requires `succeeded` and an exact match of amount and currency with what was stored, then moves the record to `queued` and dispatches the build to Storeapp's workflow.
- Distr holds no binary. A build ends as a GitHub Release asset and an email with a download link (`f.v`).

## 2. Rules from the owner (stated 2026-10-05)

1. A user gets one app free. When the user comes back, further apps require payment.
2. Any developer can list an app for sale on distr. The listing must follow the same rule: free for the first install.
3. The platform takes 30% of the price per app.
4. **All prices, charges and shares are in US dollars, with no currency conversion anywhere** (operator, 2026-10-05, supersedes the earlier "a user can pay in any currency of their choice"). B-Pay's own checkout page takes the payment. A payer whose card is in another currency is converted by their own bank or card network, not by distr or B-Pay.
5. **No logins. The marketplace is open to everyone.** This is deliberate (operator, 2026-10-05): an email or account identity would let the same person become a "new user" with a second email. The free claim is therefore tied to the **device**, by fingerprint, so a person only counts as new by changing device completely. Incognito mode and a different browser on the same device must not make a new person.

## 3. Cross-check of the rules against the code (all three repos)

| Rule | State | Evidence |
|---|---|---|
| 1. One free, then paid | **Built, but it does not yet meet rule 5.** | `FreeTierClaim` is unique on `fingerprint_hash` alone, keyed to a device with no accounts, as intended. Checked in `frontend/ui/src/util/device-fingerprint.ts`: the signals are user agent, language, platform, screen size and depth, pixel ratio, CPU cores, touch points and timezone. **Incognito on the same browser gives the same hash (works). A different browser on the same device sends a different user agent, language or platform and so gets a second free product (fails rule 5).** In the other direction, two people with the same phone model, timezone and language send near-identical signals and collide, so the second is asked to pay. The server only receives the 64-character hash, so a caller that skips the browser can send any value (`applyFreeTierGate` cannot verify it); Turnstile (checked on every request when `TurnstileSiteKey` is set) and the per-IP rate limit are the only guards. With no salt configured or no fingerprint sent, the gate fails closed to payment. |
| 2. Developer listings | **Not built; model decided (D6).** | No owner column, no create route, no listing review state. Also see the conflict in section 4. |
| 2. First install free per listed app | **Not built.** | The only free gate is the global one above, and it is about harvesting a storefront product, not installing a developer's app. Distr never sees installs. |
| 3. 30% platform share | **Not built.** | No ledger, no split, no payout table. `ProductPayment` is one row per `TenantConfig` (`tenant_config_id UNIQUE`), so it cannot hold a purchase of a developer app by a device. |
| 4. USD only (revised) | **Mostly there; one thing to confirm.** | `ProductService.price_currency` is any ISO code today, so USD-only for developer listings must be enforced (g.ii). `ConfirmPaid` already requires the exact stored amount and currency, which is correct for USD-only and needs no change. **Unconfirmed: that the live B-Pay merchant account has at least one connector that accepts USD and shows it at checkout.** `control-center` (cloned) is the stock Hyperswitch dashboard plus one Zapier CI commit and holds no live configuration, and B-Pay-backend's repo config cannot show which connectors are enabled on the live account (they are stored in its database). The operator has to read that from the control centre. Local-only payment methods (for example some Paystack, Korapay or Flutterwave rails, whose connectors exist in B-Pay) may not take USD, so USD-only can shrink the methods a payer sees. |

B-Pay-backend facts relevant to the split:
- Hyperswitch has split payments, for example `StripeSplitPaymentRequest` in `crates/common_types/src/payments.rs` with `charge_type`, `application_fees` and `transfer_account_id`, with Adyen and Xendit transformers too. That is the natural way to take 30% as a platform fee, but it needs each developer to be a connected account at the processor, and it depends on which processors the live B-Pay merchant account has enabled. I could not read the live configuration.
- **`HANDOVER.md` and `README.md` in B-Pay-backend contain no description of a Distr integration** (no word-boundary match for "distr"). So how the payment system was intended to integrate with distr is only recorded on the distr side (`f.xiii`) and in Zealot (Task 32, B-PAY: the store-listing fee, $14.99 one-time, with `draft -> awaiting_payment -> live`). That intent is not written down at B-Pay.

## 4. Cross-repo conflicts and dependencies

- **Zealot says distr never holds a binary and a harvest never creates a `Release`** (Zealot handover, "40m answered and the harvested-app revamp", points 3 and 4 of the invariants, and slice 40n-g, which points at this file). Developer-listed apps for sale are a different thing: the buyer receives the developer's app, so some system must hold or deliver that binary. This rule set does not say which (Zealot's catalog and releases, or something in distr). It must be decided before g.iii.
- **Zealot's recorded APK rule is out of date.** Zealot 40n-c still says "verify, never re-sign, reject foreign APKs". The owner has since decided: a developer who publishes through the organisation's own stores has their APK re-signed with the organisation key and gets no SDK (the existing store-listing payment gate covers them); an AAB upload gets SDK injection in a copy with the clean AAB kept for Play; a harvested tenant app gets the SDK in CI. Zealot's handover must be corrected there. This file does not change Zealot.
- **Harvest download window:** the email button and the stored file last 7 days, and the email says so. A scheduled cleanup after 7 days is needed on whichever side stores the file (slice 40n-f in Zealot).
- **Bundle route:** Storeapp's CI sends the tenant bundle straight to the storage repo, without Zealot. A workflow dispatch cannot carry a file, so the proposal is one fine-grained token in Storeapp scoped to the storage repo (contents and actions write), putting the bundle there as a temporary release asset and dispatching with its name. Not yet confirmed by the owner.
- **Two fees must not be mixed up:** the Zealot store-listing fee (what a developer pays to publish) and the distr 30% share (what the platform keeps from a sale). They run through B-Pay but are different charges.

## 5. Task board

### 🆕 Task G: developer listings, first install free, 30% platform share, USD display with pay-in-any-currency (docs only; nothing below is built)

Source: the operator's rules in section 2 and the cross-check in sections 3 and 4. Order: g.i (done), g.i-b, g.i-c, g.iii-a, g.iii-b (with g.ii), g.v, g.iv, g.ix, g.vi, g.viii, g.vii, g.x. The multi-currency proof that g.ii used to be is gone (D5). g.ii goes first among the builds because price and currency rest on it. Nothing is built until the owner answers the questions each slice names; g.i is closed by D1 and D2.

| ID | Goal (one behaviour) | Depends on | Files (predicted) | Acceptance check | Verify | Risk / revert |
|---|---|---|---|---|---|---|
| g.i | ✅ **Decided, docs only.** The free rule: one shared free claim per device across harvests and developer apps, identity is the device with no logins (D1, D2) | none | this file | The rule is in section 5b | none needed (docs) | low. Revert: this file |
| g.i-b | Make the fingerprint browser-independent and strict (D3): drop user agent, language and platform from the signals, keep the hardware and screen signals, add the renderer string only if stable | g.i | `frontend/ui/src/util/device-fingerprint.ts`, `device-fingerprint.spec.ts` | The same signals from two different user agents give the same hash; different screen or core counts give different hashes | `mise run test:frontend` **not run**; check on two browsers of one real device and on two phones of one model (not done) | medium: it resets the claimed set once (see the consequence in 5b). Revert: the two files |
| g.i-c | Ops check that production has `TURNSTILE` keys and the device-fingerprint salt set, and that the free claim cannot be reached without Turnstile (D4) | g.i | `deploy/`, `render.yaml` or the runbook, possibly one test | A request with no or a bad Turnstile token creates no free claim | `mise run test:go` **not run**; the live Render environment was not inspected | low |
| g.ii | **USD only.** Developer listing prices are USD cents and cannot be set in another currency; the payment is created in USD; `ConfirmPaid` is unchanged | D5 | migration `150` is shared with g.iii-b (price column), `internal/productpayment`, `internal/db`, tests | A listing in a non-USD currency is rejected, and a USD payment confirms exactly as today | `mise run test:go` **not run**; **operator check first:** the live B-Pay account has a USD connector and its checkout page shows `$` | medium, money. Revert: the check |
| g.iii-a | **Developer role and join path** (D6, D7, D8): a narrow `developer` role exempt from the Pro rule, an open sign-up path that adds the user to the one platform organization with no organization created, the old register path switched off, `USER_EMAIL_VERIFICATION_REQUIRED=false` set and recorded in the deploy files, MFA required before a payout detail is saved, every existing route refusing the role | D6, D7, D8 | migration `149` + `.down.sql` (role value), `internal/types`, `internal/handlers/auth.go`, `internal/routing`, `internal/middleware`, tests | A new developer account exists in the platform organization, cannot reach any vendor-portal route, and `POST /auth/register` no longer makes organizations | `mise run test:go`, `mise run lint:migrations` **not run**; an authorization test per existing route group | **high, security.** Revert: down migration and the routing change |
| g.iii-b | **Listings with an owner:** owner user id, listing status, price in USD minor units (g.ii), create and update routes, and the review states of D9 | g.iii-a, D9, D12 | migration `150` + `.down.sql`, `internal/db`, `api/`, `handlers/`, `mapping/`, `types/` | A developer creates and edits only their own listing; it is not purchasable until its status allows it | `mise run test:go`, `mise run lint:migrations` **not run** | high. Revert: down migration |
| g.iv | The free claim at the first acquisition (D2, D13) for developer apps | g.i, g.iii-b | `internal/db`, the gate code next to `applyFreeTierGate` | A second install of the same listing needs payment under the rule in g.i | `mise run test:go` **not run** | medium. Revert: the gate code; `FreeTierClaim` is untouched |
| g.v | Purchase record and 70/30 ledger in integer minor units, rounding rule fixed in the migration comment, test for an uneven split ($9.99) | D10, g.iii-b | migration `151` + `.down.sql`, `internal/productpayment`, `internal/db` | The two shares always sum to the price | `mise run test:go` **not run** | high. Revert: down migration |
| g.vi | Developer payout ledger and the operator's monthly manual payout view (D11): balance per developer, hold, minimum, a payout record | D11, Q17, g.v | migration `152` + `.down.sql`, `internal/db`, `handlers/`, `frontend/ui/` | A developer sees their balance and the operator can record a payout; nothing moves money automatically | `mise run test:go` **not run** | high, money. Revert: down migration |
| g.vii | Storefront shows the USD price on listings and in the checkout hand-off (no currency picker, D5) | g.ii | `frontend/ui/src/app/store/` | The store page shows a `$` price and the buy button opens the B-Pay checkout | `mise run test:frontend`, `mise run lint:frontend` **not run**; browser check | low. Revert: the frontend files |
| g.viii | **Campaign card** (D17): the one listing-card format on the storefront (icon, name, pitch, USD price, Get button), styled as an ad-like card | g.iii-b, Q16 | `frontend/ui/src/app/store/` | Every live listing renders as a campaign card; no paid placement | `mise run test:frontend`, `mise run lint:frontend` **not run**; browser check | low. Revert: the frontend files |
| g.ix | **Gated paid download in Zealot** (D12): a download link issued only after checking distr's entitlement, short-lived | g.iii-b, g.v | **Zealot repo, its own handover**; a distr entitlement endpoint | A buyer with an entitlement gets a link that expires; anyone else gets none | not defined here; cross-repo | high; cross-repo |
| g.x | **Harvest rebuild entitlement** (D15): a paid tenant re-requests a build without paying again, capped | Q17 | `internal/productpayment`, `internal/buildtrigger`, `internal/db`, migration `153` | A paid tenant can re-dispatch up to the cap and not beyond | `mise run test:go` **not run** | medium


## 5b. Decisions recorded (operator, 2026-10-05)

- **D1. Identity is the device, never an account** (answers Q1). No logins, no email identity. Accounts were recommended and rejected for the reasons in rule 5.
- **D2. One shared free claim per device across all of distr** (answers Q2). The same device claim covers a harvest and a developer's app: a device gets one free acquisition in total, then pays. It does not stack per listed app. For a developer app the free acquisition costs the buyer nothing and earns the developer nothing.
- **D3. The fingerprint must be browser-independent, and strict by default.** Drop the browser-specific signals (user agent, language, platform); keep the hardware and screen ones (screen size and depth, pixel ratio, CPU cores, touch points, timezone), and add the graphics-card renderer string if it proves stable. Strict means: a false match between similar devices is accepted, because the cost of a match is only that the visitor is asked to pay, while a missed match gives away a free app. Not yet measured on real devices.
- **D4. The fingerprint is a deterrent, not security.** It is computed in the browser, so it is never trusted as proof. Production must have Turnstile configured so that every request that can claim the free product is verified, and the per-IP rate limit stays.
- **D5. USD only, no conversion** (answers Q3). Developer listing prices are stored as USD minor units (cents). Every B-Pay payment is created in USD. The 30% share, the rounding and the developer payouts are all in USD, which removes the exchange-rate, quote and spread questions entirely. `ConfirmPaid` stays an exact match.
- **D6. Developers are distr Users, the platform is the one Organization, visitors stay anonymous** (answers Q4, operator 2026-10-05). A developer is a `UserAccount` in the platform's organization; the organization is the platform itself; people who visit the store have no account and the platform records only the device fingerprint (D1). This replaces my earlier recommendation to keep developer identity in Zealot. Cross-checked against the code, it holds with four constraints that g.iii-a must deal with (read from `auth.go`, `user_accounts.go`, `types.go`, `global_limits.go`; **not run**, and not every route's authorization was audited):
  1. **Sign-up creates a new organization per person.** `POST /auth/register` calls `db.CreateUserAccountWithOrganization`, so every signup would become its own platform. Developer sign-up therefore needs its own path that adds the user to the one platform organization. The existing path must be switched off (`REGISTRATION=disabled`) or left unreachable once that exists. There is also a global organization limit check on the old path.
  2. **The roles are vendor-portal roles** (`read_only`, `read_write`, `admin`). A developer must not be an admin, and there is no seller role. A developer in the platform organization could otherwise reach the platform's own vendor data (customers, deployment targets, registry, access tokens). A new narrow role (for example `developer`) with access only to its own listings and earnings is needed, and every existing route has to be checked to refuse it.
  3. **Non-admin users require a Pro subscription.** `createUserAccountHandler` refuses `read_only` and `read_write` users unless `organization.SubscriptionType.IsPro()`. **Settled by D8:** the new role is exempt in code.
  4. **Listing ownership is a user id,** so a developer can only change their own listings. This is the isolation rule the new role has to enforce in the database queries, not only in the UI.
- **D7. Developer sign-up is open, into the one platform organization, with no email verification** (answers Q14, operator 2026-10-05). Anyone can sign up as a developer. A sign-up never creates an organization: all developers who sell are users of the one platform organization (D6). **Email verification is switched off:** `USER_EMAIL_VERIFICATION_REQUIRED=false` (read in `internal/env/env.go`; the default is `true`, it is instance-wide, and `middleware.RequireEmailVerified` and `SendUserVerificationMail` follow it). This is a distr setting and does not depend on the database being Supabase Postgres. MFA is required before any payout detail is saved (distr already has MFA). The gate is the listing, not the account: nothing a developer lists is purchasable until it has been reviewed (D9). **Consequence the operator accepted:** an account can be created with an address the person does not own, so a mistyped or fake email can never be recovered through mail, and the email on an account is not proof of anything. MFA before payout and the review gate carry that weight.
- **D8. The new `developer` role is exempt from the Pro-plan rule** (answers Q15). Distr refuses non-admin users unless the organization is on Pro; the exemption is made in code for this role, so the number of developers is not capped by a plan.
- **D9. Listings are reviewed first, then pulled if there is a problem** (answers Q5). A developer's first listing and any update that changes the price or the binary are held for manual review; after that a listing is published and pulled if a problem appears. Zealot's states `draft`, `awaiting_payment`, `live`, `suspended` are the model to reuse.
- **D10. The platform keeps 30% of the gross price and absorbs B-Pay's processing fee** (answers Q6). The 30% is stored on each purchase row when the sale happens, so a later rate change cannot alter past sales. **Rounding** (answers Q7): all in integer cents; the platform share is 30% of the price rounded half up, the developer gets the remainder, so the two always sum to the price ($9.99 gives $3.00 and $6.99).
- **D11. Developer payouts are a ledger with manual operator payouts** (answers Q8): a monthly cycle, a minimum balance, and a hold of a few weeks for refunds and chargebacks. The amounts and the hold length are not set and are for the operator to choose. Connected-account splits through B-Pay are a later step, not part of Task G.
- **D12. The developer app's binary lives in Zealot; distr stores the listing, the price and the purchase entitlement** (answers Q9). Zealot issues a short-lived download link only after checking the buyer's entitlement, the same pattern as the harvest download. **Gap:** Zealot's catalog is public today, so a paid app needs a gated download in Zealot, a cross-repo slice in Zealot's own handover (g.ix).
- **D13. The free claim is counted at the first acquisition** (answers Q10): the moment distr issues the download entitlement. No install signal is needed.
- **D14. Harvested tenants pay the same fee as everyone else**, shown on the harvest page before payment (answers Q11).
- **D15. A paid harvest entitles the tenant to rebuilds** (answers Q12): the link expires after 7 days, the entitlement does not; a re-request re-dispatches the same paid `TenantConfig` free of charge, capped at about three rebuilds (the exact cap is for the operator). A small separate slice (g.x).
- **D16. A token in Storeapp is acceptable for the bundle route** (answers Q13), under conditions: prefer a GitHub App installation token minted per run (valid about an hour); if a fine-grained token is used, scope it to the storage repo only with contents and actions write and a 90-day expiry. The storage workflow must check the bundle's SHA-256 and package name against what distr recorded at dispatch time and reject a mismatch, because it signs whatever arrives with the organization key. Lives in the Storeapp and Zealot handovers.
- **D17. A developer's listing is shown on the platform as a campaign card, like an ad** (operator: "the card should come as a campaign card on the platform like an ad"). Recorded as stated. My reading, to be confirmed in Q16: "the card" is the listing card, and "campaign card" is its look and place on the storefront.
- **Consequence to plan for:** changing the signals changes every hash, so a device that already used its free product can claim once more after g.i-b ships. Accepted unless the operator says otherwise.

## 6. Open questions for the owner

1. ✅ Answered (D1, D3): identity is the device, no logins, browser-independent.
2. ✅ Answered (D2): one shared claim per device across all of distr; it does not stack per app.
3. ✅ Answered (D5): USD only, no conversion. Still to confirm by the operator: that the live B-Pay account has a connector that accepts USD.
4. ✅ Answered (D6): developers are distr Users in the one platform Organization; visitors stay anonymous. *Superseded earlier recommendation, kept for the record:* keep developer identity in Zealot.
5. ✅ Answered (D9): manual review for a first listing and for price or binary changes, then publish and pull.
6. ✅ Answered (D10): 30% of the gross price, the platform absorbs B-Pay's fee, the rate is stored per sale.
7. ✅ Answered (D10): platform share is 30% rounded half up in cents, the developer gets the remainder.
8. ✅ Answered (D11): ledger and manual monthly payouts with a minimum and a hold; amounts for the operator to set.
9. ✅ Answered (D12): the binary lives in Zealot behind an entitlement-checked, short-lived link.
10. ✅ Answered (D13): the free claim is counted at the first acquisition.
11. ✅ Answered (D14): harvested tenants pay the same fee, shown before payment.
12. ✅ Answered (D15): the link lasts 7 days, the paid entitlement allows about three free rebuilds.
13. ✅ Answered (D16): yes, a GitHub App token preferred, with a bundle hash and package-name check in the storage workflow.
14. ✅ Answered (D7): open sign-up into the one organization, no email verification, MFA before payout, review at the listing.
15. ✅ Answered (D8): the `developer` role is exempt from the Pro rule in code.
16. What is the campaign card (D17)? *My reading:* the listing card on the storefront, styled like an ad, shown for every listing. *Alternatives:* (a) a paid promotion a developer buys for extra placement, (b) a card the platform runs for its own campaigns. *Recommendation (not yet decided):* build it as the one listing-card format now (g.viii) with no paid placement. If paid placement is wanted later it is a separate feature: it needs its own price, its own ledger entry, and a visible "Ad" or "Sponsored" label on every paid card, which advertising rules in most countries require.
17. Which two numbers does the operator set for payouts (D11) and rebuilds (D15): the minimum payout balance and hold length, and the rebuild cap?

## 7. Not done

No code, no migration and no test run (operator asked for docs only). The live B-Pay configuration and processors were not inspected.

## Session log

### 2026-10-05 -- Task G: first handover for distr (operator: "check the repos, cross-check, update the handover file on the distr, provide the .patch file"; docs only; no application code; no testing; one patch)
- **Base:** distr `main` @ `436b189` (`feat(f.xiii)`). This patch creates `handover.md`, which did not exist.
- **Read:** distr payment, free-tier, request and webhook code and migrations 142 to 148; B-Pay-backend `HANDOVER.md`, currency and split-payment code; Zealot `handover.md` (Task 32 B-PAY, 39, 40n).
- **Found:** one-free-per-device is built and keyed to a device; developer listings, the 30% ledger and payout are not built; pay-in-any-currency is blocked by distr's exact currency match and unproven in B-Pay; B-Pay's own handover never mentions distr; Zealot's 40n-c APK rule is out of date against the operator's decisions.
- **Allocated:** Task G, slices g.i to g.vii.
- **Needs the operator:** apply and push (see the Handoff process); answer the questions in section 6, starting with 3, 9 and 10 (1 and 2 are answered below).

### 2026-10-05 -- Task G: no-logins decision and fingerprint finding recorded (operator: "we do not want logins we want the market place to be open to all ... the user will need to change his device completely"; docs only; no application code; no testing; one patch, replaces the earlier handover patch)
- **Base:** distr `main` @ `436b189`. `origin/main` was fetched at the start and still had no `handover.md`, so this patch also creates the file, and it **replaces** the earlier `distr-task-g-handover.patch`. If that earlier patch was already applied, do not apply this one; say so and the next session produces an incremental patch.
- **Decided:** D1 to D4 in section 5b: device identity with no logins, one shared claim per device, browser-independent strict fingerprint, fingerprint as a deterrent only with Turnstile required.
- **Verified (read, not run):** `device-fingerprint.ts` signals, `devicefingerprint.go` and `applyFreeTierGate`: a different browser on the same device is currently a new person, and similar devices can collide; the gate fails closed to payment without a salt or fingerprint.
- **Allocated:** g.i closed; new slices g.i-b (signals) and g.i-c (Turnstile and salt check).
- **Needs the operator:** apply and push; answer Q4 next.

### 2026-10-05 -- Task G: USD only (operator: "the b-pay has a checkout page there is no conversion needed ... all is USD", with a pointer to the control-center repo; docs only; no application code; no testing; one patch, replaces the earlier handover patches)
- **Base:** distr `main` @ `436b189`, still without `handover.md`; this patch creates the whole file and replaces the two earlier ones. Do not apply it on top of either.
- **Decided:** D5, USD only, no conversion. Rule 4 rewritten. Q3 answered. Slice g.ii is now the USD-only enforcement, g.vii loses its currency picker.
- **Read (not run):** `Zapier-codes/control-center` is the stock Hyperswitch dashboard plus one Zapier commit (CI, GHCR, Render deploy hook), and holds no live configuration; B-Pay-backend's `config/development.toml` `pm_filters` lists per-connector currencies but not what the live account enabled; the Paystack, Korapay, Flutterwave and Juicyway connectors exist in the Rust app.
- **Not verified:** which connectors the live B-Pay account has enabled and whether any takes USD.
- **Needs the operator:** apply and push; read the enabled connectors and their currencies from the live control centre; answer Q4 (answered in the next entry).

### 2026-10-05 -- Task G: developers are distr Users, the platform is the Organization (operator: "the platform already has user/organisation settings so that is what we are going with ... visitors are anonymous, platform only records device fingerprint"; docs only; no application code; no testing; one patch, replaces the earlier handover patches)
- **Base:** distr `main` @ `436b189`, still without `handover.md`; this patch creates the whole file and replaces the earlier ones. Do not apply it on top of any of them.
- **Decided:** D6. Q4 answered. My earlier recommendation (developer identity in Zealot) is superseded and kept in Q4 for the record.
- **Read (not run):** `internal/handlers/auth.go` (register creates an organization per signup), `internal/handlers/user_accounts.go` (non-admin users need Pro; a user limit check exists), `internal/types/types.go` (roles are `read_only`, `read_write`, `admin`), `internal/subscription/global_limits.go`. **Not read:** `checkUserCreationLimits` and the vendor organization's user-account quantity in the community build that `publish-ghcr.yaml` ships; check before g.iii-a.
- **Allocated:** g.iii is split into g.iii-a (role and join path) and g.iii-b (listings); migration numbers shift to 149, 150, 151.
- **Needs the operator:** apply and push; answer Q14 next, then Q5 and Q15.

### 2026-10-05 -- Task G: all recommendations accepted; no email verification, one organization, campaign card (operator: "I love all the recommendations add them all but no email verification ... all developers under one organisation ... the card should come as a campaign card on the platform like an ad"; docs only; no application code; no testing; one patch, replaces the earlier handover patches)
- **Base:** distr `main` @ `436b189`, still without `handover.md`; this patch creates the whole file and replaces the earlier ones. Do not apply it on top of any of them.
- **Decided:** D7 to D17; questions 5 to 15 answered. D7 differs from my recommendation on one point (email verification is off) and records its consequence. D17 is recorded as stated and its meaning is open (Q16).
- **Read (not run):** `internal/env/env.go` for `USER_EMAIL_VERIFICATION_REQUIRED` (default true, instance-wide). Not checked: whether anything else (a password reset flow, an invite) assumes a verified email.
- **Allocated:** g.iii-a and g.iii-b updated to the decisions; new g.viii (campaign card), g.ix (gated download, Zealot), g.x (rebuild entitlement); g.vi is now the ledger and manual payout view; migration numbers 149 to 153.
- **Needs the operator:** apply and push; answer Q16 (what the campaign card is) and Q17 (the numbers).
