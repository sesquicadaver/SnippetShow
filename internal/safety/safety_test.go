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
		{"echo ok", false, ""},
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
		{"#!/usr/bin/env bash\nsudo true\n", true, "sudo"},
		{"#!/usr/bin/env bash\necho hi\nrm -rf /tmp/x\n", true, "rm -rf"},
		// H2: path-prefixed and wrapper bypass matrix
		{"/usr/bin/sudo apt update", true, "sudo"},
		{"/bin/rm -rf /tmp/x", true, "rm -rf"},
		{"true && /usr/bin/dd if=a of=b", true, "dd"},
		{"env sudo apt update", true, "sudo"},
		{"env /usr/bin/sudo true", true, "sudo"},
		{"command sudo true", true, "sudo"},
		{"nice sudo true", true, "sudo"},
		{"nohup sudo true", true, "sudo"},
		{"/sbin/mkfs.ext4 /dev/sdb1", true, "mkfs"},
		{"env /bin/rm -rf /tmp/x", true, "rm -rf"},
		{"echo /usr/bin/sudo", false, ""},
		{"cat /home/user/sudoers.bak", false, ""},
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
