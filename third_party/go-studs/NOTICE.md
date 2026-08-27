# go-studs (in-repo copy)

go-studs is the author's MIT-licensed terminal UI kit (LICENSE alongside; upstream commit
`3e85e77`, pinned in LOCAL_CHANGES.md). Only the packages Watchpost imports are carried —
rendering, components, theme, tokens — without their tests or docs, as part of this module under
`third_party/go-studs`. Local changes are the patches under `patches/`, listed in
LOCAL_CHANGES.md and re-applied by `scripts/sync-go-studs.sh <path to the upstream checkout>`;
do not edit the packages directly — change upstream, or add a patch with its row.
