# Go solver

Native implementation of the DataDome fingerprint → jspl → `/include/tags.js` flow.

```bash
go build -o datadome ./cmd/datadome
go test ./...
```

Import path: `github.com/CircuitSavage/datadome-solver/pkg/datadome`

Client script reference, deobfuscation, and the telemetry write-up live in [`../reference/`](../reference/) — see [`TELEMETRY.md`](../reference/TELEMETRY.md) for signal documentation.
