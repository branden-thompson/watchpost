package app

// maptiles_proxy_test.go — QA-12: the basemap's transport reaches a proxy on
// a private address, and holds the library's own transport settings.

import (
	"crypto/tls"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestTheTilesReachALocalProxy is QA-12: behind HTTPS_PROXY on 127.0.0.1 the
// basemap never loaded - the tile transport dialled the proxy through the
// library's checked dialer, which refuses a private address.
func TestTheTilesReachALocalProxy(t *testing.T) {
	var hits atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte("tile"))
	}))
	defer proxy.Close()
	via, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: libraryTransportVia(func(*http.Request) (*url.URL, error) { return via, nil })}
	res, err := client.Get("http://tiles.example/1/2/3.pbf")
	if err != nil {
		t.Fatalf("the tile request did not reach the local proxy: %v", err)
	}
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if string(body) != "tile" || hits.Load() != 1 {
		t.Errorf("the proxy answered %q after %d hits", body, hits.Load())
	}
}

// TestTheTilesStillRefuseAPrivateHost is the exemption's bound: with no proxy
// named, a tile host on loopback is refused at the dial.
func TestTheTilesStillRefuseAPrivateHost(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits.Add(1) }))
	defer srv.Close()
	client := &http.Client{Transport: libraryTransportVia(func(*http.Request) (*url.URL, error) { return nil, nil })}
	if res, err := client.Get(srv.URL + "/1/2/3.pbf"); err == nil {
		_ = res.Body.Close()
		t.Errorf("a tile host on 127.0.0.1 was reached (%d hits)", hits.Load())
	}
}

// TestTheTileTransportMatchesTheLibrarys holds watchpost's rebuilt transport
// to the library's own, read from the library's source in the module cache:
// the same fields set, and every duration, flag and TLS floor the same. It
// skips where the source cannot be found.
func TestTheTileTransportMatchesTheLibrarys(t *testing.T) {
	src := libraryFetchSource(t)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, src, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	consts := map[string]ast.Expr{}
	var lit *ast.CompositeLit
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ValueSpec:
			for i, name := range n.Names {
				if i < len(n.Values) {
					consts[name.Name] = n.Values[i]
				}
			}
		case *ast.CompositeLit:
			if sel, ok := n.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Transport" {
				lit = n
			}
		}
		return true
	})
	if lit == nil {
		t.Fatalf("no http.Transport literal in %s", src)
	}
	ours := reflect.ValueOf(*libraryTransportVia(http.ProxyFromEnvironment))
	for _, el := range lit.Elts {
		kv := el.(*ast.KeyValueExpr)
		field := kv.Key.(*ast.Ident).Name
		got := ours.FieldByName(field)
		if !got.IsValid() || got.IsZero() {
			t.Errorf("the library sets %s; the tile transport does not", field)
			continue
		}
		switch field {
		case "Proxy", "DialContext":
		case "TLSClientConfig":
			if got.Interface().(*tls.Config).MinVersion != tls.VersionTLS12 || !strings.Contains(exprText(t, fset, kv.Value), "tls.VersionTLS12") {
				t.Errorf("TLS floor differs: library %s", exprText(t, fset, kv.Value))
			}
		case "ForceAttemptHTTP2":
			if exprText(t, fset, kv.Value) != "true" || !got.Bool() {
				t.Errorf("ForceAttemptHTTP2 differs")
			}
		default:
			want := durationOf(t, fset, consts, kv.Value)
			if d := time.Duration(got.Int()); d != want {
				t.Errorf("%s: tile transport %s, library %s", field, d, want)
			}
		}
	}
}

// libraryFetchSource is the library's internal/fetch/fetch.go in the module
// cache, at the version this binary was built with.
func libraryFetchSource(t *testing.T) string {
	t.Helper()
	info, ok := debug.ReadBuildInfo()
	if !ok {
		t.Skip("no build info")
	}
	var dir string
	for _, m := range info.Deps {
		if m.Path != "github.com/branden-thompson/go-tuimaps" {
			continue
		}
		if m.Replace != nil && filepath.IsAbs(m.Replace.Path) {
			dir = m.Replace.Path
			break
		}
		cache := os.Getenv("GOMODCACHE")
		if cache == "" {
			gopath := os.Getenv("GOPATH")
			if gopath == "" {
				home, _ := os.UserHomeDir()
				gopath = filepath.Join(home, "go")
			}
			cache = filepath.Join(strings.Split(gopath, string(os.PathListSeparator))[0], "pkg", "mod")
		}
		dir = filepath.Join(cache, "github.com", "branden-thompson", "go-tuimaps@"+m.Version)
	}
	src := filepath.Join(dir, "internal", "fetch", "fetch.go")
	if dir == "" {
		t.Skip("go-tuimaps is not in the build info")
	}
	if _, err := os.Stat(src); err != nil {
		t.Skipf("the library's source is not at %s", src)
	}
	return src
}

// exprText is an expression as written.
func exprText(t *testing.T, fset *token.FileSet, e ast.Expr) string {
	t.Helper()
	return string(mustReadRange(t, fset, e))
}

func mustReadRange(t *testing.T, fset *token.FileSet, e ast.Expr) []byte {
	t.Helper()
	start, end := fset.Position(e.Pos()), fset.Position(e.End())
	b, err := os.ReadFile(start.Filename)
	if err != nil {
		t.Fatal(err)
	}
	return b[start.Offset:end.Offset]
}

// durationOf evaluates a duration written as a constant's name, N * time.Unit
// or time.Unit.
func durationOf(t *testing.T, fset *token.FileSet, consts map[string]ast.Expr, e ast.Expr) time.Duration {
	t.Helper()
	units := map[string]time.Duration{"Millisecond": time.Millisecond, "Second": time.Second, "Minute": time.Minute, "Hour": time.Hour}
	switch e := e.(type) {
	case *ast.Ident:
		if v, ok := consts[e.Name]; ok {
			return durationOf(t, fset, consts, v)
		}
	case *ast.SelectorExpr:
		if u, ok := units[e.Sel.Name]; ok {
			return u
		}
	case *ast.BinaryExpr:
		if lit, ok := e.X.(*ast.BasicLit); ok && e.Op == token.MUL {
			n, err := strconv.Atoi(lit.Value)
			if err == nil {
				return time.Duration(n) * durationOf(t, fset, consts, e.Y)
			}
		}
	}
	t.Fatalf("cannot read a duration from %s", exprText(t, fset, e))
	return 0
}
