import sys
def spread(s): return " ".join(s)
def centered(t, inner):
    if len(t) > inner: t = t.replace(" ", "")
    if len(t) > inner: t = t[:inner]
    pad = inner-len(t); return " "*(pad//2)+t+" "*(pad-pad//2)
def bracket(t, w): return "["+centered(t, w-2)+"]"
def panel(title, stamp, lines, w, ascii_):
    tl,tr,bl,br,hz,vt = ("+","+","+","+","-","|") if ascii_ else ("┌","┐","└","┘","─","│")
    fill = w - 10 - len(title) - len(stamp)
    head = title + " " + hz*fill + " " + stamp if fill > 1 else title
    pad = max(0, w-len(tl+hz+hz+" "+head+" ")-1)
    out=[tl+hz+hz+" "+head+" "+hz*pad+tr]
    for l in lines:
        l=l[:w-4]; out.append(vt+"  "+l+" "*(w-4-len(l))+vt)
    out.append(bl+hz*(w-2)+br); return "\n".join(out)
def rail(lines, maxl, scroll, inner, ascii_):
    up,dn,th,bar = ("^","v","#","|") if ascii_ else ("▲","▼","█","│")
    n=len(lines)
    if n<=maxl: return [l[:inner]+" "*(inner-len(l[:inner])) for l in lines]
    maxs=n-maxl; scroll=max(0,min(scroll,maxs)); thumb=1+scroll*(maxl-3)//max(1,maxs) if maxl>3 else 1
    out=[]
    for i in range(maxl):
        g=bar
        if i==0: g=up
        elif i==maxl-1: g=dn
        elif i==thumb: g=th
        l=lines[scroll+i][:inner]; out.append(l+" "*(inner-len(l))+" "+g)
    return out
def browse(W, ascii_, rows, cat="Warnings", total=None, empty=False):
    optsw = max(W-5,40)-2; w = min(optsw, 110); inner = w-7  # rail budget
    ptr,mark = (">","*") if ascii_ else ("›","▶")
    tabs=["Warnings","Watches","Advisories","Spec. Statements","Sig. Quakes","Tropical"]
    short={"Warnings":"Warn","Watches":"Watch","Advisories":"Advis","Spec. Statements":"Stmts","Sig. Quakes":"Quakes","Tropical":"Tropical"}
    def tabrow(fmt):
        return " ".join(fmt(t) for t in tabs)
    forms=[lambda t: ("[ "+ptr+" "+t+" ]") if t==cat else ("[ "+t+" ]"),
           lambda t: ("["+ptr+t+"]") if t==cat else ("["+t+"]"),
           lambda t: ("["+ptr+short[t]+"]") if t==cat else ("["+short[t]+"]")]
    tr=next((tabrow(f) for f in forms if len(tabrow(f))<=inner), tabrow(forms[-1]))
    lines=["", "  "+tr, ""]
    n = len(rows) if total is None else total
    if empty:
        lines += ["  "+cat+" — no active events", "", "  No active "+cat.lower()+" events · Updated 08/28 15:38 PDT"]
        if cat in ("Advisories","Spec. Statements"): lines.append("  (tracks your watchlist — add locations with ctrl+a)")
        body=lines
    else:
        lines.append("  "+cat+" — "+str(len(rows))+" active" + (f" · showing {len(rows)} of {n}" if n>len(rows) else ""))
        # columns: marks 7 | num 5 | EVENT fill | LOC 22 | DECL 15 | EXP 15, gutters 2; degrade: EXPIRES then DECLARED then LOC→16
        cols=[("LOCATION",22),("DECLARED",15),("EXPIRES",15)]
        def evw(): return inner-(7+5)-sum(2+w for _,w in cols)
        if evw()<22: cols=cols[:2]
        if evw()<22: cols=cols[:1]
        if evw()<22: cols=[("LOCATION",16)]
        ev=max(14,evw())
        hdr = " "*12 + bracket(spread("EVENT"), ev) + "".join("  "+bracket(spread(n),w) for n,w in cols)
        table=[hdr]
        sevg={"!!":("!!" if ascii_ else "⚠⚠"),"!":("! " if ascii_ else "⚠ ")}
        for i,(sev,event,where,dec,exp) in enumerate(rows,1):
            m = (ptr+"  "+mark+" " if i==1 else " "*5) + sevg[sev][0] + " "
            e = event if len(event)<=ev else event[:ev-1]+"…"
            vals={"LOCATION":where,"DECLARED":dec,"EXPIRES":exp}
            cells="".join("  "+(vals[n] if len(vals[n])<=w else vals[n][:w-1]+"…").ljust(w) for n,w in cols)
            table.append(m + f"{i:03d}. " + e.ljust(ev) + cells)
        body=(lines,table)
    maxl = 14
    chips = "  [↑↓] Navigate  [←→] Category  [enter] Event Details  [esc] Close" if not ascii_ else "  [up/dn] Navigate  [lt/rt] Category  [enter] Event Details  [esc] Close"
    if empty:
        shown=[l[:w-4] for l in body+["", (f"{n} Total Category Events").rjust(inner), "", chips]]
        return panel("SEVERE WEATHER / DISASTER EVENTS", "Updated 08/28/2026 15:38:05 PDT", shown, w, ascii_)
    top, table = body
    foot = [(f"{n} Total Category Events").rjust(inner), "", chips]
    win = maxl - len(top) - len(foot)          # table rows visible (header + rows), rail spans exactly these
    shown = [l[:w-4] for l in top] + rail(table, win, 0, inner, ascii_) + [l[:w-4] for l in foot]
    return panel("SEVERE WEATHER / DISASTER EVENTS", "Updated 08/28/2026 15:38:05 PDT", shown, w, ascii_)
rows=[("!" ,"Extreme Heat Warning","Wicomico, MD","08/28 11:20 EDT","08/28 20:00 EDT"),
      ("!!","Tornado Warning","Olathe, KS","08/28 08:45 CDT","08/28 09:00 CDT"),
      ("!" ,"Flash Flood Warning","Palomar Mountain, CA","08/28 09:30 PDT","08/28 10:45 PDT"),
      ("!" ,"Severe Thunderstorm Warning","San Diego, CA","08/28 07:00 PDT","08/28 13:00 PDT"),
      ("!" ,"Gale Warning","Cape Cod Bay, MA","08/28 05:12 EDT","08/29 06:00 EDT"),
      ("!" ,"Red Flag Warning","Kern County Mtns, CA","08/28 04:00 PDT","08/28 20:00 PDT"),
      ("!" ,"Flood Warning","Wakefield, VA","08/27 22:41 EDT","08/29 12:00 EDT"),
      ("!" ,"Special Marine Warning","Chesapeake Bay, MD","08/27 21:05 EDT","08/27 22:00 EDT"),
      ("!" ,"High Wind Warning","Laramie, WY","08/27 18:00 MDT","08/28 06:00 MDT")]
for W,asc in [(120,False),(100,False),(80,False),(120,True)]:
    print(f"=== {W} cols {'--ascii' if asc else ''} ==="); print(browse(W,asc,rows)); print()
print("=== 120 cols EMPTY (Advisories, empty watchlist) ==="); print(browse(120,False,[],cat="Advisories",empty=True))
