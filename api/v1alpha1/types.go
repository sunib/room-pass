// Package v1alpha1 defines the Room Pass enrollment API.
// +kubebuilder:object:generate=true
// +groupName=room-pass.koudijs.dev
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var GroupVersion = schema.GroupVersion{Group: "room-pass.koudijs.dev", Version: "v1alpha1"}

func AddToScheme(s *runtime.Scheme) error {
	s.AddKnownTypes(GroupVersion, &Room{}, &RoomList{}, &Participant{}, &ParticipantList{})
	metav1.AddToGroupVersion(s, GroupVersion)
	return nil
}

// JoinCodeSpec controls rotation and the grace period for typing a previous code.
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="joinCode is immutable"
// +kubebuilder:validation:XValidation:rule="duration(self.validFor) >= duration(self.rotateEvery) + duration('5s') && duration(self.validFor) <= duration(self.rotateEvery) + duration(self.rotateEvery) + duration(self.rotateEvery) + duration(self.rotateEvery)",message="validFor must provide at least five seconds overlap and retain at most four codes"
type JoinCodeSpec struct {
	// +kubebuilder:default="15s"
	// +kubebuilder:validation:Enum="10s";"15s";"30s";"60s"
	RotateEvery string `json:"rotateEvery,omitempty"`
	// +kubebuilder:default="30s"
	// +kubebuilder:validation:Enum="20s";"30s";"45s";"60s";"90s";"120s";"180s";"240s"
	ValidFor string `json:"validFor,omitempty"`
	// +kubebuilder:default=6
	// +kubebuilder:validation:Minimum=6
	// +kubebuilder:validation:Maximum=12
	Length int `json:"length,omitempty"`
}

type RoomSpec struct {
	// Title appears on the participant join page.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=120
	Title string `json:"title"`
	// AttributionNote is the operator's one sentence about what the issued
	// address is for. Room Pass knows the address is synthetic and never a real
	// mailbox; what the application behind it does with the address -- put it on
	// a Git commit, write it to an audit log, nothing at all -- is not Room
	// Pass's to claim. Empty means the join page claims nothing beyond the
	// address being unroutable.
	//
	// It is appended to a sentence that already ends "...it is never a real
	// mailbox", so say only what Room Pass cannot: "It labels your changes in
	// Git." Repeating the mailbox clause reads as a stutter on the page.
	// +kubebuilder:validation:MaxLength=200
	// +kubebuilder:validation:Pattern="^[^<>\\x00-\\x1f\\x7f]*$"
	// +optional
	AttributionNote string `json:"attributionNote,omitempty"`
	// Appearance dresses the join page for this event. Empty, the page looks
	// as it always has.
	//
	// Pictures are paths on the join host, because the join page admits images
	// from its own origin only. Serve them there without a login: participants
	// have not signed in yet.
	//
	// Like any spec change, editing it starts a fresh join-code epoch, which
	// invalidates the code on screen. Set it before the event.
	// +optional
	Appearance *RoomAppearance `json:"appearance,omitempty"`
	// Question asks everyone who joins to pick one answer, and the group of
	// that answer joins the audienceGroup in their token. Empty, nobody is
	// asked and the audienceGroup is their only group, as before.
	//
	// The participant chooses, so an answer is as unverified as a display
	// name: bind its group only to what anyone in the room may choose to have.
	//
	// The answer is kept on the Participant (spec.groups) at enrollment.
	// Changing the question later changes nothing for those already enrolled,
	// and nobody who enrolled before it existed is asked. Like any spec
	// change, editing it starts a fresh join-code epoch. Set it before the
	// event.
	// +optional
	Question *RoomQuestion `json:"question,omitempty"`
	// BrowserGroups adds two groups to everyone who joins, named after the
	// browser and the platform they joined with, such as demo:browser-safari
	// and demo:platform-ios. Empty, nothing is added.
	//
	// Like an answer, they are kept on the Participant (spec.groups) at
	// enrollment and are only what the browser says it is.
	// +optional
	BrowserGroups *BrowserGroups `json:"browserGroups,omitempty"`
	// EndsAt can be extended without changing participant identities.
	EndsAt metav1.Time `json:"endsAt"`
	// +kubebuilder:default=Closed
	// +kubebuilder:validation:Enum=Open;Closed
	Enrollment string `json:"enrollment"`
	// Stopped permanently denies enrollment and new identity assertions.
	// +kubebuilder:default=false
	// +kubebuilder:validation:XValidation:rule="!oldSelf || self",message="stopping is irreversible"
	Stopped bool `json:"stopped"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10000
	MaxParticipants int `json:"maxParticipants"`
	// AudienceGroup selects operator-managed RBAC; Room Pass never creates grants.
	// +kubebuilder:validation:Pattern="^demo:[a-zA-Z0-9][a-zA-Z0-9:_-]*$"
	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="audienceGroup is immutable"
	AudienceGroup string `json:"audienceGroup"`
	// AllowedReturnURLs must also appear in the deployment allowlist.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// +kubebuilder:validation:items:MaxLength=2048
	// +kubebuilder:validation:items:Pattern="^https://[^/?#@]+/[^#]*$"
	// +listType=set
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="allowedReturnURLs is immutable"
	AllowedReturnURLs []string `json:"allowedReturnURLs"`
	// +kubebuilder:default={rotateEvery:"15s",validFor:"30s",length:6}
	JoinCode JoinCodeSpec `json:"joinCode"`
}

// RoomAppearance is display only. It never reaches the subject, the identity
// headers or the groups, and it cannot reword what the join page says about
// identity: the instructions, the issued address and the unverified-label
// notice stay Room Pass's own.
//
// Paths are the only way in for a picture. The join page's CSP is img-src
// 'self', so an absolute URL would be blocked in the browser anyway; the
// pattern refuses it, and a "//host" path, at apply time instead. Colours and
// paths land in CSS and URL contexts, so the patterns are deliberately narrow,
// and internal/server checks them again before rendering.
type RoomAppearance struct {
	// Tagline is one line under the title: the event, the room, the time.
	// +kubebuilder:validation:MaxLength=120
	// +kubebuilder:validation:Pattern="^[^<>\\x00-\\x1f\\x7f]*$"
	// +optional
	Tagline string `json:"tagline,omitempty"`
	// Picture is shown above the title, as a path on the join host such as
	// /talks/my-talk/logo.png.
	// +kubebuilder:validation:MaxLength=512
	// +kubebuilder:validation:Pattern="^/[A-Za-z0-9._~%!$&+,;=:@?-][A-Za-z0-9._~%!$&+,;=:@/?-]*$"
	// +optional
	Picture string `json:"picture,omitempty"`
	// PictureAlt is what a screen reader says for the picture. Empty marks it
	// decorative.
	// +kubebuilder:validation:MaxLength=120
	// +kubebuilder:validation:Pattern="^[^<>\\x00-\\x1f\\x7f]*$"
	// +optional
	PictureAlt string `json:"pictureAlt,omitempty"`
	// AccentColor fills the buttons and outlines focused fields, as #rrggbb.
	// The button text is white or near-black, whichever contrasts more.
	// +kubebuilder:validation:Pattern="^#[0-9a-fA-F]{6}$"
	// +optional
	AccentColor string `json:"accentColor,omitempty"`
	// BackgroundColor is the page behind the form, as #rrggbb. Setting it, or
	// BackgroundImage, puts the form on a white card so it stays readable.
	// +kubebuilder:validation:Pattern="^#[0-9a-fA-F]{6}$"
	// +optional
	BackgroundColor string `json:"backgroundColor,omitempty"`
	// BackgroundImage covers the page behind the form, as a path on the join
	// host. BackgroundColor shows until it arrives. The whole room loads it at
	// once over the venue's network, so keep it small.
	// +kubebuilder:validation:MaxLength=512
	// +kubebuilder:validation:Pattern="^/[A-Za-z0-9._~%!$&+,;=:@?-][A-Za-z0-9._~%!$&+,;=:@/?-]*$"
	// +optional
	BackgroundImage string `json:"backgroundImage,omitempty"`
}

// RoomQuestion is the one question a Room asks at the door. Every answer names
// a group, so a group from this list is all a participant can choose: never a
// username, never a group outside demo:, never one the operator did not write.
type RoomQuestion struct {
	// Prompt is the question, shown above the answers.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=120
	// +kubebuilder:validation:Pattern="^[^<>\\x00-\\x1f\\x7f]*$"
	Prompt string `json:"prompt"`
	// Answers are shown in this order, and every participant picks one.
	// +kubebuilder:validation:MinItems=2
	// +kubebuilder:validation:MaxItems=8
	// +listType=map
	// +listMapKey=group
	// +kubebuilder:validation:XValidation:rule="self.all(a, self.exists_one(b, b.label == a.label))",message="answer labels must be unique"
	Answers []RoomAnswer `json:"answers"`
}

type RoomAnswer struct {
	// Label is what the participant picks, such as Svelte.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=60
	// +kubebuilder:validation:Pattern="^[^<>\\x00-\\x1f\\x7f]*$"
	Label string `json:"label"`
	// Group is what picking it adds to the participant's groups, such as
	// demo:framework-svelte. It follows audienceGroup's rules, so the
	// apiserver's demo:-only containment rule covers it unchanged.
	// +kubebuilder:validation:Pattern="^demo:[a-zA-Z0-9][a-zA-Z0-9:_-]*$"
	// +kubebuilder:validation:MaxLength=128
	Group string `json:"group"`
}

// BrowserGroups names the two groups that say how a participant joined:
// <prefix>browser-<browser> and <prefix>platform-<platform>.
//
// Both names come from fixed lists, whatever the browser sends. Browsers:
// safari, chrome, firefox, edge, samsung, opera, other. Platforms: ios,
// android, macos, windows, chromeos, linux, other. A forged User-Agent can
// only pick another name from them, and anyone can claim to be Safari, so
// bind these groups only to what anyone may have.
//
// There is no phone brand: Chrome on Android no longer sends one. ios is the
// closest there is to Apple.
type BrowserGroups struct {
	// Prefix starts both names: demo: gives demo:browser-safari, and
	// demo:my-talk: gives demo:my-talk:browser-safari.
	// +kubebuilder:validation:MaxLength=100
	// +kubebuilder:validation:Pattern="^demo:([a-zA-Z0-9][a-zA-Z0-9:_-]*)?$"
	Prefix string `json:"prefix"`
}

type Code struct {
	// +kubebuilder:validation:MaxLength=12
	Code      string      `json:"code"`
	IssuedAt  metav1.Time `json:"issuedAt"`
	ExpiresAt metav1.Time `json:"expiresAt"`
}
type RoomStatus struct {
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=8
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	JoinCode   *Code              `json:"joinCode,omitempty"`
	// ValidJoinCodes is operator-only credential material, never a public response.
	// +kubebuilder:validation:MaxItems=4
	ValidJoinCodes   []Code `json:"validJoinCodes,omitempty"`
	ParticipantCount int    `json:"participantCount"`
}

// Code leads the printer columns below, because on stage it is the one thing
// worth reading off a terminal: `kubectl get room` answers "what do I type"
// without a jsonpath. It rotates on spec.joinCode.rotateEvery, so the column is
// expected to differ between two gets -- that is the credential working.
//
// It exposes nothing new. A printer column is a projection of status, not a
// grant: anyone who can already `get rooms` could read .status.joinCode.code,
// and participants have no rooms grant at all. The operator-only material is
// status.validJoinCodes, which stays off the list.
//
// Kept clear of the marker block on purpose -- a comment contiguous with it is
// the type's doc comment, and controller-gen would compile these paragraphs
// into the Room schema's description for every client to download.
//
// Ends is typed string rather than date for a reason that looks like a mistake.
// kubectl renders a date column as an AGE -- time SINCE the value -- and the age
// of a future timestamp is negative, which it formats as the literal
// "<invalid>". A room that has not ended yet is the only case anyone looks at,
// so the date form showed "<invalid>" exactly when it was asked. As a string the
// column prints the timestamp. The date form is right for Created and
// SubmittedAt, which are always in the past; it is wrong for a deadline.

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Code",type=string,JSONPath=`.status.joinCode.code`
// +kubebuilder:printcolumn:name="Enrollment",type=string,JSONPath=`.spec.enrollment`
// +kubebuilder:printcolumn:name="Participants",type=integer,JSONPath=`.status.participantCount`
// +kubebuilder:printcolumn:name="Ends",type=string,JSONPath=`.spec.endsAt`
type Room struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RoomSpec   `json:"spec"`
	Status            RoomStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type RoomList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Room `json:"items"`
}

// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="roomRef is immutable"
type RoomRef struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`
	// UID prevents name reuse from reviving enrollment.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	UID string `json:"uid"`
}
type ParticipantSpec struct {
	RoomRef RoomRef `json:"roomRef"`
	// DisplayName is an unverified demo label, never an authorization key.
	//
	// It is stored in its FOLDED form (see labelName in internal/server): what a
	// participant types is normalized before it reaches here, so this value is
	// legal in all three places it travels to -- a Kubernetes label value on
	// every ballot the voter app writes, one path segment in the mirrored audit
	// trail, and the tail of a submission's object name. The pattern is exactly
	// what the fold emits and is deliberately narrower than a label value: no
	// "_" or ".", because an object name is DNS-1123 and rejects the first.
	//
	// MaxLength matches maxParticipantID rather than a label value's 63, so the
	// display name and the address derived from it share one limit instead of
	// two that disagree past forty characters.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=40
	// +kubebuilder:validation:Pattern="^[A-Za-z0-9]([-A-Za-z0-9]*[A-Za-z0-9])?$"
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="displayName is immutable"
	DisplayName string `json:"displayName"`
	// Groups are this participant's groups beside the Room's audienceGroup:
	// the group of the answer they picked, when the Room asked a question.
	// Room Pass sends them to Dex at every sign-in. Like the display name, they
	// are the participant's choice, never verified.
	//
	// An operator may change them. The next sign-in carries the change; a
	// token already issued keeps the groups it was issued with.
	// +kubebuilder:validation:MaxItems=8
	// +kubebuilder:validation:items:Pattern="^demo:[a-zA-Z0-9][a-zA-Z0-9:_-]*$"
	// +kubebuilder:validation:items:MaxLength=128
	// +listType=set
	// +optional
	Groups []string `json:"groups,omitempty"`
	// +kubebuilder:default=false
	// +kubebuilder:validation:XValidation:rule="!oldSelf || self",message="revocation is irreversible"
	Revoked bool `json:"revoked"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Room",type=string,JSONPath=`.spec.roomRef.name`
// +kubebuilder:printcolumn:name="Revoked",type=boolean,JSONPath=`.spec.revoked`
type Participant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ParticipantSpec `json:"spec"`
}

// +kubebuilder:object:root=true
type ParticipantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Participant `json:"items"`
}
