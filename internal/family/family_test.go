package family

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	ownFP   = "0123456789ABCDEF0123456789ABCDEF01234567"
	otherFP = "89ABCDEF0123456789ABCDEF0123456789ABCDEF"
	fp3     = "ABCDEF0123456789ABCDEF0123456789ABCDEF01"
	fp4     = "BCDEF0123456789ABCDEF0123456789ABCDEF012"
	fp5     = "CDEF0123456789ABCDEF0123456789ABCDEF0123"
	fp6     = "DEF0123456789ABCDEF0123456789ABCDEF01234"
)

func TestParse(t *testing.T) {
	data := []byte("# family members\n" +
		"$0123456789abcdef0123456789abcdef01234567\n" +
		"89ABCDEF0123456789ABCDEF0123456789ABCDEF\n" +
		"0123456789ABCDEF0123456789ABCDEF01234567 # duplicate\n" +
		"\n")
	got, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := []string{ownFP, otherFP}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("Parse = %v, want %v", got, want)
	}

	if _, err := Parse([]byte("0123456789ABCDEF\n")); err == nil {
		t.Fatal("want error for a short fingerprint")
	} else if !strings.Contains(err.Error(), "line 1") {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := Parse([]byte("not-hex\n")); err == nil {
		t.Fatal("want error for a non-hex line")
	}

	empty, err := Parse([]byte("# only comments\n\n"))
	if err != nil {
		t.Fatalf("Parse empty: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("Parse empty = %v, want none", empty)
	}
}

func TestReadOwnFingerprint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fingerprint")
	if err := os.WriteFile(path, []byte("TestRelay "+ownFP+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	fp, err := ReadOwnFingerprint(path)
	if err != nil {
		t.Fatalf("ReadOwnFingerprint: %v", err)
	}
	if fp != ownFP {
		t.Fatalf("fp = %q, want %q", fp, ownFP)
	}

	if _, err := ReadOwnFingerprint(filepath.Join(dir, "missing")); !os.IsNotExist(err) {
		t.Fatalf("missing file error = %v, want os.ErrNotExist", err)
	}

	if err := os.WriteFile(path, []byte("TestRelay\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadOwnFingerprint(path); err == nil {
		t.Fatal("want error for a file without a fingerprint")
	}
}

func TestWriteAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".torinit", "family.conf")
	if err := Write(path, []string{ownFP, otherFP}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("mode = %o, want 0600", info.Mode().Perm())
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 2 || got[0] != ownFP || got[1] != otherFP {
		t.Fatalf("Load = %v, want [%s %s]", got, ownFP, otherFP)
	}

	if err := Write(path, nil); err != nil {
		t.Fatalf("Write empty: %v", err)
	}
	got, err = Load(path)
	if err != nil {
		t.Fatalf("Load empty: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("Load empty = %v, want none", got)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("leftover temp files: %v", entries)
	}
}

func newManager(t *testing.T, srv *httptest.Server, opts Options) *Manager {
	t.Helper()
	m := New(opts)
	m.Client = srv.Client()
	m.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return fmt.Errorf("refusing redirect to %s", req.URL)
		}
		return nil
	}
	m.Client.Timeout = 5 * time.Second
	m.Logf = t.Logf
	return m
}

func familyServer(t *testing.T) (*httptest.Server, func(string)) {
	t.Helper()
	var mu sync.Mutex
	body := ""
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, func(s string) {
		mu.Lock()
		defer mu.Unlock()
		body = s
	}
}

func TestRefreshDropGuard(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "family.conf")
	current := []string{ownFP, otherFP, fp3, fp4, fp5}
	if err := Write(path, current); err != nil {
		t.Fatal(err)
	}

	srv, setBody := familyServer(t)
	m := newManager(t, srv, Options{
		URL:             srv.URL,
		MaxDropPct:      20,
		Path:            path,
		FingerprintPath: filepath.Join(dir, "missing-fingerprint"),
	})

	setBody(ownFP + "\n" + otherFP + "\n" + fp3 + "\n")
	changed, err := m.Refresh(0)
	if err == nil {
		t.Fatal("want drop guard error")
	}
	if changed {
		t.Error("rejected update reported a change")
	}
	if !strings.Contains(err.Error(), "40% > 20%") {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("rejected update changed the file: %v", got)
	}

	setBody(ownFP + "\n" + otherFP + "\n" + fp3 + "\n" + fp4 + "\n")
	changed, err = m.Refresh(0)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if !changed {
		t.Fatal("accepted update did not report a change")
	}
	got, err = Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("got %d entries, want 4", len(got))
	}
}

func TestRefreshRejectsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "family.conf")
	if err := Write(path, []string{ownFP, otherFP}); err != nil {
		t.Fatal(err)
	}

	srv, setBody := familyServer(t)
	m := newManager(t, srv, Options{
		URL:             srv.URL,
		MaxDropPct:      100,
		Path:            path,
		FingerprintPath: filepath.Join(dir, "missing-fingerprint"),
	})

	setBody("# no fingerprints today\n")
	changed, err := m.Refresh(0)
	if err == nil {
		t.Fatal("want empty update error")
	}
	if changed {
		t.Error("rejected update reported a change")
	}
	if !strings.Contains(err.Error(), "refusing empty update") {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("rejected update changed the file: %v", got)
	}
}

func TestRefreshRefusesHTTPRedirect(t *testing.T) {
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, otherFP)
	}))
	defer plain.Close()

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL+"/family.txt", http.StatusFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "family.conf")
	if err := Write(path, []string{ownFP}); err != nil {
		t.Fatal(err)
	}

	m := newManager(t, srv, Options{
		URL:             srv.URL,
		MaxDropPct:      100,
		Path:            path,
		FingerprintPath: filepath.Join(dir, "missing-fingerprint"),
	})

	changed, err := m.Refresh(0)
	if err == nil {
		t.Fatal("want redirect error")
	}
	if changed {
		t.Error("failed fetch reported a change")
	}
	if !strings.Contains(err.Error(), "refusing redirect") {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != ownFP {
		t.Fatalf("failed fetch changed the file: %v", got)
	}
}

func TestRefreshExcludesOwnFingerprintAndSignals(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "family.conf")
	fpPath := filepath.Join(dir, "fingerprint")
	if err := os.WriteFile(fpPath, []byte("TestRelay "+ownFP+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	srv, setBody := familyServer(t)
	setBody(ownFP + "\n" + otherFP + "\n")

	m := newManager(t, srv, Options{
		URL:             srv.URL,
		MaxDropPct:      100,
		Path:            path,
		FingerprintPath: fpPath,
	})

	var signaled int
	m.Signal = func(pid int) error {
		signaled = pid
		return nil
	}

	changed, err := m.Refresh(42)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if !changed {
		t.Fatal("want a change")
	}
	if signaled != 42 {
		t.Errorf("signaled pid = %d, want 42", signaled)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != otherFP {
		t.Fatalf("Load = %v, want only %s", got, otherFP)
	}
}

func TestRefreshNoChangeDoesNotSignal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "family.conf")
	if err := Write(path, []string{otherFP}); err != nil {
		t.Fatal(err)
	}

	srv, setBody := familyServer(t)
	setBody(otherFP + "\n")

	m := newManager(t, srv, Options{
		URL:             srv.URL,
		MaxDropPct:      100,
		Path:            path,
		FingerprintPath: filepath.Join(dir, "missing-fingerprint"),
	})
	m.Signal = func(pid int) error {
		t.Errorf("unexpected SIGHUP to %d", pid)
		return nil
	}

	changed, err := m.Refresh(7)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if changed {
		t.Fatal("unchanged list reported a change")
	}
}

func TestStartupWritesEmptyWhenFetchFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".torinit", "family.conf")

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	m := New(Options{
		URL:             srv.URL,
		Path:            path,
		FingerprintPath: filepath.Join(dir, "fingerprint"),
	})
	m.Client.Timeout = 2 * time.Second
	m.Logf = t.Logf

	m.Startup()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("family.conf was not created: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("Load = %v, want empty", got)
	}
}

func TestStartupKeepsExistingOnFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".torinit", "family.conf")
	if err := Write(path, []string{ownFP}); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	m := New(Options{
		URL:             srv.URL,
		Path:            path,
		FingerprintPath: filepath.Join(dir, "fingerprint"),
	})
	m.Client.Timeout = 2 * time.Second
	m.Logf = t.Logf

	m.Startup()

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 1 || got[0] != ownFP {
		t.Fatalf("Load = %v, want [%s]", got, ownFP)
	}
}

func TestStartupWritesFetchedList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".torinit", "family.conf")

	srv, setBody := familyServer(t)
	setBody("$" + otherFP + "\n" + fp6 + "\n")

	m := newManager(t, srv, Options{
		URL:             srv.URL,
		Path:            path,
		FingerprintPath: filepath.Join(dir, "missing-fingerprint"),
	})

	m.Startup()

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 2 || got[0] != otherFP || got[1] != fp6 {
		t.Fatalf("Load = %v, want [%s %s]", got, otherFP, fp6)
	}
}
