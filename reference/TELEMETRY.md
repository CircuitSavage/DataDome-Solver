# Datadome Client-Side Telemetry & Fingerprinting

A research write-up of what the Datadome client script (`tags.js`, version 5.6.6 — see `SOURCE.md`) collects, how it collects it, and how it ships it back to the server. All line references point at `tags_deobfuscated.js` in this directory.

This document is descriptive — it explains the collection surface. It does not document or assist with defeating the system.

---

## 1. Architecture at a glance

The script runs three loops concurrently:

1. **Bootstrap & cookie handling** (lines 1–200). Reads/writes the `datadome` cookie across candidate domains, with `localStorage`/`sessionStorage` fallbacks when cookie writes fail. Establishes a session ID (`cid`) used for the lifetime of the page.
2. **Continuous behavioral capture** (lines ~440–710). Installs listeners for mouse, keyboard, touch, scroll, pointer, focus, and visibility events. Maintains running counters, inter-event timing arrays, and a "user interaction hash" (`uish`).
3. **One-shot environment fingerprinting** (lines ~1200–2100). On script start, probes navigator, screen, WebGL, audio/video codecs, fonts, plugins, MIME types, permissions, storage quotas, and ~50 automation markers.

When the script decides to report (either on a timer, on `pagehide`/`beforeunload`, or after a challenge), it bundles everything through a three-stage XOR/hash/encryption pipeline (lines ~2211–2246) and POSTs it via `navigator.sendBeacon` with an `XMLHttpRequest` fallback (lines 448–470).

The transport key is the cookie named **`datadome`**, plus a per-page **`cid`** session ID and a versioned **`ddk`** integrity token.

---

## 2. Environment fingerprinting

These signals are read once on startup and ride in the payload as named fields.

### 2.1 Navigator surface

| Field | Source | Notes |
|---|---|---|
| `ua` | `navigator.userAgent` | Full UA string (line 2515) |
| `lg` | `navigator.language` and friends | Falls back through `userLanguage`/`browserLanguage`/`systemLanguage` (line 1723) |
| `vnd` | `navigator.vendor` | (line 1728) |
| `bid` | `navigator.buildID` | Firefox-only (line 2587) |
| `hc` | `navigator.hardwareConcurrency` | Read via `Object.getOwnPropertyDescriptor` to dodge naïve overrides (line 1950) |
| `dvm` | `navigator.deviceMemory` | (line 1734) |
| `mtp` | `navigator.maxTouchPoints` | (line 2515) |
| `wbd` | `navigator.webdriver` | Also re-checked inside an iframe (line 1756) |
| `onL` | `navigator.onLine` | |
| `med` | `navigator.mediaDevices` presence | (line 1728) |
| `crt` | `navigator.connection.rtt` | |
| `niet` | `navigator.connection.effectiveType` | (line 1654) |
| `nid` | `navigator.connection.downlink` | |
| `nisd` | `navigator.connection.saveData` | |
| `isb` | `window.Brave` / `navigator.brave` | Brave detection (line 1727) |
| `idp` | `navigator.IdleDetector` | Capability probe (line 1724) |
| `xt1` | `navigator.pdfViewerEnabled` | (line 1931) |
| `nhi` | `navigator.userAgentData.getHighEntropyValues([...])` | Pulls architecture, bitness, model, platformVersion, uaFullVersion, wow64 (line 1251) |

Two integrity checks live alongside the data reads:

- **`pltod`** (line 1728): asks `Object.getOwnPropertyDescriptor(navigator, "platform")` and flags if a descriptor exists where one shouldn't.
- **`plggt`** (line 1379): stringifies the `plugins` getter and checks for `"return"` — a quick way to spot custom getter shims.

### 2.2 Screen, window, document

`outerWidth`/`outerHeight`/`innerWidth`/`innerHeight`/`screen.width`/`screen.height`/`screen.orientation.type` are all read (lines 1395–1408, 2515). Two derived flags ride along:

- **`isf`**: `outerHeight - innerHeight <= 1` (fullscreen guess, line 2075).
- **`isf2`**: `matchMedia("(display-mode: fullscreen)")` cross-check.
- **`dt`** (line 2083): DevTools heuristic — if the outer/inner delta exceeds 170 px, or `Firebug.chrome.isInitialized` is truthy, devtools is presumed open.

Document state: `document.hidden`, `document.hasFocus()`, `performance.getEntriesByType("visibility-state")[0].name` (initial visibility), and `window.XMLDocument.toString().length` as a tampering tripwire (line 1931).

### 2.3 GPU / WebGL

Function `J1` (lines 2580–2607) creates a hidden canvas, grabs a WebGL context, and:

- On Firefox ≥ 91, reads raw `VENDOR`/`RENDERER`.
- Otherwise loads `WEBGL_debug_renderer_info` and reads `UNMASKED_VENDOR_WEBGL` (`glvd`) and `UNMASKED_RENDERER_WEBGL` (`glrd`).

`"NA"` is the fallback on any error. The vendor+renderer pair is one of the highest-entropy non-behavioral signals in the payload.

### 2.4 Audio / video codec matrix

Lines 1845–2043 walk through ~14 `canPlayType`/`MediaSource.isTypeSupported` pairs (Theora, AVC, VP8, 3GPP, AV1, Matroska, QuickTime, HEVC on the video side; WAV, 3GPP, MPEG, OGG Vorbis, M4A, FLAC on the audio side). Each pair becomes two boolean fields (`vco`/`vcots`, `acw`/`acwts`, etc.). A separate flag `ocpt` checks whether `HTMLAudioElement.prototype.canPlayType.toString()` still contains the substring `"canPlayType"` — a cheap test for monkey-patched codec reporting (line 2040).

### 2.5 Fonts, plugins, MIME types, devices

- **Fonts**: `document.fonts` iterator, collecting each `FontFace.family` into `dffls` (line 1651).
- **Plugins**: `navigator.plugins.length` → `plu`; names joined into `plg`; plus four self-consistency checks (`plgne`, `plgre`, `plgof`, `plggt`) that compare reference equality across `plugins[0]`, `plugins[0][0].enabledPlugin`, and `plugins.item(random)` (line 1379).
- **MIME types**: enumerated into `mmt` (lines 1815–1820).
- **Media devices**: `navigator.mediaDevices.enumerateDevices()` results are collected into `emd` along with the microphone permission state (line 1309).

### 2.6 CSS / display capabilities

Tests on `window.CSS.supports` (e.g. `-webkit-touch-callout`, `-moz-osx-font-smoothing`) populate `csssp` (lines 1900, 1907). Computed style probes on a styled test element read `color`/`backgroundColor` (line 1888). Media queries probe color gamut (`cg:`), dynamic range (`dr:`), and PWA display mode (`dm:`) at line 2062.

### 2.7 Storage capability

- **`ckwa`** (line 1972): writes `dd_testcookie=1; SameSite=None; Secure` and reads it back to verify cookie capability.
- **`stqe`** / **`stqu`** (line 1718): `navigator.storage.estimate()` quota and usage.
- localStorage / sessionStorage availability flags are also set at lines 144–150.

---

## 3. Behavioral telemetry

This is the part designed to separate humans from headless drivers. Listeners are installed around line 499 for `mousemove`, `mousedown`, `mouseup`, `click`, `keydown`, `keyup`, `touchstart`, `touchmove`, `touchend`, `pointermove`, `pointerdown`, `scroll`, `pagehide`, and `beforeunload`.

### 3.1 Event counters

Maintained in a private object, summarized into the payload as `m_m_c`, `m_s_c`, `m_c_c`, `k_kdc`, `k_kuc` and the derived ratios `m_cm_r` (clicks/moves) and `m_ms_r` (moves/scrolls) (lines 511, 699–711). A `-1` sentinel is used when a divisor is zero.

### 3.2 Mouse trajectory

For every accepted `mousemove`, the script records `clientX`/`clientY`/`screenX`/`screenY` plus the timestamp (lines 614–628). From this it computes:

- A `_dists` array of Euclidean distances between consecutive samples — used to look at movement smoothness.
- `_completedStrokes` — clusters of moves separated by gaps > 500 ms, giving a "stroke" view of intent.

`pointerdown.pressure` is captured when `pointerType === "mouse"` and `buttons > 0` (line 2070) — non-zero pressure with mouse pointer type is anomalous for synthetic events.

### 3.3 Keystroke timing

`_keyEvents` (lines 670–684) stores `{type, key, timeStamp}` for each keydown/keyup. Inter-keystroke deltas (`d.ts - i.ts`) and same-key down→up deltas (`d.ts - s.ts`) are derived. The script also filters `event.isTrusted === false` and `event.repeat === true` synthetic events (line 570).

### 3.4 Touch

`touchstart`/`touchmove`/`touchend` are counted. When available, the script pulls `event.getCoalescedEvents()` and `event.getPredictedEvents()` (line 649) to get the highest-resolution touch sample stream the platform exposes.

### 3.5 Event authenticity

Every event runs through a small filter (lines 570–572):

- `event.isTrusted` must be true.
- `event.timeStamp` must fall within ±5 s of `performance.now()`.
- For keys, `event.repeat` must be false.

Forged or replayed events get dropped before they reach the counters.

### 3.6 The UI hash (`uish`)

Computed at lines 514, 541–542: takes `mousemove/10`, `touchmove/10`, `scroll`, `click`, presence-of-keydown, presence-of-keyup, joins them with `_`, then FNV-1a hashes the string. This compact field lets the server cheaply spot impossible interaction-mix patterns (e.g. 0 moves, 10 clicks).

### 3.7 Frame liveness

A `requestAnimationFrame` callback (line 522) sets a `d = true` flag. A page that never paints — common in some headless setups — never flips the flag, and the server sees a fingerprint without it.

---

## 4. Timing & navigation

`PerformanceNavigationTiming` is mined heavily (lines 1662–1665) into ~17 named fields:

| Field | Meaning |
|---|---|
| `nt_tcp` | `connectEnd - connectStart` |
| `nt_dns` | `domainLookupEnd - domainLookupStart` |
| `nt_rd` | `redirectEnd - redirectStart` |
| `nt_rt` | `responseStart - requestStart` |
| `nt_irt` | `firstInterimResponseStart - requestStart` |
| `nt_tls` | `requestStart - secureConnectionStart` |
| `nt_ttf` | `responseEnd - fetchStart` |
| `nt_swt` | `fetchStart - workerStart` |
| `nt_csd` | `decodedBodySize - encodedBodySize` (compression delta) |
| `nt_nhp` | `nextHopProtocol` (h2/h3/http1.1) |
| `nt_rdc` | `redirectCount` |
| `nt_it` | `initiatorType` |
| `nt_prs` | `requestStart - connectEnd` |
| `nt_esc` | `secureConnectionStart - connectStart` |
| `nt_ttrd` | Derived TLS overhead ratio |
| `nt_le` | `loadEventEnd - loadEventStart` |
| `nt_dcle` | DCL event duration |
| `nt_di` | `domInteractive` |
| `nt_dc` | `domComplete` |

These are useful for the server in two ways: (a) plausibility — a real browser session has a self-consistent timing profile, and (b) network/server-class inference from RTT and protocol.

---

## 5. Automation & headless detection

This is where the script puts the most effort. Five overlapping mechanisms run.

### 5.1 Global-variable sweep

Lines 1781–1799 check `window` and `document` for ~25 known automation globals, with both a one-shot scan and a `setInterval` that keeps re-scanning `Object.keys(document)` for `$cdc_`:

```
__driver_evaluate, __webdriver_evaluate, __selenium_evaluate, __fxdriver_evaluate,
__webdriver_unwrapped, _Selenium_IDE_Recorder, _selenium, calledSelenium,
$cdc_asdjflasutopfhvcZLmcfl_, $chrome_asyncScriptInfo,
__$webdriverAsyncExecutor, webdriver, __webdriverFunc,
domAutomation, domAutomationController,
__lastWatirAlert, __lastWatirConfirm, __lastWatirPrompt,
__webdriver_script_fn, __webdriver_script_func, __webdriver_script_function,
_WEBDRIVER_ELEM_CACHE
```

Plus a `document.addEventListener` for the synthetic events Selenium/WebDriver dispatches: `driver-evaluate`, `webdriver-evaluate`, `selenium-evaluate`, `webdriverCommand`, `webdriver-evaluate-response`.

### 5.2 Framework-specific globals

Line 2064–2080 covers the niche frameworks:

| Field | Probe |
|---|---|
| `phe` | `window.callPhantom` (PhantomJS) |
| `awe` | `window.awesomium` |
| `nm` | `window.__nightmare` |
| `geb` | `window.geb` |
| `sqt` | `window.external.toString().includes("Sequentum")` |
| `pw` | One of `__playwright_builtins__`, `__pwInitScripts`, `__playwright__binding__`, `__pwWebSocketDispatch`, `__playwright__binding__controller__` |
| `spwn` / `emt` | `window.spawn` / `window.emit` (Node-injected globals) |
| `bbs3` | `__stagehandV3__` or shadow-DOM tampering signature |
| `fai` | Fellou-webview partition globals |
| `pcb` | CSS signature: `z-index: 2147483647` + `min-height: 32px` on body |

### 5.3 DOM-method wrapping with caller inspection

The most aggressive check. At line 1708 the script wraps `document.getElementById`, `getElementsByTagName`, `querySelector`, `querySelectorAll`, and `evaluate` with a function `t()` that, on each call, decrements a counter (initial value 50 at line 1678) and inspects `arguments.callee.caller.toString()` for tell-tale substrings:

- `function (){var _0x` / `function(){var _0x` — generic obfuscator output
- `puppeteer | pptr: | ElementHandle | evaluateHandle` (regex) — Puppeteer
- `eval at evaluate` — CDP `Runtime.evaluate` frames
- `eval at executeScript` — Selenium executeScript
- Plus several site-framework strings (`#iadvize-container`, `dot-optimeeze`, `ga-disable-`, YouTube `iframe_api`+`player_api`, etc.) that don't signal bots — they're context for the server to know what page library called us.

Findings get serialized into fields `cfpfe`, `cffrb`, `cfpp`, `cfcpw`, `cfse`, plus `stcfp` (the last 150 chars of the stack) and `slat` (a Selenium-active marker). The counter exists to bound CPU cost — after ~50 wrapped calls the inspection is disabled.

### 5.4 Native-function integrity

Several small checks confirm that built-in functions still look native:

- `Function.toString().match(/\{\s*\[native code\]\s*\}$/)` on the audited functions (lines 1677, 2089).
- `Function.toString().length === 37` (line 2348) — a Chrome-specific signature length.
- `Math.random.toString()` and `Math.cos.toString()` checked for native-code shape (lines 1442–1446).
- A `C1()` helper (lines 2318–2341) tries `Object.setPrototypeOf(fn, fn)` and treats the resulting `RangeError` as proof the function is a real native (synthetic wrappers don't throw on cyclic prototypes). Used to set `hcovdr`, `plovdr`, `ftsovdr`.

### 5.5 Iframe cross-checks

At line 2518 the script injects a hidden iframe with `srcdoc="/**/"` and reads its `window.navigator.webdriver`, plugin/mimeType native-ness, and compares `iframe.self.toString().length` against the top window (lines 1733–1756). The technique exploits an asymmetry: most automation tools patch the top window but leave the iframe's window alone, so disagreement between the two is itself a fingerprint.

The `npmtm` check is particularly clever — it `postMessage`s the plugin/mimeType objects to itself. Native types throw on structured-clone; synthetic JS replacements clone cleanly. So a "successful" clone is the bot tell.

---

## 6. Anti-tampering on the script itself

### 6.1 Caller-hash chain

At lines 1764–1772 the script collects:

- `ccsT` — first 150 chars of the calling stack
- `ccsB` — last 150 chars
- `ccsH` — SHA-1 of the stack
- `ccsV` — variant hashes for cross-comparison

Designed to detect a third party wrapping or proxying the Datadome function in a way that changes the call shape.

### 6.2 XOR accumulators

Three running 32-bit accumulators — `e.L`, `e.P`, `e.j` — are updated as signals are collected via three helper methods (lines 2524–2561):

- `.T(str)` — folds into `e.L` and `e.j` (used for timezone, language, device fields)
- `.S(str)` — folds into `e.P` and `e.j` (used for platform, codecs)
- `.m(str)` — folds into `e.j` only (used for media types)

The final fields `sgb`, `sgc`, `sgd` are `e.L>>>0`, `e.P>>>0`, `e.j>>>0` (line 2114). The server independently computes the same XOR over the field values it receives and compares — any client that tampered with a field after collection but before signing fails the check.

---

## 7. Payload assembly & transport

### 7.1 The form fields

The body POSTed to `dataDomeOptions.endpoint` is `application/x-www-form-urlencoded` (line 459) with these top-level keys:

| Key | Contents |
|---|---|
| `jspl` | Encrypted signal blob (see §7.2) |
| `eventCounters` | JSON-stringified `{mousemove, scroll, click, keydown, keyup, touchmove}` |
| `jsType` | `"le"` (live event, on user action) or `"fm"` (fast mode, async) |
| `cid` | Session ID, pulled from the `datadome` cookie or `window.ddm.cid` |
| `ddk` | Versioned integrity token |
| `Referer` | Filtered `document.referrer` (≤1024 chars) |
| `request` | `location.pathname + search + hash` (≤1024 chars) |
| `responsePage` | Challenge response URL if applicable |
| `ddv` | `dataDomeOptions.version` |
| `custom` | Optional, only if `dataDomeOptions.customParam` is set |

### 7.2 The encryption pipeline

Function `u1(f, version)` at line 2211 returns a 2-tuple `[v, l]`. Function `j1(...)` at line 2563 then runs each collected signal through `U(n)(o, a, u, f)` with that tuple plus a payload accumulator `w` (lines 2563–2579). The result of the whole stage is `[j1(d,v,l,w,s), j1(h,v,l,w,s), f]` (line 2246) — three segments returned.

A custom base64 alphabet appears at line 2262:

```
H1DAxCvrj7IaPRL8GSJZKX3f62e9d0VTilFEOWgUB=/t+QmMwuskNnhpb4oyq5Yzc
```

Used by helper `A()` for non-standard base64 encoding/decoding of the encrypted segments. The actual symmetric cipher is dispatched through `u1` and isn't a single recognizable named primitive in the deobfuscated output — it's mixed with the running XOR state, which is why the integrity accumulators in §6.2 also feed it.

### 7.3 Send paths

Three transport paths, tried in order:

1. **`navigator.sendBeacon(endpoint, URLSearchParams)`** (line 448) — preferred because it survives `pagehide`.
2. **`XMLHttpRequest` POST** (line 456) — fallback if sendBeacon returns `false`.
3. **Service worker** — if `dataDomeOptions.enableServiceWorkerPlugin` is set, fingerprints are passed via `navigator.serviceWorker.controller.postMessage()` over a `MessageChannel` (lines 2677–2683) so collection can continue after the page is gone.

### 7.4 Replay / interception

`window.fetch` and `XMLHttpRequest.prototype.open` / `.send` are wrapped (lines 861–1024). The script keeps a queue of in-flight requests so that, if a challenge is interposed, the original requests can be replayed after the challenge succeeds. This is how Datadome stays mostly invisible to site code — the wrapped calls hold and replay rather than fail.

---

## 8. Challenge flow

Signal-side, this is short. The server returns a directive in the `x-dd-b` response header (or `x-sf-cc-x-dd-b` for Salesforce-fronted deployments) at line 23. Body JSON carries `{cookie, url, ...}`. Four severity buckets exist (lines 38–41):

- `BLOCK`
- `HARD_BLOCK`
- `DEVICE_CHECK`
- `DEVICE_CHECK_INVISIBLE_MODE`

For visible challenges the script injects a full-viewport iframe (line 411):

```html
<iframe id="ddChallengeBody..."
        sandbox="allow-same-origin allow-scripts ..."
        allow="accelerometer; gyroscope; magnetometer"
        style="height:100vh;width:100%;position:fixed;top:0;z-index:2147483647;">
```

The `allow` list is worth noting — the challenge iframe is permitted access to motion/orientation sensors. The base script does not read those itself; this implies the in-challenge page may use them for an additional fingerprinting pass that doesn't appear in `tags.js`.

Result delivery is via `window.postMessage` (`window.addEventListener("message", v, false)` at line 407). On success: server-provided `datadome` cookie is set, queued requests replay, and custom events (`dd_post_done`, `dd_captcha_passed`) fire on `document` so page code can react.

---

## 9. Storage & persistence

Three layers, in priority order:

1. **`datadome` cookie** — primary persistence. Lines 23–113 attempt to write across multiple domain levels (full host, registrable domain, parent labels) and probe for `Partitioned` (CHIPs) support. The successful domain is cached in `sessionStorage` as `ddCookieCandidateDomain` (line 84) so subsequent writes skip the probing.
2. **`localStorage`** — fallback when cookies are blocked. Key name comes from `dataDomeOptions.ddCookieSessionName` (line 95).
3. **`sessionStorage`** — used for the domain-candidate hint above; not the session ID itself.

---

## 10. What the server can derive

Putting the pieces together, the payload lets the server compute or check:

- **Static device identity** — UA + UA-CH + GPU (`glvd`/`glrd`) + codec matrix + plugins + screen metrics + font list. Together these are near-unique on a typical install.
- **Network locality** — `nt_*` timings, `niet`/`nid`, `nt_nhp`. Distinguish a residential link from a datacenter egress at the packet-timing level.
- **Live-user plausibility** — `uish`, the raw event counters, `_dists`, `_completedStrokes`, keystroke deltas, `pointerdown.pressure`, the rAF liveness flag. A scripted browser tends to have impossible ratios (clicks without movement, perfectly straight `_dists`, monotonic key deltas).
- **Automation tooling** — direct global checks + the synthetic-event listeners + the DOM-method wrapper + the iframe cross-check. Each one a different vendor; together they cover the popular tools (Selenium, Puppeteer, Playwright, Phantom, Nightmare, Stagehand, plus generic CDP).
- **Tampering** — `sgb`/`sgc`/`sgd` XOR signatures, `ccsT`/`ccsB`/`ccsH` caller hashes, the `pltod`/`plggt`/`ocpt`/`hcovdr`-style descriptor checks. These exist not because they directly identify a bot, but because they catch payloads that *were tampered with after collection* — a common pattern in solver libraries that fill in plausible-looking fingerprints.

---

## 11. Notes on what's outside this file

The base `tags.js` is one half of the system. A few things the server side (and the in-challenge page loaded under §8) almost certainly does that aren't observable from the client script alone:

- **TLS/JA3/JA4 fingerprinting** — done at the edge before the script even loads.
- **HTTP/2 SETTINGS frame fingerprinting** — same.
- **The actual cipher inside `u1()`** — the deobfuscated output makes the *shape* of the pipeline visible (three stages, XOR accumulators, custom base64) but the symmetric primitive itself is dispatched through wrapped functions and obfuscated control flow. Without the server-side validator, the exact transform isn't recoverable from the client alone.
- **Sensor-based fingerprinting inside the challenge iframe** — implied by the `allow="accelerometer; gyroscope; magnetometer"` on the iframe but not visible in `tags.js`.

---

## File / line index

All line numbers in this document are relative to `tags_deobfuscated.js` (the deobfuscated form), v5.6.6 as captured in `reference/`. The corresponding offsets in `tags.js` differ due to obfuscation but the structure maps 1:1.
