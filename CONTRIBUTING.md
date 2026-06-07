# Contributing

Thanks for your interest in yaks-tui!

## Setup

You need Go and the [`yx`](https://github.com/mattwynne/yaks) binary on your
PATH. `fzf` is optional (enables fuzzy jump).

## Build and test

```bash
make build      # build the binary
go test ./...   # run the full suite
go vet ./...
gofmt -l .      # must report nothing
```

## Conventions

- Test-driven where practical; follow the table-driven test style already in
  the `*_test.go` files.
- Keep pure logic (tree/cursor math) in `internal/tree`, free of UI and I/O.
- Run `gofmt` before committing.
- See [AGENTS.md](AGENTS.md) for architecture notes and one critical gotcha
  (never query the terminal in the render loop).

## Task tracking

This repo tracks its own work with yaks, in `.yaks/`. Run `yx list` to see
what's in flight.
