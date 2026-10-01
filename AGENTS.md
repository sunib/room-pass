# Working in this repository

Room Pass is room-code sign-in for applications, powered by Dex. It was
extracted from [Voter](https://github.com/sunib/voter) on 2026-09-30; read
[docs/origin.md](docs/origin.md) for why it exists and who it is for, and
[README.md](README.md) for how it behaves today.

## Everything runs in the devcontainer, and `task ci` is CI

`.github/workflows/ci.yml` builds the `ci` stage of `.devcontainer/Dockerfile`
and runs `task ci` inside it. The devcontainer is that same image plus editor
tools, so a red CI run reproduces with `task ci` here, with the same tools at
the same versions. Change what CI checks in `Taskfile.yaml`, not in the workflow.

`task ci` creates two disposable k3d clusters on the Docker daemon, one after
the other, and deletes both. For a quicker loop on one area: `task lint`,
`task test`, `task integration`, or `task e2e-up` once and then `task browser`
as often as needed.

**Run browser and e2e changes locally before pushing.** CI takes over ten
minutes, most of it bringing a cluster up. A Playwright locator, a fixture
change or an `up.sh` edit is proven with `task e2e-up && task browser` in
seconds once the cluster is up; pushing to find out is the slow way to learn
the same thing.

## Releases are cut from commit messages

Pull requests are squash-merged, so the PR title becomes the commit, and
release-please reads it:

- `fix:` is a patch, `feat:` a minor, `feat!:` (or a `BREAKING CHANGE:` footer)
  a major.
- `build:`, `ci:`, `test:`, `chore:`, `docs:` and `refactor:` cut nothing on
  their own.

`ghcr.io/sunib/room-pass` is followed by a real cluster (k8s.koudijs.dev) inside
one major range. A patch or minor on that major **deploys itself** there; a
major never does. So a breaking change must be `feat!:`, and a `fix:` must be
safe to roll out unattended.

## The identity contract is what consumers depend on

These are visible to every application and audit trail behind Room Pass. Do not
change them without a major release and a migration note:

- the API group `room-pass.koudijs.dev`, and the Room and Participant schemas;
- the subject, derived from the display name (`participantID`), so re-enrolling
  under the same name gives the same identity;
- the Dex connector ID `room-pass`, the identity headers sent to Dex's authproxy
  connector, and the cookie-key Secret name pinned in RBAC.

Known Voter-era assumptions that should become configuration, not grow:
`demo:` group validation and the `@koudijs.dev.test` address suffix. See
[OPEN-SOURCE-PLAN.md](OPEN-SOURCE-PLAN.md).

One Room, one replica, `Recreate`. Do not scale it or add a second enrollment
writer; that needs a design, not a replica count.

## Generated files

`config/crd/` (the CRDs in `bases/` and the `kustomization.yaml` that lists
them) and `api/v1alpha1/zz_generated.deepcopy.go` come from `task generate`.
Never edit them by hand; CI's `task verify-generate` fails when they are stale.

## Tool versions

Every tool in `.devcontainer/Dockerfile` is a pinned release asset checked
against a SHA-256 written beside its version. No install scripts, no vendor apt
repositories. To bump one: change the version, download the asset, compare its
hash with the checksum file the project publishes, then change the hash.
Dependabot moves Go modules, the Playwright package, actions and `FROM` lines,
grouped into one pull request a week; it cannot see the `ENV` pins.

## Pushing workflow changes

GitHub refuses a push that touches `.github/workflows/` unless the credential
has the `workflow` scope. If `git push` is rejected for that reason while
`gh auth status` shows the scope, git is using a different credential (for
example one forwarded by the editor); push once through gh:

```bash
git -c credential.helper= -c 'credential.helper=!gh auth git-credential' push
```

Commits are signed with an SSH key from the forwarded agent; a commit that
seems to hang is usually waiting for the agent to approve the signature.

## Not from here

Room Pass's consumers live elsewhere. Voter (`sunib/voter`) and the cluster's
GitOps repository are changed in their own repositories, never from this one.
