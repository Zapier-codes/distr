# Deploying distr on Render (Storeapp Track f, leaf `f.x`)

Written, **not applied, not deployed, not run**. Everything below is the intended procedure, not a record of one.

## What exists

- `.github/workflows/publish-ghcr.yaml` builds the community edition and pushes `ghcr.io/zapier-codes/distr:deploy-<sha>` and `:deploy-latest`, then calls Render's deploy hook with `imgURL=...:deploy-<sha>`.
- `/render.yaml` is the Blueprint: one web service `distr-web` (image runtime) and its own Postgres `distr-db`. Nothing is shared with Zealot.
- `build-distr.yaml` (the upstream pipeline) is untouched.

## Operator steps, in order

1. Push this to `main`. The workflow runs and publishes the image (the first run has no deploy hook, which is only a notice).
2. In GitHub, **Packages -> distr -> Package settings**, make the package public, or add a Render Registry Credential and a `creds` block (see the comment in `render.yaml`).
3. In Render: **New -> Blueprint**, pick this repository. It creates `distr-web` and `distr-db`.
4. In the `distr-web` environment set the `sync: false` values:
   - `DATABASE_ENCRYPTION_KEY` and `JWT_SECRET`: `openssl rand -base64 32` each. Keep a copy somewhere that is not Render. Losing the encryption key makes encrypted rows unreadable.
   - `DISTR_HOST`: the public https origin, no path.
   - `STOREAPP_BUILD_GITHUB_TOKEN`: fine-grained token on the Storeapp repository only, **Actions: read and write** and **Contents: read** (`f.iii` and `f.v` flagged both).
   - `STOREAPP_BUILD_CONFIG_TOKEN`: at least 32 characters, the same value as the Storeapp repository secret of that name.
   - `DEVICE_FINGERPRINT_SALT`: at least 32 random characters (`openssl rand -hex 32`). Without it no request is free (`f.xii`); changing it later forgets every free claim.
   - `BPAY_API_URL` / `BPAY_API_KEY` / `BPAY_WEBHOOK_SECRET` (all three or none; `BPAY_PROFILE_ID` optional): B-Pay, the payment gateway (`f.xiii`). In B-Pay, point the business profile's webhook at `https://<DISTR_HOST>/api/public/v1/bpay-webhook` and use its payment response hash key as `BPAY_WEBHOOK_SECRET`. Then set a price on each product (see `configuration.mdx`, "Paid products"), or no request can be paid for.
   - `TURNSTILE_SITE_KEY` / `TURNSTILE_SECRET` (set both or neither), and `NOVU_API_KEY` (plus `NOVU_API_URL` for the EU region or a self-hosted Novu) from distr's own Novu instance, provisioned by `deploy/novu/README.md` (`f.xi`); that file also has the check that confirms a mail is sent.
5. Copy the service's **Deploy Hook** URL into the GitHub repository secret `RENDER_DEPLOY_HOOK_URL`. Pushes then deploy.
6. Check `https://<DISTR_HOST>/ready` answers 200, then `/store`.

## Object storage

The leaf asks for "its own Postgres and storage". Postgres is in the Blueprint. **No object storage is provisioned, on purpose:** Render has none, and nothing in Track f needs it. Build artifacts live only in GitHub Releases (a recorded decision), and tenant logos are `bytea` in `distr`'s own Postgres (`f.ii`). distr's S3 setting exists only for its built-in OCI registry, which is off (`REGISTRY_ENABLED=false`). If that registry is ever turned on, bring a bucket this program owns (S3-compatible, not Zealot's) and set `REGISTRY_ENABLED`, `REGISTRY_HOST`, `REGISTRY_S3_BUCKET`, `REGISTRY_S3_REGION`, `REGISTRY_S3_ENDPOINT`, `REGISTRY_S3_ACCESS_KEY_ID`, `REGISTRY_S3_SECRET_ACCESS_KEY`.

## Known gaps

- `LOKI_URL` is required at start-up by distr's env loader, so the Blueprint sets a placeholder. Anything that reads agent logs will fail; the request flow does not.
- `plan: starter` (web) and `basic-256mb` (database) are guesses at the smallest sensible paid sizes, not measured. Check them against Render's current plan names before syncing.
- The GHCR package visibility and the deploy hook are manual steps; nothing here can do them.
- The first real boot is also the first run of migrations 142-146 against a fresh database. None of them has ever run.
