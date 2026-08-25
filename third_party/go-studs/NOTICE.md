# go-studs (in-repo copy)

go-studs is the author's MIT-licensed terminal UI kit (LICENSE alongside; upstream commit
`3e85e77`). Only the packages Watchpost imports are carried — rendering, components, theme,
tokens — without their tests or docs, as part of this module under `third_party/go-studs`.
Do not edit here: change upstream and rerun `scripts/sync-go-studs.sh <path to the upstream
checkout>`, which copies the packages and rewrites their import paths to this module.
