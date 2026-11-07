import pathlib
# 0.19.5 Q6.4 (D-14, A-16): the map's builder reads every one of its Settings
# from the Config.
# Killed by TestEveryConfigFieldReachesTheModel.
p = pathlib.Path('modes/tty/dashboard_choices.go'); s = p.read_text()
old = '\t\tmapQuakeFeed:    quakeFeedByKey(cfg.MapQuakeFeed),\n'
assert s.count(old) == 1, 'm781'
p.write_text(s.replace(old, '\t\tmapQuakeFeed:    quakeFeedByKey(""),\n', 1))
