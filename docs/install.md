# Installing Room Pass

This guide installs Room Pass for one event on a Kubernetes cluster you
operate: Room Pass, a Dex issuer behind it, the edge routes, and, if
participants act in Kubernetes with their own tokens, the kube-apiserver's
trust in that issuer. Every file it mentions is in [deploy/](../deploy). The
local fixture deploys the same components, so CI runs them on every change.

It takes the opinionated route: Dex as the issuer, Room Pass in front of all of
it, and Traefik at the edge. Other ingress controllers work, as long as you
configure the same routes and protections there; [Sharing an existing
Dex](#sharing-an-existing-dex) covers a Dex you already run.

## How the pieces fit

```
browser ──> app.example.com ──┬── /bind, /join, /logout ──> Room Pass
                              └── everything else ────────> your application
browser ──> login.example.com ───── all of it ────────────> Room Pass ──> Dex
app backend, kube-apiserver ──> login.example.com            (protocol endpoints only)
```

- **Room Pass** checks the room code, records a Participant, and is the only
  thing that asserts an identity to Dex. It answers the whole issuer host and
  forwards only Dex's protocol endpoints (`/auth`, `/token`, `/keys`, ...).
- **Dex** issues the OIDC tokens your application receives. Its `authproxy`
  connector believes the identity headers on its callback; that is why nothing
  but Room Pass may reach it.
- **Your application** is an ordinary OIDC client of Dex.
- **The kube-apiserver** (optional) accepts those tokens, so a participant's
  own token is the credential and RBAC on the Room's group is the decision.

The join page must be on the application's host: Room Pass's cookies are
host-only, and the QR code flow hands a code from the application to the join
page in a cookie ([qr-join.md](qr-join.md)).

## Before you start

| You need | Why |
|---|---|
| Kubernetes 1.30 or later | The CRDs use CEL validation. |
| A CNI that **enforces** NetworkPolicy | Dex's isolation is a NetworkPolicy. Without enforcement it is accepted and does nothing. |
| Traefik (or the equivalent configuration elsewhere) | Routes, a per-source rate limit and header stripping. |
| Two hostnames with TLS | The application's, and the issuer's. |
| The issuer resolvable from inside the cluster **and** from the control plane | Your application's backend exchanges codes there, and the apiserver fetches the issuer's keys from it. A name that only resolves for browsers makes every token fail. |
| Control over the apiserver's flags | Only for participants acting in Kubernetes: `--authentication-config` is a control-plane setting that most managed offerings do not expose. |

Room Pass serves **one Room from one replica**. Do not scale it.

## 1. Choose the values that must agree

Most installation failures are two of these disagreeing. Decide them first.

| Value | Example | Must equal |
|---|---|---|
| Application origin | `https://app.example.com` | `JOIN_ORIGIN`; the host of the redirect URI and return URLs |
| Issuer | `https://login.example.com` | Dex's `issuer`; `ISSUER_ORIGIN`; the apiserver's issuer `url` |
| Client ID | `my-app` | Dex's `staticClients[].id`; the application's client; the apiserver's `audiences` |
| Connector ID | `room-pass` | Dex's connector `id`; Room Pass's `CONNECTOR_ID` (default `room-pass`); the apiserver's rules. Keep the default. |
| Return URLs | `https://app.example.com/` | Each Room's `allowedReturnURLs` must also be in `ALLOWED_RETURN_URLS` |
| Audience group | `demo:my-talk` | The Room's `audienceGroup`; your RoleBindings. Must start with `demo:`. |

## 2. Make the overlay yours

Copy [deploy/example](../deploy/example) into your own repository and replace
its relative resources with a release:

```yaml
resources:
  - https://github.com/sunib/room-pass//deploy/base?ref=vX.Y.Z
  - https://github.com/sunib/room-pass//deploy/dex?ref=vX.Y.Z
  - https://github.com/sunib/room-pass//deploy/edge/traefik?ref=vX.Y.Z
  - ingress.yaml
```

Then edit, in order:

1. [room-pass-environment.yaml](../deploy/example/room-pass-environment.yaml):
   `JOIN_ORIGIN`, `ISSUER_ORIGIN`, `ALLOWED_RETURN_URLS`, and the image (pin a
   digest). Room Pass refuses to start while any of the three is missing.
2. [dex-config.yaml](../deploy/example/dex-config.yaml): the `issuer` and your
   application's client. Keep the connector as it is.
3. [ingress.yaml](../deploy/example/ingress.yaml): both hosts and the TLS
   Secret. Room Pass claims three exact paths on the application's host; your
   application's own Ingress serves the rest of it.
4. [room.yaml](../deploy/example/room.yaml): the event. See [step 5](#5-create-the-room).

## 3. Create the Secrets

Room Pass's cookie keys. Losing or replacing them signs every enrolled browser
out; keep them out of Git (or encrypt them with SOPS or similar) and back them
up with the Room and its Participants:

```sh
kubectl create namespace room-pass
umask 077
openssl rand 32 > hash-key
openssl rand 32 > block-key
kubectl -n room-pass create secret generic room-pass-cookie \
  --from-file=hash-key --from-file=block-key
rm hash-key block-key
```

The client secret, shared by Dex and your application. Dex reads it only when
its pod starts, so restart Dex after changing it (a restart signs nobody out: Dex
keeps its signing keys in Kubernetes):

```sh
kubectl -n room-pass create secret generic dex-clients \
  --from-literal=MY_APP_CLIENT_SECRET="$(openssl rand -hex 32)"
```

## 4. Apply

CRDs first, and wait for them: a Room in the same apply as its CRD fails with
"no matches for kind". Two sets: Room Pass's, and Dex's storage
(`dex.coreos.com`), which Dex is configured not to create itself, so it runs
with a Role in its own namespace instead of the right to create cluster-wide
types. Without them Dex starts, and every login fails.

```sh
kubectl apply -k https://github.com/sunib/room-pass//config/crd?ref=vX.Y.Z
kubectl apply -k https://github.com/sunib/room-pass//deploy/dex-crds?ref=vX.Y.Z
kubectl wait --for=condition=Established \
  crd/rooms.room-pass.koudijs.dev crd/participants.room-pass.koudijs.dev \
  crd/authcodes.dex.coreos.com crd/authrequests.dex.coreos.com \
  crd/connectors.dex.coreos.com crd/devicerequests.dex.coreos.com \
  crd/devicetokens.dex.coreos.com crd/oauth2clients.dex.coreos.com \
  crd/offlinesessionses.dex.coreos.com crd/passwords.dex.coreos.com \
  crd/refreshtokens.dex.coreos.com crd/signingkeies.dex.coreos.com
kubectl apply -k path/to/your/overlay
kubectl -n room-pass rollout status deployment/dex
kubectl -n room-pass rollout status deployment/room-pass
```

Under Flux or Argo CD, put the CRDs in a separate Kustomization or Application
that the rest depends on, and do the same for the Room.

Dex's CRDs are cluster-scoped and shared by every Dex in the cluster that uses
Kubernetes storage; each keeps its resources in its own namespace. If another
Dex (or its chart) already owns them, leave them to it and skip `deploy/dex-crds`,
as long as that Dex is the same release or close to it.

## 5. Create the Room

```sh
kubectl apply -f room.yaml
kubectl -n room-pass get rooms.room-pass.koudijs.dev demo
```

Three fields are immutable: `joinCode`, `audienceGroup` and
`allowedReturnURLs`. Changing one means deleting and recreating the Room, which
deletes its Participants (they are owned by it) and signs every browser out.
GitOps tools that "force" a replacement on an immutable-field error do exactly
that, silently. Settle them before the event.

`spec.endsAt` must be in the future or nobody can join. `joinCode.validFor` may
be at most four `rotateEvery` periods; a slower rotation with the same
`validFor` exposes nothing more, it only redraws the QR code less often.

### Dress the join page for the talk (optional)

`spec.appearance` gives the join page a look for the event without writing any
HTML. Every field is optional, and without any the page looks as it always has.

```yaml
spec:
  title: Everyone gets a vote
  appearance:
    tagline: Platform Day 2027 · Room B · Thursday 14:30   # a line under the title
    picture: /talks/platform-day/logo.png                  # above the title
    pictureAlt: Platform Day 2027                          # empty: decorative
    accentColor: "#f2a541"                                 # buttons and focus rings
    backgroundColor: "#0f3d3e"                             # puts the form on a card
    backgroundImage: /talks/platform-day/stage.jpg         # covers the page
```

- **Pictures are paths on the application's host.** The join page runs there,
  and its CSP loads images from that origin only, so Room Pass needs no change
  for them: serve the files from your application, or from anything your edge
  routes on that host. Serve them without a login, because participants have
  not signed in yet. Absolute URLs and `//host` paths are refused at apply time.
- **Keep the background image small.** The whole room loads it at the same
  moment over the venue's network; 200 to 300 KB of JPEG or WebP is plenty for a
  phone. Pick a `backgroundColor` close to it, because that shows first.
- **Colours are `#rrggbb`.** The button text is white or near-black, whichever
  contrasts more with `accentColor`.
- **Set it before you go on stage.** Like any change to a Room's spec, editing
  it starts a fresh code epoch: the code on the screen stops working, and
  anyone typing it has to type the new one.

The instructions, the issued address and the notice that a name is an
unverified label stay as Room Pass writes them; `attributionNote` remains the
one sentence an operator adds about identity.

### Ask one question at the door (optional)

`spec.question` asks everyone who joins to pick one answer, and puts that
answer's group in their token beside `audienceGroup`. Bind a permission to one
answer's group and only part of the room has it, until you bind it to
`audienceGroup` as well.

```yaml
spec:
  audienceGroup: demo:my-talk
  question:
    prompt: Your favourite frontend framework?
    answers:                          # 2 to 8, shown in this order
      - label: React
        group: demo:framework-react
      - label: Vue
        group: demo:framework-vue
      - label: Angular
        group: demo:framework-angular
      - label: Svelte
        group: demo:framework-svelte
```

Somebody who picks Svelte signs in with the groups `demo:my-talk` and
`demo:framework-svelte`.

- **An answer is the participant's word.** Anyone can pick Svelte, so give an
  answer's group only what anyone in the room could be allowed to have. Ask
  nothing personal: the group travels in the token, into audit events and into
  anything that mirrors them.
- **Answer groups follow `audienceGroup`'s rules**: `demo:`, then letters,
  digits, `:`, `_` and `-`. The apiserver's containment rule (step 7) already
  admits them, so its configuration does not change. Room Pass sends Dex the
  groups comma-separated, the authproxy connector's default
  `groupHeaderSeparator`; leave that unset.
- **The answer is kept on the Participant**, in `spec.groups`, and sent at every
  sign-in. Nobody is asked twice. To move someone, edit their `spec.groups`:
  their next sign-in carries it, while a token already issued keeps its groups.
- **Set it before you go on stage.** Editing it starts a fresh code epoch, and
  anyone who joined before the question existed is not asked it.

## 6. Connect your application

Your application is an OIDC client of the issuer:

- Use the authorization code flow with PKCE, with the client from step 1.
- Request `openid profile email groups federated:id`. Without `federated:id`,
  Dex leaves out the connector the apiserver keys on, and it rejects the token.
- The identity is the token's `sub`. `name` is the display name the participant
  chose, folded to letters, digits and hyphens; `email` is synthetic and never a
  mailbox, although Dex marks it verified.
- For QR joining, the application's login endpoint accepts `code` and `return`
  and hands the code to Room Pass in a cookie: [qr-join.md](qr-join.md).

Authorization is yours. If participants act in Kubernetes, bind permissions to
the Room's group and nothing else:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: my-talk-participants
  namespace: my-app
subjects:
  - kind: Group
    name: demo:my-talk
    apiGroup: rbac.authorization.k8s.io
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: my-talk-participant
```

Room Pass never creates grants.

## 7. Trust the issuer in the apiserver (optional)

Only needed when the application sends a participant's token to Kubernetes.
Start from [deploy/apiserver/authentication-config.yaml](../deploy/apiserver/authentication-config.yaml),
set the issuer and audiences from step 1, and pass it with
`--authentication-config` on every control-plane node.

What it enforces, and why each rule matters:

- **The username comes from the connector**, as `demo:<sub>`. A participant can
  never produce a name in another connector's namespace.
- **The containment rule**: tokens from the `room-pass` connector may only carry
  `demo:` groups. If every control in front of Dex failed, a forged identity
  still could not reach a group your platform binds.
- **Audit extras** carry the display name and address into audit events, for
  attribution only.

Check it with a participant's token:

```sh
kubectl --token="$ID_TOKEN" auth whoami
```

## 8. Verify before the event

From outside the cluster:

```sh
curl -s https://login.example.com/.well-known/openid-configuration | head -c 200
# Only Room Pass's own callback is served; anything else under /callback is refused.
curl -s -o /dev/null -w '%{http_code}\n' https://login.example.com/callback/other  # 403
```

From inside, Dex must refuse every pod but Room Pass:

```sh
kubectl run probe --rm -it --restart=Never --image=curlimages/curl -- \
  curl -s -m 5 http://dex.room-pass.svc:5556/.well-known/openid-configuration
# should time out
```

Then rehearse: join from a phone on mobile data, sign in to the application,
and, if it acts in Kubernetes, make the change it makes.

## Running the event

Project the rotating code as a QR code from a checkout of this repository,
under your own kubeconfig (the code is operator-only):

```sh
task present BASE=https://app.example.com ROOM_NAMESPACE=room-pass NEXT=/
```

Or read it: `kubectl -n room-pass get rooms.room-pass.koudijs.dev demo -o jsonpath='{.status.joinCode.code}'`.

| To | Do |
|---|---|
| Close new joins; enrolled browsers can still sign in | `kubectl -n room-pass patch rooms.room-pass.koudijs.dev demo --type=merge -p '{"spec":{"enrollment":"Closed"}}'` |
| Reopen (a fresh code is published) | the same with `"Open"` |
| Stop the Room for good | the same with `{"spec":{"stopped":true}}`; irreversible |
| Stop tokens already issued from writing | delete the RoleBinding for the Room's group |
| Revoke one participant | `kubectl -n room-pass patch participants.room-pass.koudijs.dev <name> --type=merge -p '{"spec":{"revoked":true}}'`; irreversible |

Stopping a Room prevents new sign-ins. It does **not** revoke ID tokens already
issued; they stay valid until they expire. Removing the grant is what stops
them. If GitOps owns the RoleBinding, suspend it first, or it comes back.

Afterwards, deleting the Room deletes its Participants. That does not erase
what the application, audit logs or any Git mirror recorded under those names.

## Sharing an existing Dex

If Dex already serves operators (for example through a GitHub connector), Room
Pass can add its connector to it instead of fronting a Dex of its own. This
trades a smaller footprint for more controls that must all stay correct:

1. **Routes.** On the issuer host, route exactly `/callback/room-pass` and the
   `/room-pass/` prefix to Room Pass, with an explicitly higher priority than
   the catch-all route to Dex, and strip the `X-Remote-*` headers on them. Do
   not rely on the ingress controller's rule-length heuristics: this split is
   the authentication boundary.
2. **Network.** Dex's NetworkPolicy admits the ingress controller and Room
   Pass's pods, and nothing else. Select every pod in Dex's namespace rather
   than guessing the chart's labels: a selector that matches nothing protects
   nothing, silently. [test/network](../test/network) tests this shape of policy.
3. **Room Pass** points `DEX_UPSTREAM` at that Dex's Service.
4. **The apiserver** keeps the containment rule, and maps each connector to its
   own username prefix (`github:`, `demo:`), as the comments in
   [authentication-config.yaml](../deploy/apiserver/authentication-config.yaml)
   describe. Clients that participants use pass `connector_id=room-pass` to
   Dex's `/auth`, so they skip the connector chooser.

## Upgrading

Pin the CRDs, the components and the image to the same release. A patch or
minor release keeps the identity contract (API group, subject, connector ID,
cookie keys); a major release says in the CHANGELOG what to do.
[README.md](../README.md#upgrading-from-1x) covers 1.x to 2.x.

## When it does not work

| Symptom | Usually |
|---|---|
| Room Pass exits with `JOIN_ORIGIN must be an exact HTTPS origin` | The overlay does not set it, or sets a path. |
| Login works, but the application gets `Unauthorized` from Kubernetes | The client ID is missing from the apiserver's `audiences`, or the client did not request `federated:id`. |
| Every participant token is rejected | The control plane cannot resolve or reach the issuer to fetch its keys. |
| `Unsupported callback` | `CONNECTOR_ID` and Dex's connector `id` differ. |
| Dex stays in `CreateContainerConfigError` | The `dex-clients` Secret does not exist yet. |
| `invalid client` at login | The client secret changed and Dex was not restarted, or the application holds a different value. |
| Nobody can join | `endsAt` has passed, enrollment is Closed, or the Room is stopped: `kubectl -n room-pass get rooms.room-pass.koudijs.dev demo -o yaml`. |
| Dex logs `storage is not initialized, CRDs are not created` | Dex's CRDs are missing. Apply `deploy/dex-crds` and restart Dex. |
| Dex logs `forbidden` for `dex.coreos.com` resources | Dex runs in a namespace other than its RoleBinding's, or without the `dex` ServiceAccount. |
| Everyone was signed out and had to enroll again | The cookie Secret was replaced, or the Room was recreated. |
