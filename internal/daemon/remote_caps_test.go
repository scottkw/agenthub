package daemon

import (
	"os"
	"regexp"
	"sync"
	"testing"
)

func TestRemoteCapStore_NewIsEmpty(t *testing.T) {
	s := NewRemoteCapStore()
	if s == nil {
		t.Fatal("NewRemoteCapStore returned nil")
	}
	if _, _, ok := s.Get("anything"); ok {
		t.Fatalf("freshly constructed store should be empty")
	}
}

func TestRemoteCapStore_PutThenGet(t *testing.T) {
	s := NewRemoteCapStore()
	if err := s.Put("sid-1", "https://peer.example", "tok-abc"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	base, tok, ok := s.Get("sid-1")
	if !ok {
		t.Fatal("Get returned ok=false after Put")
	}
	if base != "https://peer.example" || tok != "tok-abc" {
		t.Fatalf("Get returned wrong values: (%q, %q)", base, tok)
	}
}

func TestRemoteCapStore_GetUnknown(t *testing.T) {
	s := NewRemoteCapStore()
	base, tok, ok := s.Get("unknown")
	if ok || base != "" || tok != "" {
		t.Fatalf("Get(unknown) = (%q, %q, %v); want (\"\", \"\", false)", base, tok, ok)
	}
}

func TestRemoteCapStore_PutOverwrites(t *testing.T) {
	s := NewRemoteCapStore()
	_ = s.Put("sid-1", "https://old.example", "tok-old")
	if err := s.Put("sid-1", "https://new.example", "tok-new"); err != nil {
		t.Fatalf("Put overwrite: %v", err)
	}
	base, tok, _ := s.Get("sid-1")
	if base != "https://new.example" || tok != "tok-new" {
		t.Fatalf("overwrite: got (%q, %q); want new values", base, tok)
	}
}

func TestRemoteCapStore_Delete(t *testing.T) {
	s := NewRemoteCapStore()
	_ = s.Put("sid-1", "https://peer.example", "tok-abc")
	s.Delete("sid-1")
	if _, _, ok := s.Get("sid-1"); ok {
		t.Fatal("Get after Delete returned ok=true")
	}
	// Deleting an unknown sessionID should be a no-op (no panic).
	s.Delete("never-existed")
}

func TestRemoteCapStore_ConcurrentPutGet(t *testing.T) {
	// Exercise the RWMutex contract: 100 goroutines hammering Put + Get
	// must not race under `go test -race`.
	s := NewRemoteCapStore()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			_ = s.Put("sid", "https://peer.example", "tok")
			_ = i
		}(i)
		go func(i int) {
			defer wg.Done()
			_, _, _ = s.Get("sid")
			_ = i
		}(i)
	}
	wg.Wait()
}

func TestRemoteCapStore_PutRejectsEmpty(t *testing.T) {
	s := NewRemoteCapStore()
	cases := []struct {
		name, sid, base, tok string
	}{
		{"empty sid", "", "https://peer.example", "tok"},
		{"empty base", "sid", "", "tok"},
		{"empty tok", "sid", "https://peer.example", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := s.Put(c.sid, c.base, c.tok); err == nil {
				t.Fatal("expected error from Put with empty input; got nil")
			}
		})
	}
}

func TestRemoteCapStore_NoDiskWriteAPIs(t *testing.T) {
	// Source-grep guard for the no-disk-persistence invariant. If a future
	// edit reaches for os.WriteFile, os.Create, or ioutil.WriteFile in
	// remote_caps.go, this test breaks and the author is forced to either
	// remove the disk write or update the threat model.
	src, err := os.ReadFile("remote_caps.go")
	if err != nil {
		t.Fatalf("read remote_caps.go: %v", err)
	}
	// Strip line comments so the doc comment forbidding disk writes does
	// not itself trigger the match.
	stripped := regexp.MustCompile(`(?m)^\s*//.*$`).ReplaceAll(src, []byte(""))
	banned := regexp.MustCompile(`os\.WriteFile|ioutil\.WriteFile|os\.Create`)
	if loc := banned.FindIndex(stripped); loc != nil {
		t.Fatalf("remote_caps.go must not reference disk-write APIs; found %q", stripped[loc[0]:loc[1]])
	}
}
