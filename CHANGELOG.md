# Changelog

## [2.0.0](https://github.com/sunib/room-pass/compare/v1.1.0...v2.0.0) (2026-10-01)


### ⚠ BREAKING CHANGES

* the Room and Participant CRDs are rooms.room-pass.koudijs.dev and participants.room-pass.koudijs.dev. This replaces roompass.koudijs.dev from earlier in this release; coming from Room Pass 1.x, the group changes from roompass.configbutler.ai to room-pass.koudijs.dev.
* the Room and Participant CRDs are now rooms.roompass.koudijs.dev and participants.roompass.koudijs.dev. Existing roompass.configbutler.ai objects are not read by this version. Install the new CRDs, recreate the Room under the new group, and update RBAC rules that name the old group. Participants re-enroll; their identity is derived from the display name, so a returning participant who enters the same name gets the same subject. The cookie-key Secret, connector ID and subject format are unchanged.

### Features

* move to github.com/sunib/room-pass and the roompass.koudijs.dev API group ([ab02e42](https://github.com/sunib/room-pass/commit/ab02e42d4db963a88e7947a5d0c8818e2b07f367))
* the API group is room-pass.koudijs.dev, and the fixture hosts are *.room-pass.test ([#10](https://github.com/sunib/room-pass/issues/10)) ([28d18b1](https://github.com/sunib/room-pass/commit/28d18b190d8af3165f55819272d18edba4901842))


### Documentation

* this is the Room Pass repository now ([6360075](https://github.com/sunib/room-pass/commit/6360075bc83bc47e1376416d8e810360038886f5))
* where Room Pass came from, and how to work in it without Voter's context ([#12](https://github.com/sunib/room-pass/issues/12)) ([9125fd7](https://github.com/sunib/room-pass/commit/9125fd76e01e793508a28f12f84020f5cb37a4b7))
