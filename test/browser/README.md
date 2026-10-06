# Browser authentication tests

```bash
task e2e-up
task browser
```

This runs real Chromium through Traefik, Room Pass, Dex and the local OIDC demo
client. The tests cover new enrollment, returning enrollment with the same
Participant UID, invalid codes, tampered CSRF forms, secure enrollment-cookie flags, closed enrollment,
a dressed Room whose pictures load from the join host under its CSP, and a Room's
question whose answer arrives in the token's groups and is accepted by Kubernetes, and
browser groups read from an iPhone's User-Agent. A phone-sized viewport
exercises the room form without mocking browser headers or the login protocol.

The test runner reads only `.local/kubeconfig`, never the current user
context. Kubernetes supplies the rotating code and verifies enrollment records.
Tests delete only Participants with their unique test display name. The enrollment-closure
test restores the prior enrollment setting in a finally block. Do not run
concurrently with the other fixture suites or against a shared presentation room.

The local fixture uses self-signed TLS and synthetic identities. Chromium ignores
that certificate only for this test configuration; host resolver rules map the two
fixture hosts to the isolated Docker gateway. No external provider credentials are
needed. The task installs Chromium and its OS libraries; the latter may use sudo.

Videos are retained for every test. The successful login test also saves
`logged-in.png`. Inspect `test-results/` or run `npx playwright show-report` from
this directory. Failure traces can be opened with `npx playwright show-trace`.
Artifacts are gitignored and contain local test sessions; treat them as sensitive
if adapting the tests for a different environment.

It does not claim production capacity or that GitHub/LinkedIn login works. CI
builds the local fixture, runs these tests and retains recordings/reports for seven
days. Metrics unit tests and the independent Dex network suite are also gates.

Metrics assertions use the authenticated Kubernetes pod proxy to port 9090. The
browser also verifies that the application origin returns 404 for `/metrics`.
