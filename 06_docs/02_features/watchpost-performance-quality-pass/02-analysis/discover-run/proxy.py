#!/usr/bin/env python3
"""Counting CONNECT proxy (DISCOVER A3): tunnels TLS untouched, logs one line per
outbound connection — timestamp, host — so requests/host/hour can be counted with
no change to the app (Go's transport honours HTTPS_PROXY)."""
import socket, threading, sys, time, select
LOG = sys.argv[1]; PORT = int(sys.argv[2])
lock = threading.Lock()
def log(host):
    with lock, open(LOG, "a") as f: f.write(f"{time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime())} {host}\n")
def pipe(a, b):
    try:
        while True:
            r, _, _ = select.select([a, b], [], [], 300)
            if not r: break
            for s in r:
                d = s.recv(65536)
                if not d: return
                (b if s is a else a).sendall(d)
    except OSError: pass
def handle(c):
    try:
        head = b""
        while b"\r\n\r\n" not in head:
            chunk = c.recv(4096)
            if not chunk: return
            head += chunk
        line = head.split(b"\r\n", 1)[0].decode(errors="replace")
        method, target = line.split(" ")[:2]
        if method != "CONNECT": c.sendall(b"HTTP/1.1 405 Method Not Allowed\r\n\r\n"); return
        host, port = target.rsplit(":", 1)
        log(host)
        up = socket.create_connection((host, int(port)), timeout=20)
        c.sendall(b"HTTP/1.1 200 Connection Established\r\n\r\n")
        pipe(c, up); up.close()
    except Exception as e:
        try: c.sendall(b"HTTP/1.1 502 Bad Gateway\r\n\r\n")
        except OSError: pass
    finally: c.close()
srv = socket.socket(); srv.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1); srv.bind(("127.0.0.1", PORT)); srv.listen(64)
while True:
    conn, _ = srv.accept(); threading.Thread(target=handle, args=(conn,), daemon=True).start()
