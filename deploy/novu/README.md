# distr's own Novu instance (leaf f.xi)

`distr` hands the "your app is ready" mail to Novu (`internal/novu`, leaf f.v). This is the runbook for the Novu instance
that serves **only** `distr`. Nothing here can be done by code: it is an account, a mail provider and a workflow in
Novu's dashboard, followed by three environment values and one check.

## Why a second instance

Zealot has its own Novu pipeline for operators and catalog subscribers (update alerts, moderation, distribution
status). `distr` writes to a first-time requester going through a self-service "get your branded app or site" flow. The
audience, the tone, the sender identity and the templates differ, so this is a separate Novu organization with its own
credentials, not a second workflow inside Zealot's. Do not reuse any Zealot key, integration or sender address.

## 1. Create the instance

1. Sign up at Novu with a login and an organization that are **not** Zealot's. Create the organization for `distr`
   alone.
2. Pick the region when you create it. The API base URL depends on it: `https://api.novu.co` (US, the default of
   `NOVU_API_URL`) or `https://eu.api.novu.co` (EU). A self-hosted Novu has its own URL. Check the URL against
   Novu's current docs, it was written from memory.
3. Work in the **Production** environment. From its **API Keys** page copy the **Secret Key**. The Application
   Identifier is public and is not what `distr` uses.

## 2. Sender identity

In the **Integration Store**, connect one email provider and make it the primary email integration of the Production
environment.

- **From address:** an address on a domain of this program, with SPF and DKIM set up at the provider. A mail that fails
  both goes to spam, and this mail is one the requester is waiting for.
- **Sender name:** the name of the self-service product the requester used, not Zealot's. This is a decision for the
  operator and is not made here.
- **Reply-to:** an inbox somebody reads, since the mail is addressed to a person who asked for something.

## 3. The workflow

Create a workflow whose **identifier** is exactly `tenant-build-ready` (or set `NOVU_BUILD_READY_WORKFLOW_ID` to
whatever you use). Give it one step, **Email**:

- **Subject:** `Your {{payload.appName}} app is ready`
- **Body:** the custom HTML in [`tenant-build-ready.html`](tenant-build-ready.html)
- No digest, no delay, no in-app or push step.
- If the workflow has a "critical" or "cannot be unsubscribed" setting, turn it on: the subscriber is created by the
  trigger itself and has no preferences, and the mail is transactional.

`distr` sends two payload fields and nothing else: `appName` (string) and `downloadUrl` (string). The subscriber id is
the id of the tenant record, and its email is the requester's. If your Novu version validates the payload against a
schema, declare exactly those two as required strings.

## 4. Set the three values on `distr-web`

| Variable                       | Value                                                                          |
| ------------------------------ | ------------------------------------------------------------------------------ |
| `NOVU_API_KEY`                 | the Secret Key from step 1. Secret; never put it in the repository.            |
| `NOVU_API_URL`                 | only for the EU region or a self-hosted Novu; leave unset for `api.novu.co`.   |
| `NOVU_BUILD_READY_WORKFLOW_ID` | only if the identifier in step 3 is not `tenant-build-ready`.                  |

They are already declared as `sync: false` in `render.yaml`, so a Blueprint re-sync never replaces them.

## 5. Check it sends

The image is distroless, so the Render Shell tab has no shell to run this in. Run the same image anywhere Docker is
available, with the same Novu values. `distr novu-test` needs the five variables the server requires to start, but it
connects to none of them, so the database and Loki values can be placeholders; `DISTR_HOST` should be real, because the
sample mail's button points at `<DISTR_HOST>/store`.

```sh
docker run --rm \
  -e DATABASE_URL=postgres://unused \
  -e DATABASE_ENCRYPTION_KEY=unused \
  -e JWT_SECRET="$(openssl rand -base64 32)" \
  -e LOKI_URL=http://localhost:3100 \
  -e DISTR_HOST=https://<your distr host> \
  -e NOVU_API_KEY=<the secret key> \
  ghcr.io/zapier-codes/distr:deploy-latest novu-test you@example.com
```

Expected output, and what to do when it is not:

- `Novu: api.novu.co, workflow tenant-build-ready` then `Novu accepted the trigger for you@example.com.` Check the
  inbox, and the spam folder. A mail in the inbox with the right sender name and a working button is what closes the
  open gate of `f.v` ("confirm it actually sends").
- `NOVU_API_KEY is not set, or NOVU_API_URL is not an http(s) URL`: the value did not reach the container.
- `Novu refused the trigger with status 401`: wrong key, or a key of the other region. `404` or a message about the
  workflow: the identifier in step 3 does not match.
- Accepted, but nothing arrives: the problem is on Novu's side of the trigger, so look at the **Activity Feed** of
  the Production environment and at the email integration (step 2).

The request `distr` makes (`POST {NOVU_API_URL}/v1/events/trigger`, `Authorization: ApiKey <key>`) was written from
memory of Novu's REST API in `f.v` and has never run against a real instance. The first run of this check is also the
first test of that request. If Novu changed the endpoint or the header scheme, the fix is in `internal/novu/client.go`.

## Keeping it separate

- A different Novu organization, different Secret Key, different integration and sender address from Zealot's.
- Zealot's deployment never gets this key, and this deployment never gets Zealot's.
- The two systems cross-communicate through the signed tenant record only, never through Novu.

## Not covered here

- Billing, free-tier or any other mail than "your app is ready": `f.xii` and `f.xiii` are separate leaves, and any new
  mail needs its own workflow in this instance and its own payload contract.
- A failed-build mail does not exist: a failed build sends nothing (`f.iv.zo`).
