import pathlib
# An effect is dropped in grouping: work nobody does, and a card that waits for
# ever for a completion that can never arrive.
#
# REWRITTEN: the first form weakened the CHECK to its vacuous predecessor rather
# than breaking the rule, so both forms passed and it could never be caught. A
# mutant deletes a RULE; mutating an assertion only measures the assertion.
p = pathlib.Path("app/pump.go"); s = p.read_text()
old = "\t\truns = append(runs, []lineup.Effect{f})\n"
new = "\t\tif len(runs) > 0 {\n\t\t\tcontinue\n\t\t}\n\t\truns = append(runs, []lineup.Effect{f})\n"
assert old in s, "mDF"
p.write_text(s.replace(old, new, 1))
