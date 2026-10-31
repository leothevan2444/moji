package releasedate

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestSystemTimezoneAutoAndOverride(t *testing.T) {
	if os.Getenv("MOJI_TEST_TIMEZONE_CHILD") == "1" {
		now := time.Date(2026, 10, 31, 16, 0, 0, 0, time.UTC)
		_, _ = os.Stdout.WriteString(EffectiveTimezone("") + "|" + now.In(Location("auto")).Format("2006-01-02") + "|" + EffectiveTimezone("Asia/Tokyo"))
		os.Exit(0)
	}
	for _, tc := range []struct{ zone, date, name string }{{"UTC", "2026-10-31", "UTC"}, {"Asia/Shanghai", "2026-11-01", "Asia/Shanghai"}, {"America/New_York", "2026-10-31", "America/New_York"}, {"", "2026-10-31", "UTC"}} {
		t.Run(tc.name+tc.zone, func(t *testing.T) {
			command := exec.Command(os.Args[0], "-test.run=^TestSystemTimezoneAutoAndOverride$")
			env := []string{}
			for _, entry := range os.Environ() {
				if !strings.HasPrefix(entry, "TZ=") && !strings.HasPrefix(entry, "MOJI_TEST_TIMEZONE_CHILD=") {
					env = append(env, entry)
				}
			}
			command.Env = append(env, "TZ="+tc.zone, "MOJI_TEST_TIMEZONE_CHILD=1")
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatal(err, string(output))
			}
			want := tc.name + "|" + tc.date + "|Asia/Tokyo"
			if string(output) != want {
				t.Fatalf("got %s want %s", output, want)
			}
		})
	}
}
