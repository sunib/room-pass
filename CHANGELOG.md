# Changelog

## [2.1.0](https://github.com/sunib/room-pass/compare/v2.0.0...v2.1.0) (2026-10-06)


### Features

* a reference deployment of Room Pass, Dex, edge and apiserver config that CI runs ([#18](https://github.com/sunib/room-pass/issues/18)) ([9ca3062](https://github.com/sunib/room-pass/commit/9ca3062f3a0001d6dca1454cd73815279a5dc201))
* a Room can dress its join page with a tagline, picture, accent and background ([#25](https://github.com/sunib/room-pass/issues/25)) ([8413d57](https://github.com/sunib/room-pass/commit/8413d57050b70a5cd011f8d138a61a0dafcb8eca))
* Dex keeps its state in Kubernetes, with its CRDs in deploy/dex-crds ([#23](https://github.com/sunib/room-pass/issues/23)) ([417080e](https://github.com/sunib/room-pass/commit/417080ed22eea1e62452e2484e8a8e11314fd91d))


### Bug Fixes

* config/crd and deploy/base can be used as remote kustomize bases ([#14](https://github.com/sunib/room-pass/issues/14)) ([c62a04c](https://github.com/sunib/room-pass/commit/c62a04c87945bb18ff077c37279b239cd51a26a3))
* deploy/dex keeps its state in memory and needs no volume ([#22](https://github.com/sunib/room-pass/issues/22)) ([6e20196](https://github.com/sunib/room-pass/commit/6e20196ada1dba0a2cbb78a1e5e153a6e1e82c74))


### Documentation

* an installation guide, from hosts to the event runbook ([#19](https://github.com/sunib/room-pass/issues/19)) ([c514cd1](https://github.com/sunib/room-pass/commit/c514cd1bd33ee27d15d76d86733da09af441ff2e))
* the plan's deployment gap is closed by deploy/ and the install guide ([#20](https://github.com/sunib/room-pass/issues/20)) ([9247859](https://github.com/sunib/room-pass/commit/924785975c5262b7e4a259b4e2276605f8be0ff0))
* what happens to the 1.x CRDs after upgrading to 2.x ([#15](https://github.com/sunib/room-pass/issues/15)) ([bbe4305](https://github.com/sunib/room-pass/commit/bbe4305f720b907858ccb4bdd263be08fcdb0ad8))

## [2.0.0](https://github.com/sunib/room-pass/tree/v2.0.0) (2026-10-01)

The first release from this repository. Room Pass 1.x was released from [Voter](https://github.com/sunib/voter/releases/tag/v1.1.0).

### ⚠ BREAKING CHANGES

* the Room and Participant CRDs are rooms.room-pass.koudijs.dev and participants.room-pass.koudijs.dev; in 1.x they were in roompass.configbutler.ai. Existing roompass.configbutler.ai objects are not read by this version. Install the new CRDs, recreate the Room under the new group, and update RBAC rules that name the old group. Participants re-enroll; their identity is derived from the display name, so a returning participant who enters the same name gets the same subject. The cookie-key Secret, connector ID and subject format are unchanged.

### Features

* move to github.com/sunib/room-pass, with its own Go module (was github.com/sunib/voter/room-pass) ([ab02e42](https://github.com/sunib/room-pass/commit/ab02e42d4db963a88e7947a5d0c8818e2b07f367))
* the API group is room-pass.koudijs.dev, and the fixture hosts are *.room-pass.test ([#10](https://github.com/sunib/room-pass/issues/10)) ([28d18b1](https://github.com/sunib/room-pass/commit/28d18b190d8af3165f55819272d18edba4901842))


### Documentation

* this is the Room Pass repository now ([6360075](https://github.com/sunib/room-pass/commit/6360075bc83bc47e1376416d8e810360038886f5))
* where Room Pass came from, and how to work in it without Voter's context ([#12](https://github.com/sunib/room-pass/issues/12)) ([9125fd7](https://github.com/sunib/room-pass/commit/9125fd76e01e793508a28f12f84020f5cb37a4b7))
