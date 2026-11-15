package scriptbuilder

import "testing"

func TestResolveBuiltinShadows(t *testing.T) {
	cases := map[string]string{
		"[ --help":           "/usr/bin/[ --help",
		"  [ --help":         "  /usr/bin/[ --help",
		"test --help":        "/usr/bin/test --help",
		"/usr/bin/[ --help":  "/usr/bin/[ --help",
		"/bin/test -n x":     "/bin/test -n x",
		"ls --help":          "ls --help",
		"[[ -f x ]]":         "[[ -f x ]]",
		"# [ --help":         "# [ --help",
		"":                   "",
	}
	for in, want := range cases {
		if got := ResolveBuiltinShadows(in); got != want {
			t.Fatalf("ResolveBuiltinShadows(%q)=%q want %q", in, got, want)
		}
	}
}

func TestResolveBuiltinShadowsBody(t *testing.T) {
	in := "#!/usr/bin/env bash\n[ --help\ntest -n x\n"
	want := "#!/usr/bin/env bash\n/usr/bin/[ --help\n/usr/bin/test -n x\n"
	if got := ResolveBuiltinShadowsBody(in); got != want {
		t.Fatalf("body=\n%q\nwant\n%q", got, want)
	}
}
