// Package bucket splits an ordered slice into per-bucket index lists.
//
// ONE OWNER FOR AN OPERATION THAT HAD TWO (metric D, 2026-09-08).
// `severe.ByTab` and `tty.bucketSevere` were the same walk — same guard, same
// order-preserving append — over different payloads. They live in packages that
// CANNOT share code directly: `modes/` may not import `domains/`
// (architecture §1, gated by lint-imports), so the shared owner has to be here.
package bucket

// ByIndex returns, for each bucket, the indices of the items that fall in it,
// in the order they appear. Items whose bucket is out of range are dropped —
// that is the guard both callers already had.
//
// IT RETURNS INDICES, NOT ITEMS, and that is the tty caller's requirement
// rather than a style choice: the severe table indexes into the published rows
// and never copies them (R3-B-11). A caller that wants values maps them back,
// which is what the domains side does.
// B IS ~int, NOT int, because every caller has a NAMED bucket type — a
// category.Category, a Tab, a SevereTab. A plain `int` would push a conversion
// to both call sites, and a conversion at a call site is where an off-by-one
// between two enums hides.
func ByIndex[T any, B ~int](items []T, n B, of func(T) B) [][]int {
	out := make([][]int, int(n))
	for i, it := range items {
		if b := of(it); b >= 0 && b < n {
			out[b] = append(out[b], i)
		}
	}
	return out
}
