package keys

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func TestDiscover(t *testing.T) {
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "a.secret_family_key"), "a", 0600)
	writeFile(t, filepath.Join(src, "b"), "b", 0600)
	writeFile(t, filepath.Join(src, "c.txt"), "c", 0600)
	writeFile(t, filepath.Join(src, "d.public_family_id"), "d", 0600)
	if err := os.Mkdir(filepath.Join(src, "sub"), 0700); err != nil {
		t.Fatal(err)
	}

	files, err := Discover(src, true)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("got %d files, want 3: %+v", len(files), files)
	}
	want := []string{
		"a.secret_family_key",
		"b.secret_family_key",
		"c.txt.secret_family_key",
	}
	for i, file := range files {
		if file.Name != want[i] {
			t.Errorf("file %d name = %q, want %q", i, file.Name, want[i])
		}
		if file.Source == "" {
			t.Errorf("file %d has empty source", i)
		}
	}

	onlyKeys, err := Discover(src, false)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(onlyKeys) != 1 || onlyKeys[0].Name != "a.secret_family_key" {
		t.Fatalf("got %+v, want only a.secret_family_key", onlyKeys)
	}

	missing, err := Discover(filepath.Join(src, "missing"), true)
	if err != nil {
		t.Fatalf("Discover missing: %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("got %+v, want no files", missing)
	}
}

func TestValidate(t *testing.T) {
	if err := Validate([]string{"id"}, nil, "/run/secrets"); err == nil {
		t.Fatal("want error for FamilyId without keys")
	} else if err.Error() != "FamilyId set but no *.secret_family_key found in /run/secrets" {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := Validate(nil, []KeyFile{{Source: "x", Name: "x.secret_family_key"}}, "/run/secrets"); err == nil {
		t.Fatal("want error for keys without FamilyId")
	} else if err.Error() != "family key present but TOR_FAMILY_ID not set" {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := Validate([]string{"id"}, []KeyFile{{Source: "x", Name: "x.secret_family_key"}}, "/run/secrets"); err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	if err := Validate(nil, nil, "/run/secrets"); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
}

func TestInstall(t *testing.T) {
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "a.secret_family_key"), "one", 0600)
	writeFile(t, filepath.Join(src, "b"), "two", 0600)

	files, err := Discover(src, true)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	dest := filepath.Join(t.TempDir(), "keys")
	result, err := Install(files, dest)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(result.Installed) != 2 || len(result.Replaced) != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}

	dirInfo, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if dirInfo.Mode().Perm() != 0700 {
		t.Errorf("key dir mode = %o, want 0700", dirInfo.Mode().Perm())
	}

	for name, want := range map[string]string{
		"a.secret_family_key": "one",
		"b.secret_family_key": "two",
	} {
		path := filepath.Join(dest, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != want {
			t.Errorf("%s content = %q, want %q", name, data, want)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("%s mode = %o, want 0600", name, info.Mode().Perm())
		}
	}

	writeFile(t, filepath.Join(src, "a.secret_family_key"), "three", 0600)
	files, err = Discover(src, true)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	result, err = Install(files, dest)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(result.Replaced) != 1 || result.Replaced[0] != "a.secret_family_key" {
		t.Fatalf("unexpected result: %+v", result)
	}
	data, err := os.ReadFile(filepath.Join(dest, "a.secret_family_key"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "three" {
		t.Errorf("replaced content = %q, want three", data)
	}
}

func TestFingerprints(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "fingerprint"), "ExampleRelay 7A11C9EB35045D74B4C81942EE97AA083332797F\n", 0600)
	writeFile(t, filepath.Join(dir, "fingerprint-ed25519"), "ExampleRelay WqMjyrXYMionEMhmKmG6q6S0yGHAw0onG1uRDw0e2pY\n", 0600)

	rsa, ed25519, err := Fingerprints(dir)
	if err != nil {
		t.Fatalf("Fingerprints: %v", err)
	}
	if rsa != "7A11C9EB35045D74B4C81942EE97AA083332797F" {
		t.Errorf("rsa = %q", rsa)
	}
	if ed25519 != "WqMjyrXYMionEMhmKmG6q6S0yGHAw0onG1uRDw0e2pY" {
		t.Errorf("ed25519 = %q", ed25519)
	}

	if _, _, err := Fingerprints(t.TempDir()); err == nil {
		t.Fatal("want error for missing fingerprint files")
	} else if !strings.Contains(err.Error(), "read") {
		t.Fatalf("unexpected error: %v", err)
	}

	bad := t.TempDir()
	writeFile(t, filepath.Join(bad, "fingerprint"), "ExampleRelay\n", 0600)
	writeFile(t, filepath.Join(bad, "fingerprint-ed25519"), "ExampleRelay WqMjyrXYMionEMhmKmG6q6S0yGHAw0onG1uRDw0e2pY\n", 0600)
	if _, _, err := Fingerprints(bad); err == nil {
		t.Fatal("want error for malformed fingerprint file")
	}
}

func TestKeygenFamily(t *testing.T) {
	fakeDir := t.TempDir()
	fakeTor := filepath.Join(fakeDir, "tor")
	writeFile(t, fakeTor,
		"#!/bin/sh\nprintf 'fake-key\\n' > myfamily.secret_family_key\necho 'FamilyId fakefamilyid'\n",
		0755,
	)

	oldBinary := TorBinary
	TorBinary = fakeTor
	defer func() { TorBinary = oldBinary }()

	outDir := t.TempDir()
	line, err := KeygenFamily("myfamily", outDir, false)
	if err != nil {
		t.Fatalf("KeygenFamily: %v", err)
	}
	if line != "FamilyId fakefamilyid" {
		t.Fatalf("line = %q, want FamilyId fakefamilyid", line)
	}
	data, err := os.ReadFile(filepath.Join(outDir, "myfamily.secret_family_key"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fake-key\n" {
		t.Fatalf("key content = %q, want fake-key", data)
	}

	writeFile(t, filepath.Join(outDir, "myfamily.secret_family_key"), "old", 0600)
	if _, err := KeygenFamily("myfamily", outDir, false); err == nil {
		t.Fatal("want overwrite error")
	} else if !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("unexpected error: %v", err)
	}

	line, err = KeygenFamily("myfamily", outDir, true)
	if err != nil {
		t.Fatalf("KeygenFamily force: %v", err)
	}
	if line != "FamilyId fakefamilyid" {
		t.Fatalf("line = %q, want FamilyId fakefamilyid", line)
	}
	data, err = os.ReadFile(filepath.Join(outDir, "myfamily.secret_family_key"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fake-key\n" {
		t.Fatalf("forced key content = %q, want fake-key", data)
	}

	if _, err := KeygenFamily("bad/name", outDir, false); err == nil {
		t.Fatal("want invalid name error")
	} else if !strings.Contains(err.Error(), "invalid family name") {
		t.Fatalf("unexpected error: %v", err)
	}
}
