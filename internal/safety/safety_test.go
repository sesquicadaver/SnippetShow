package safety

import "testing"

func TestAnalyzeMatrix(t *testing.T) {
	cases := []struct {
		cmd       string
		dangerous bool
		reason    string
	}{
		{"ls -la", false, ""},
		{"echo sudo", false, ""},
		{"sudo apt update", true, "sudo"},
		{"echo hi && sudo reboot", true, "sudo"},
		{"dd if=/dev/zero of=/dev/sda", true, "dd"},
		{"true; dd if=a of=b", true, "dd"},
		{"mkfs.ext4 /dev/sdb1", true, "mkfs"},
		{"mkfs /dev/sdb", true, "mkfs"},
		{"rm -rf /tmp/x", true, "rm -rf"},
		{"rm -fr /tmp/x", true, "rm -rf"},
		{"rm -r -f /tmp/x", true, "rm -rf"},
		{"rm -f -r /tmp/x", true, "rm -rf"},
		{"rm --recursive --force /tmp/x", true, "rm -rf"},
		{"echo ok; rm -rf /", true, "rm -rf"},
		{"sudo rm -rf /", true, "rm -rf"},
		{"sudo rm -rf /", true, "sudo"},
		{"rm -r /tmp/only-recursive", false, ""},
		{"rm -f /tmp/only-force", false, ""},
		{"ffmpeg -i in.mp4 out.mkv", false, ""},
	}
	for _, tc := range cases {
		r := Analyze(tc.cmd)
		if r.Dangerous != tc.dangerous {
			t.Fatalf("%q: dangerous=%v want %v reasons=%v", tc.cmd, r.Dangerous, tc.dangerous, r.Reasons)
		}
		if tc.reason != "" {
			found := false
			for _, reason := range r.Reasons {
				if reason == tc.reason {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("%q: missing reason %q in %v", tc.cmd, tc.reason, r.Reasons)
			}
		}
	}
}

func TestAnalyzeEmpty(t *testing.T) {
	r := Analyze("   ")
	if r.Dangerous || len(r.Reasons) != 0 {
		t.Fatalf("%+v", r)
	}
}
