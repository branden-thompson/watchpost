import pathlib
# 0.19.3 AX-5 (A-107 (7)): without the picture the region beyond an edge is said.
# Killed by TestWithoutThePictureTheEdgeIsSaid.
p = pathlib.Path('modes/tty/map_boxes.go'); s = p.read_text()
old = "\tif !d.mapPane.edgeShown {\n\t\treturn \"\"\n\t}\n\tnext, ok := geo.Neighbour(d.mapPane.region.Name, d.mapPane.edge)\n\tif !ok {\n\t\treturn \"\"\n\t}\n\tside"
assert s.count(old) == 1, 'm633'
p.write_text(s.replace(old, "\tif true {\n\t\treturn \"\"\n\t}\n\tnext, ok := geo.Neighbour(d.mapPane.region.Name, d.mapPane.edge)\n\tif !ok {\n\t\treturn \"\"\n\t}\n\tside", 1))
