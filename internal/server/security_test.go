package server

import "testing"

func TestValidateAppName(t *testing.T) {
	valid := []string{"Agent_TNLX", "SAP TOOL", "SSO.Check-1"}
	for _, name := range valid {
		if err := ValidateAppName(name); err != nil {
			t.Fatalf("%q should be valid: %v", name, err)
		}
	}
	invalid := []string{"", "..", "../x", `..\x`, `C:\Windows`, `\\server\share`, "bad/name", "bad:name"}
	for _, name := range invalid {
		if err := ValidateAppName(name); err == nil {
			t.Fatalf("%q should be invalid", name)
		}
	}
}

func TestNormalizeAppNameTrimsSpaces(t *testing.T) {
	got, err := NormalizeAppName(" car ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "car" {
		t.Fatalf("got %q want car", got)
	}
}

func TestCleanRelativePath(t *testing.T) {
	valid := map[string]string{"config/app.json": "config/app.json", `dll\a.dll`: "dll/a.dll", "app.exe": "app.exe"}
	for in, want := range valid {
		got, err := CleanRelativePath(in)
		if err != nil {
			t.Fatalf("%q should be valid: %v", in, err)
		}
		if got != want {
			t.Fatalf("%q got %q want %q", in, got, want)
		}
	}
	invalid := []string{"", "../x", `..\x`, "/Windows/test.dll", `C:\Windows\test.dll`, `\\server\share\a.dll`}
	for _, p := range invalid {
		if _, err := CleanRelativePath(p); err == nil {
			t.Fatalf("%q should be invalid", p)
		}
	}
}
