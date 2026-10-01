# Where Room Pass came from, and who it is for

Room Pass is room-code sign-in for applications, powered by Dex. This page is
the background a newcomer would otherwise have to reconstruct from the commit
history: why it exists, what it was built for, how it became its own
repository, and which documents here still read as if it had not.

## Built for one talk

Room Pass was written in September 2026 inside
[Voter](https://github.com/sunib/voter), the live demo for a talk at Swiss Cloud
Native Day on **17 September 2026**. The talk's argument was that Kubernetes
can be an application's backend, and that a GitOps audit trail can name the
person behind every change. To show that, the audience had to take part on
their phones, live, and every change they made had to be attributed to them.

That made identity the hard part. The audience could not be asked to create
accounts, and a shared login would have made "who changed this" meaningless. The
answer was a room code on the screen and a display name typed on a phone, turned
into a stable identity that Dex signs and Kubernetes authenticates. RBAC then
decided what each person could do, and the audit trail recorded who did it.

It worked on the day: **66 people signed in within 75 seconds**, and **43 of
them authored 105 commits** in the demo hour. Each commit carried the name the
API server had authenticated, not one the application wrote down.
[Voter's report](https://github.com/sunib/voter/blob/main/docs/demo-2026-09-17-report.md)
has the full numbers, and its
[post-mortem](https://github.com/sunib/voter/blob/main/docs/post-demo-2026-09-17.md)
has what went wrong.

## Why it is its own project

Nothing in the enrollment flow was specific to voting. Any application that
speaks OIDC can sit behind it, and any speaker running an interactive demo has
the same problem the talk had. Keeping it inside Voter would have hidden that,
and tied its releases to an application it does not need.

It was extracted on **30 September 2026**, with the history of its 50 commits
in Voter's `room-pass/` directory. Along the way:

- **Owner.** It lives at `github.com/sunib/room-pass`, a personal account. It is
  a personal project, deliberately not an asset of ConfigButler, the
  organization behind the audit-trail tooling the talk demonstrated.
- **API group.** It moved from `roompass.configbutler.ai` to
  `room-pass.koudijs.dev`, the author's own domain and the project's own name.
- **Version.** Voter released Room Pass as 1.x. The first release from here is
  **2.0.0**: the group change breaks every stored Room and Participant, and the
  cluster that runs Voter follows Room Pass 1.x automatically, so a major is
  what keeps it from deploying itself.
- **Tests.** The specs that exercised Voter (voting, the shared editor, live
  streams, a ballot load test) stayed in Voter. What is here tests Room Pass's
  own contract against a minimal OIDC demo client.

Voter is now one consumer among any others: it uses a released image and the
CRDs, through Dex and OIDC, never through Go imports.

## Who it is for

[PRODUCT-VISION.md](../PRODUCT-VISION.md#who-it-serves) is the fuller answer. In
short:

- **Speakers and workshop leaders** who want a room full of people inside a
  live application in about a minute, and who can run Kubernetes and Dex
  themselves. They are the first users, and the docs should not pretend
  otherwise.
- **Application developers** who want those participants as ordinary OIDC
  users, without writing enrollment.
- **Participants**, who should get in without handing over a personal account,
  and who should be able to see where their chosen name will appear.

It is **not** an identity provider for anything that needs to know who someone
really is. A display name is typed, not verified, and the identity lasts for a
session in one room. It also serves one Room from one replica; more than that
is future work, not a configuration option.

## Documents that predate the extraction

These were written while Room Pass lived in Voter. They are kept because their
reasoning still holds, but they speak from inside that repository:

- [requirements.md](../requirements.md) is the initial implementation contract
  from 2026-09-09. Its examples use the talk's hosts (`demo.configbutler.ai`),
  and its instruction to stay inside Voter belonged to that phase.
- [PRODUCT-VISION.md](../PRODUCT-VISION.md) and
  [OPEN-SOURCE-PLAN.md](../OPEN-SOURCE-PLAN.md) were written to argue for this
  extraction. The plan's status section records what is done and what is open.
- The notes in [docs/](.) are dated findings. Where they name Voter, it was the
  application in the fixture at the time.
- [test/network](../test/network/README.md) checks a snapshot of the Dex
  NetworkPolicy from the cluster the talk ran on, where Room Pass lives in a
  namespace called `voter`.
