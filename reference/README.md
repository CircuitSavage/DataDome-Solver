# DataDome client reference

Captured browser script and deobfuscated output used to implement the Go solver. **Not executed by the Go build.**

## Files

| File | Description |
|------|-------------|
| `tags.js` | Obfuscated client as served by the protected site |
| `tags_deobfuscated.js` | Readable output from `deobfuscator.js` |
| `tags_deobfuscated_strings.json` | Decoded `f1` / `s1` string tables |
| `deobfuscator.js` | Babel AST deobfuscator |
| [`TELEMETRY.md`](TELEMETRY.md) | Signal-by-signal write-up (line refs into `tags_deobfuscated.js`) |
| `SOURCE.md` | Capture URL, date, and file sizes |

## Refresh `tags.js`

Download from the target origin (same path the site uses in production):

```bash
curl -fsSL "https://<origin>/include/tags.js" \
  -H "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36" \
  -o tags.js
```

This copy was taken from `https://www.etsy.com/include/tags.js`.

## Deobfuscate

```bash
cd reference
npm install
npm run deobfuscate
```

Or: `node deobfuscator.js tags.js tags_deobfuscated.js`

After DataDome updates their script, re-fetch, deobfuscate, then diff against the previous `tags_deobfuscated.js` and update `../go/internal/builder` and `../go/internal/crypto` if signal order or encryption changed.
