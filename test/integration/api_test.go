package integration

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	api "github.com/sunib/room-pass/api/v1alpha1"
	"github.com/sunib/room-pass/internal/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestAPISchemaAndReconcile(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("set KUBEBUILDER_ASSETS or run task integration")
	}
	env := &envtest.Environment{CRDDirectoryPaths: []string{"../../config/crd/bases"}, ErrorIfCRDPathMissing: true}
	cfg, e := env.Start()
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := env.Stop(); e != nil {
			t.Error(e)
		}
	}()
	scheme := runtime.NewScheme()
	_ = api.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	db, e := client.New(cfg, client.Options{Scheme: scheme})
	if e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	if e = db.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "room-pass"}}); e != nil {
		t.Fatal(e)
	}
	room := &api.Room{ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "room-pass"}, Spec: api.RoomSpec{Title: "Demo", EndsAt: metav1.NewTime(time.Now().Add(time.Hour)), MaxParticipants: 3, AudienceGroup: "demo:test", AllowedReturnURLs: []string{"https://demo.test/app/"}, Enrollment: "Open"}}
	if e = db.Create(ctx, room); e != nil {
		t.Fatal(e)
	}
	key := client.ObjectKeyFromObject(room)
	if e = db.Get(ctx, key, room); e != nil {
		t.Fatal(e)
	}
	if room.Spec.JoinCode.ValidFor != "30s" {
		t.Fatalf("defaults missing: %+v", room.Spec.JoinCode)
	}
	for _, mutate := range []func(*api.Room){func(r *api.Room) { r.Spec.Enrollment = "Maybe" }, func(r *api.Room) { r.Spec.AudienceGroup = "system:masters" }, func(r *api.Room) { r.Spec.AllowedReturnURLs = []string{"https://evil.test/"} }, func(r *api.Room) { r.Spec.JoinCode.ValidFor = "60s" }, func(r *api.Room) { r.Spec.AttributionNote = "<b>bold</b>" }, func(r *api.Room) { r.Spec.AttributionNote = strings.Repeat("x", 201) },
		// Appearance lands in CSS and URL contexts on the join page, and its
		// pictures may come from the join host only.
		func(r *api.Room) { r.Spec.Appearance = &api.RoomAppearance{AccentColor: "red"} },
		func(r *api.Room) { r.Spec.Appearance = &api.RoomAppearance{BackgroundColor: "#fff;x:y"} },
		func(r *api.Room) { r.Spec.Appearance = &api.RoomAppearance{Picture: "https://evil.test/logo.png"} },
		func(r *api.Room) { r.Spec.Appearance = &api.RoomAppearance{Picture: "//evil.test/logo.png"} },
		func(r *api.Room) { r.Spec.Appearance = &api.RoomAppearance{BackgroundImage: `/a")`} },
		func(r *api.Room) { r.Spec.Appearance = &api.RoomAppearance{Tagline: "<b>bold</b>"} },
		func(r *api.Room) { r.Spec.Appearance = &api.RoomAppearance{PictureAlt: strings.Repeat("x", 121)} },
		// A question's answers are groups, so they follow audienceGroup's rules.
		func(r *api.Room) { r.Spec.Question = question(answer("A", "system:masters"), answer("B", "demo:b")) },
		func(r *api.Room) {
			r.Spec.Question = question(answer("A", "demo:a,system:masters"), answer("B", "demo:b"))
		},
		func(r *api.Room) { r.Spec.Question = question(answer("A", "demo:a")) },
		func(r *api.Room) { r.Spec.Question = question(answer("A", "demo:a"), answer("B", "demo:a")) },
		func(r *api.Room) { r.Spec.Question = question(answer("Same", "demo:a"), answer("Same", "demo:b")) },
		func(r *api.Room) { r.Spec.Question = question(answer("<b>A</b>", "demo:a"), answer("B", "demo:b")) },
		func(r *api.Room) {
			r.Spec.Question = question(answer("A", "demo:a"), answer("B", "demo:b"))
			r.Spec.Question.Prompt = ""
		},
		// The browser groups' prefix must leave every name a demo: group.
		func(r *api.Room) { r.Spec.BrowserGroups = &api.BrowserGroups{Prefix: "system:"} },
		func(r *api.Room) { r.Spec.BrowserGroups = &api.BrowserGroups{Prefix: "demo:a,system:"} },
		func(r *api.Room) { r.Spec.BrowserGroups = &api.BrowserGroups{} }} {
		bad := room.DeepCopy()
		mutate(bad)
		if db.Update(ctx, bad) == nil {
			t.Fatal("invalid update accepted")
		}
	}
	room.Spec.AttributionNote = "It labels your changes in Git."
	if e = db.Update(ctx, room); e != nil {
		t.Fatalf("attributionNote rejected: %v", e)
	}
	if e = db.Get(ctx, key, room); e != nil || room.Spec.AttributionNote != "It labels your changes in Git." {
		t.Fatalf("attributionNote did not round-trip: %q %v", room.Spec.AttributionNote, e)
	}
	look := &api.RoomAppearance{Tagline: "Platform Day · Room B", Picture: "/talks/logo.png?v=2", PictureAlt: "Platform Day", AccentColor: "#F2A541", BackgroundColor: "#0f3d3e", BackgroundImage: "/talks/stage.jpg"}
	room.Spec.Appearance = look
	if e = db.Update(ctx, room); e != nil {
		t.Fatalf("appearance rejected: %v", e)
	}
	if e = db.Get(ctx, key, room); e != nil || room.Spec.Appearance == nil || *room.Spec.Appearance != *look {
		t.Fatalf("appearance did not round-trip: %+v %v", room.Spec.Appearance, e)
	}
	// Mutable: the next talk in the same Room brings its own.
	room.Spec.Appearance = nil
	if e = db.Update(ctx, room); e != nil {
		t.Fatalf("appearance cannot be removed: %v", e)
	}
	asked := question(answer("React", "demo:framework-react"), answer("Svelte", "demo:framework-svelte"))
	room.Spec.Question = asked
	if e = db.Update(ctx, room); e != nil {
		t.Fatalf("question rejected: %v", e)
	}
	if e = db.Get(ctx, key, room); e != nil || room.Spec.Question == nil || !reflect.DeepEqual(*room.Spec.Question, *asked) {
		t.Fatalf("question did not round-trip: %+v %v", room.Spec.Question, e)
	}
	for _, prefix := range []string{"demo:", "demo:my-talk:"} {
		room.Spec.BrowserGroups = &api.BrowserGroups{Prefix: prefix}
		if e = db.Update(ctx, room); e != nil {
			t.Fatalf("browserGroups prefix %q rejected: %v", prefix, e)
		}
	}
	rec := &controller.Reconciler{Client: db, Room: key}
	if _, e = rec.Reconcile(ctx, ctrl.Request{NamespacedName: key}); e != nil {
		t.Fatal(e)
	}
	_ = db.Get(ctx, key, room)
	if room.Status.JoinCode == nil {
		t.Fatal("code not persisted")
	}
	code := room.Status.JoinCode.Code
	rv := room.ResourceVersion
	if _, e = rec.Reconcile(ctx, ctrl.Request{NamespacedName: key}); e != nil {
		t.Fatal(e)
	}
	_ = db.Get(ctx, key, room)
	if room.ResourceVersion != rv || room.Status.JoinCode.Code != code {
		t.Fatal("reconcile not idempotent")
	}
	room.Spec.Stopped = true
	if e = db.Update(ctx, room); e != nil {
		t.Fatal(e)
	}
	room.Spec.Stopped = false
	if db.Update(ctx, room) == nil {
		t.Fatal("stop reversed")
	}
	// A Participant's groups follow the same rules as the answers they come from.
	for _, groups := range [][]string{{"system:masters"}, {"demo:a,system:masters"}, {"demo:a", "demo:a"}} {
		bad := &api.Participant{ObjectMeta: metav1.ObjectMeta{Name: "p-bad", Namespace: room.Namespace}, Spec: api.ParticipantSpec{RoomRef: api.RoomRef{Name: room.Name, UID: string(room.UID)}, DisplayName: "Bad", Groups: groups}}
		if db.Create(ctx, bad) == nil {
			t.Fatalf("invalid groups accepted: %v", groups)
		}
	}
	p := &api.Participant{ObjectMeta: metav1.ObjectMeta{Name: "p-test", Namespace: room.Namespace}, Spec: api.ParticipantSpec{RoomRef: api.RoomRef{Name: room.Name, UID: string(room.UID)}, DisplayName: "Ada", Groups: []string{"demo:framework-svelte"}}}
	if e = db.Create(ctx, p); e != nil {
		t.Fatal(e)
	}
	// An operator may move a participant to another group; their next sign-in
	// carries it.
	p.Spec.Groups = []string{"demo:framework-react"}
	if e = db.Update(ctx, p); e != nil {
		t.Fatalf("groups are not editable: %v", e)
	}
	p.Spec.Revoked = true
	if e = db.Update(ctx, p); e != nil {
		t.Fatal(e)
	}
	p.Spec.Revoked = false
	if db.Update(ctx, p) == nil {
		t.Fatal("revocation reversed")
	}
}

func question(answers ...api.RoomAnswer) *api.RoomQuestion {
	return &api.RoomQuestion{Prompt: "Your favourite frontend framework?", Answers: answers}
}

func answer(label, group string) api.RoomAnswer { return api.RoomAnswer{Label: label, Group: group} }
