## What

<!-- One paragraph: what changes and why. Link the issue or the UAT session it answers. -->

## How to see it

<!-- Commands, keys, or a screenshot. For UI changes, the mock it matches. -->

## Checks

- [ ] `make verify` is green (fmt, vet, race, import direction, watermark gates)
- [ ] `golangci-lint run ./...` and `staticcheck ./...` are clean
- [ ] Tests added or updated for the behaviour that changed
- [ ] Docs touched where the behaviour is described (README, CHANGELOG, `docs/extending.md`)
- [ ] No secrets, personal addresses, or machine-local paths in the diff

## Notes for the reviewer

<!-- Risks, follow-ups, anything deliberately left out. -->
