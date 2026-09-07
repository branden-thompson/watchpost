import pathlib
# A band that cannot be cued takes the card off the air (DR-18 reversed): the words
# never read because the callout could not be shown.
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = '\t\tx.report(v, "the producer holds no alert for this card; the band keeps what it shows")\n\t\treturn nil\n'
new = '\t\treturn x.decline(v, v.ID, "the producer holds no alert for this card; the band keeps what it shows")\n'
assert old in s, "mC7"
p.write_text(s.replace(old, new, 1))
