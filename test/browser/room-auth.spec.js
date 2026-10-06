import { test, expect } from "@playwright/test";
import { execFileSync } from "node:child_process";
import { resolve } from "node:path";
const kubeconfig = resolve("../../.local/kubeconfig");
const kube = (...args) =>
  execFileSync("kubectl", ["--kubeconfig", kubeconfig, ...args], {
    encoding: "utf8",
  });
const room = () =>
  JSON.parse(kube("-n", "room-pass", "get", "room", "demo", "-o", "json"));
const participants = () =>
  JSON.parse(kube("-n", "room-pass", "get", "participants", "-o", "json"))
    .items;
const patchRoom = (spec) =>
  kube(
    "-n",
    "room-pass",
    "patch",
    "room",
    "demo",
    "--type=merge",
    "-p",
    JSON.stringify({ spec }),
  );
// Any spec change starts a fresh code epoch: the controller clears every code
// and issues a new one only once it has seen the change. A test that changed
// the Room waits for that before going on, or before handing the Room back to
// a test that reads its code straight away.
const roomSettled = () =>
  expect
    .poll(() => {
      const r = room();
      return (
        r.status.observedGeneration === r.metadata.generation &&
        (r.spec.enrollment !== "Open" || Boolean(r.status.joinCode?.code))
      );
    })
    .toBe(true);
// What the participant TYPES. Room Pass stores the folded form -- spaces become
// dashes -- because that value has to be legal as a Kubernetes label value, a
// path segment in the mirrored audit trail, and the tail of an object name.
// Keeping both here is the point: the gap between them is the fold, and this is
// the only place it runs in a real browser against the real server.
const testName = `Browser test ${Date.now()}`;
const storedName = testName.replaceAll(" ", "-");

test.afterEach(() => {
  for (const p of participants().filter(
    (p) => p.spec.displayName === storedName,
  ))
    kube("-n", "room-pass", "delete", "participant", p.metadata.name);
});

test("room login and returning enrollment work in a phone-sized browser", async ({
  page,
  context,
}, testInfo) => {
  await page.goto("/app/login");
  await expect(page.getByLabel("Room code")).toBeVisible();
  await page.getByLabel("Room code").fill(room().status.joinCode.code);
  // The address is derived in Go and previewed in the page's own script. This
  // is the only place both halves run at once, so the preview is read from a
  // real browser and compared to the object the server goes on to create.
  const preview = page.locator("#rp-email");
  await expect(preview).toHaveText("your-name@koudijs.dev.test");
  await page.getByLabel("Display name").fill(testName);
  const previewed = await preview.textContent();
  expect(previewed).toBe(`${storedName.toLowerCase()}@koudijs.dev.test`);
  // The name preview is the other half of the same fold, and the half a
  // participant actually reads: it is what they will be called in a Git commit
  // they cannot edit, shown before they commit to it.
  await expect(page.locator("#rp-display")).toHaveText(storedName);
  await page.getByRole("button", { name: "Continue", exact: true }).click();
  await expect(
    page.getByText(`Welcome, ${storedName}.`, { exact: true }),
  ).toBeVisible();
  const cookies = await context.cookies();
  const enrollment = cookies.find((c) => c.name === "__Host-rp-session");
  expect(enrollment).toMatchObject({
    secure: true,
    httpOnly: true,
    path: "/",
    sameSite: "Lax",
  });
  const before = participants().filter(
    (p) => p.spec.displayName === storedName,
  );
  expect(before).toHaveLength(1);
  expect(`${before[0].metadata.name.replace(/^p-/, "")}@koudijs.dev.test`).toBe(
    previewed,
  );
  await page.screenshot({
    path: testInfo.outputPath("logged-in.png"),
    fullPage: true,
  });
  await page.getByRole("link", { name: "Sign in again" }).click();
  await expect(
    page.getByText(`You’re already enrolled as ${storedName}.`, {
      exact: false,
    }),
  ).toBeVisible();
  await expect(page.getByLabel("Room code")).toHaveCount(0);
  await page.getByRole("button", { name: "Continue", exact: true }).click();
  await expect(
    page.getByText(`Welcome, ${storedName}.`, { exact: true }),
  ).toBeVisible();
  const after = participants().filter((p) => p.spec.displayName === storedName);
  expect(after.map((p) => p.metadata.uid)).toEqual(
    before.map((p) => p.metadata.uid),
  );
});

test("invalid room code does not enroll and is corrected in place", async ({
  page,
}) => {
  await page.goto("/app/login");
  await page.getByLabel("Room code").fill("INVALID");
  await page.getByLabel("Display name").fill(testName);
  await page.getByRole("button", { name: "Continue", exact: true }).click();
  await expect(
    page.getByText("That code is invalid or joining has closed.", {
      exact: false,
    }),
  ).toBeVisible();
  expect(
    participants().filter((p) => p.spec.displayName === storedName),
  ).toHaveLength(0);
  // The form stays put with the code marked and the name kept, so a mistyped
  // code is retyped rather than started over. What comes back is the FOLDED
  // name, which is also the first place a participant sees the fold applied.
  await expect(page.getByLabel("Room code")).toHaveAttribute(
    "aria-invalid",
    "true",
  );
  await expect(page.getByLabel("Display name")).toHaveValue(storedName);
  await page.getByLabel("Room code").fill(room().status.joinCode.code);
  await page.getByRole("button", { name: "Continue", exact: true }).click();
  await expect(
    page.getByText(`Welcome, ${storedName}.`, { exact: true }),
  ).toBeVisible();
  expect(
    participants().filter((p) => p.spec.displayName === storedName),
  ).toHaveLength(1);
});

test("a tampered form CSRF token does not enroll", async ({ page }) => {
  await page.goto("/app/login");
  // Read once. The code rotates, and the page is expected to give back what was
  // TYPED -- comparing it with a second read fails whenever a rotation lands in
  // between, which is a test bug, not a server one. The typed code stays
  // acceptable for the second press because the Room keeps recent codes valid.
  const code = room().status.joinCode.code;
  await page.getByLabel("Room code").fill(code);
  await page.getByLabel("Display name").fill(testName);
  await page
    .locator('input[name="csrf"]')
    .first()
    .evaluate((input) => {
      input.value = "forged";
    });
  await page.getByRole("button", { name: "Continue", exact: true }).click();
  await expect(
    page.getByText(
      "This page had been open too long. Your answers are still here; please press Continue again.",
      { exact: true },
    ),
  ).toBeVisible();
  expect(
    participants().filter((p) => p.spec.displayName === storedName),
  ).toHaveLength(0);
  // Refused, but still the login page. A token that expired while the form sat
  // open looks exactly like this, and telling that participant to reload -- and
  // to retype a room code they can no longer see -- is the wrong answer to much
  // the commoner cause. The page comes back with a fresh token and everything
  // they typed, so one more press finishes the join.
  await expect(page.getByLabel("Room code")).toHaveValue(code);
  await expect(page.getByLabel("Display name")).toHaveValue(testName);
  await page.getByRole("button", { name: "Continue", exact: true }).click();
  await expect(
    page.getByText(`Welcome, ${storedName}.`, { exact: true }),
  ).toBeVisible();
  expect(
    participants().filter((p) => p.spec.displayName === storedName),
  ).toHaveLength(1);
});

test("closing enrollment removes the browser join form", async ({ page }) => {
  const enrollment = room().spec.enrollment;
  try {
    patchRoom({ enrollment: "Closed" });
    await page.goto("/app/login");
    await expect(
      page.getByText(
        "Joining is closed. If you already joined, return to the application.",
        { exact: true },
      ),
    ).toBeVisible();
    await expect(page.getByLabel("Room code")).toHaveCount(0);
    await expect(
      page.getByRole("button", { name: "Continue", exact: true }),
    ).toHaveCount(0);
  } finally {
    patchRoom({ enrollment });
    await roomSettled();
  }
});

// A dressed Room, in the browser that enforces the join page's CSP. The
// pictures come from the application on the join host, which is why the policy
// needed no change for them; one it blocked would load nothing and say so only
// on the console, so both are checked.
test("a Room's appearance dresses the join page with pictures from the join host", async ({
  page,
}, testInfo) => {
  const blocked = [];
  page.on("console", (m) => {
    if (/Content Security Policy/i.test(m.text())) blocked.push(m.text());
  });
  try {
    patchRoom({
      appearance: {
        tagline: "Platform Day 2027 · Room B",
        picture: "/app/talk/logo.png",
        pictureAlt: "Platform Day 2027",
        accentColor: "#f2a541",
        backgroundColor: "#0f3d3e",
        backgroundImage: "/app/talk/stage.png",
      },
    });
    await roomSettled();
    await page.goto("/app/login");
    await expect(
      page.getByText("Platform Day 2027 · Room B", { exact: true }),
    ).toBeVisible();
    const loaded = (img) =>
      expect.poll(() => img.evaluate((i) => i.complete && i.naturalWidth));
    await loaded(page.getByRole("img", { name: "Platform Day 2027" })).toBe(
      240,
    );
    await loaded(page.locator("img.backdrop-image")).toBe(120);
    // Amber is too light for white text; Room Pass picks near-black.
    const button = page.getByRole("button", { name: "Continue", exact: true });
    await expect(button).toHaveCSS("background-color", "rgb(242, 165, 65)");
    await expect(button).toHaveCSS("color", "rgb(17, 24, 39)");
    await expect(page.locator("main")).toHaveCSS(
      "background-color",
      "rgb(255, 255, 255)",
    );
    await page.screenshot({
      path: testInfo.outputPath("dressed.png"),
      fullPage: true,
    });
    expect(blocked).toEqual([]);
  } finally {
    patchRoom({ appearance: null });
    await roomSettled();
  }
});

// Scanning the presenter's QR code should leave one field to fill in. The
// interesting part is not the form -- it is that the room code survives a
// redirect chain that leaves this origin for Dex and comes back, carried by a
// cookie the application set before any of that happened. Only a real browser
// enforces the SameSite rule that makes or breaks it, which is why this test
// is here and not in the Go suite.
test("a scanned QR code joins the room without typing a code", async ({
  page,
  context,
}, testInfo) => {
  const code = room().status.joinCode.code;

  // What the QR code encodes: the application's login URL, carrying the code
  // on screen. The real one also carries ?return=, which this demo client has
  // no pages to return to.
  await page.goto(`/app/login?code=${code}`);

  await expect(page.getByText("Choose a display name to join.")).toBeVisible();
  await expect(page.getByLabel("Room code")).toHaveCount(0);
  await expect(page.getByText(code, { exact: false })).toBeVisible();
  await page.screenshot({
    path: testInfo.outputPath("scanned.png"),
    fullPage: true,
  });

  await page.getByLabel("Display name").fill(testName);
  await page.getByRole("button", { name: "Continue", exact: true }).click();
  await expect(
    page.getByText(`Welcome, ${storedName}.`, { exact: true }),
  ).toBeVisible();
  expect(
    participants().filter((p) => p.spec.displayName === storedName),
  ).toHaveLength(1);

  // Used once. A code left in the jar is one that gets silently replayed on
  // the next join, long after it stopped being the code on the screen.
  expect(
    (await context.cookies()).find(
      (c) => c.name === "__Host-room-pass-joincode",
    ),
  ).toBeUndefined();
});

// A cookie anything on this host can write must be worth nothing on its own.
test("a forged join code is still just a wrong code", async ({ page }) => {
  await page.goto("/app/login?code=ZZZZZZ");
  await expect(page.getByLabel("Room code")).toHaveCount(0);
  await page.getByLabel("Display name").fill(testName);
  await page.getByRole("button", { name: "Continue", exact: true }).click();
  await expect(page.getByText(/invalid or joining has closed/i)).toBeVisible();
  expect(
    participants().filter((p) => p.spec.displayName === storedName),
  ).toHaveLength(0);
});

// A Room's question, end to end: the browser will not join without an answer,
// the answer is kept on the Participant, Dex splits the group header into the
// token's groups, and the apiserver accepts a token carrying both. Each of those
// is a different program, and this is the only place they all run together.
test("a Room's question adds the picked answer to the token's groups", async ({
  page,
}, testInfo) => {
  try {
    patchRoom({
      question: {
        prompt: "Your favourite frontend framework?",
        answers: [
          { label: "React", group: "demo:framework-react" },
          { label: "Vue", group: "demo:framework-vue" },
          { label: "Svelte", group: "demo:framework-svelte" },
        ],
      },
    });
    await roomSettled();
    await page.goto("/app/login");
    const question = page.getByRole("group", {
      name: "Your favourite frontend framework?",
    });
    await expect(question.getByRole("radio")).toHaveCount(3);
    await page.getByLabel("Room code").fill(room().status.joinCode.code);
    await page.getByLabel("Display name").fill(testName);
    // Nothing picked: the browser holds the form back.
    await page.getByRole("button", { name: "Continue", exact: true }).click();
    expect(
      await question
        .getByRole("radio", { name: "React" })
        .evaluate((r) => r.validity.valueMissing),
    ).toBe(true);
    expect(
      participants().filter((p) => p.spec.displayName === storedName),
    ).toHaveLength(0);
    await question.getByRole("radio", { name: "Svelte" }).check();
    await page.screenshot({
      path: testInfo.outputPath("question.png"),
      fullPage: true,
    });
    await page.getByRole("button", { name: "Continue", exact: true }).click();
    await expect(
      page.getByText(`Welcome, ${storedName}.`, { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText(
        "Your groups: demo:room-pass-test, demo:framework-svelte",
        { exact: true },
      ),
    ).toBeVisible();
    const [p] = participants().filter((p) => p.spec.displayName === storedName);
    expect(p.spec.groups).toEqual(["demo:framework-svelte"]);
    await page
      .getByRole("button", { name: "Write a message to Kubernetes" })
      .click();
    await expect(page.getByText(/^Kubernetes returned 201/)).toBeVisible();
    // Returning, nobody is asked twice; the page says what they picked.
    await page.getByRole("link", { name: "Sign in again" }).click();
    await expect(
      page.getByText(
        `You’re already enrolled as ${storedName}, and you picked Svelte.`,
        { exact: false },
      ),
    ).toBeVisible();
    await expect(page.getByRole("radio")).toHaveCount(0);
  } finally {
    patchRoom({ question: null });
    await roomSettled();
  }
});

// The browser groups, from a User-Agent this Chromium sends as an iPhone's. The
// page says what will be shared before anyone presses Continue, and the token
// Dex issues carries both names.
test.describe("on an iPhone", () => {
  test.use({
    userAgent:
      "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
  });
  test("a Room's browser groups name the browser in the token's groups", async ({
    page,
  }) => {
    try {
      patchRoom({ browserGroups: { prefix: "demo:" } });
      await roomSettled();
      await page.goto("/app/login");
      await expect(
        page.getByText("and your browser, Safari on iOS.", { exact: false }),
      ).toBeVisible();
      await page.getByLabel("Room code").fill(room().status.joinCode.code);
      await page.getByLabel("Display name").fill(testName);
      await page.getByRole("button", { name: "Continue", exact: true }).click();
      await expect(
        page.getByText(
          "Your groups: demo:room-pass-test, demo:browser-safari, demo:platform-ios",
          { exact: true },
        ),
      ).toBeVisible();
      const [p] = participants().filter(
        (p) => p.spec.displayName === storedName,
      );
      expect(p.spec.groups).toEqual([
        "demo:browser-safari",
        "demo:platform-ios",
      ]);
    } finally {
      patchRoom({ browserGroups: null });
      await roomSettled();
    }
  });
});
