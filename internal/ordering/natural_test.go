package ordering

import "testing"

func TestCompareNumericTitlesWithoutOverflow(t *testing.T) {
	for _, pair := range [][2]string{{"ABC-2", "abc-10"}, {"Actor 9", "Actor 100"}, {"A999999999999999999999", "A1000000000000000000000"}, {"演员2", "演员10"}} {
		if Compare(pair[0], pair[1]) >= 0 || Compare(pair[1], pair[0]) <= 0 {
			t.Fatal(pair)
		}
	}
	if Compare("ABC-002", "abc-2") != 0 {
		t.Fatal("numeric equality lost")
	}
}
