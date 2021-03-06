package semver

import (
	"strings"
	"testing"
)

type comparatorTest struct {
	input      string
	comparator func(comparator) bool
}

func TestParseComparator(t *testing.T) {
	compatorTests := []comparatorTest{
		{">", testGT},
		{">=", testGE},
		{"<", testLT},
		{"<=", testLE},
		{"", testEQ},
		{"=", testEQ},
		{"==", testEQ},
		{"!=", testNE},
		{"!", testNE},
		{"-", nil},
		{"<==", nil},
		{"<<", nil},
		{">>", nil},
	}

	for _, tc := range compatorTests {
		if c := parseComparator(tc.input); c == nil {
			if tc.comparator != nil {
				t.Errorf("Comparator nil for case %q\n", tc.input)
			}
		} else if !tc.comparator(c) {
			t.Errorf("Invalid comparator for case %q\n", tc.input)
		}
	}
}

var (
	v1 = MustParse("1.2.2")
	v2 = MustParse("1.2.3")
	v3 = MustParse("1.2.4")
)

func testEQ(f comparator) bool {
	return f(v1, v1) && !f(v1, v2)
}

func testNE(f comparator) bool {
	return !f(v1, v1) && f(v1, v2)
}

func testGT(f comparator) bool {
	return f(v2, v1) && f(v3, v2) && !f(v1, v2) && !f(v1, v1)
}

func testGE(f comparator) bool {
	return f(v2, v1) && f(v3, v2) && !f(v1, v2)
}

func testLT(f comparator) bool {
	return f(v1, v2) && f(v2, v3) && !f(v2, v1) && !f(v1, v1)
}

func testLE(f comparator) bool {
	return f(v1, v2) && f(v2, v3) && !f(v2, v1)
}

func TestSplitComparatorVersion(t *testing.T) {
	tests := []struct {
		i string
		p []string
	}{
		{">1.2.3", []string{">", "1.2.3"}},
		{">=1.2.3", []string{">=", "1.2.3"}},
		{"<1.2.3", []string{"<", "1.2.3"}},
		{"<=1.2.3", []string{"<=", "1.2.3"}},
		{"1.2.3", []string{"", "1.2.3"}},
		{"=1.2.3", []string{"=", "1.2.3"}},
		{"==1.2.3", []string{"==", "1.2.3"}},
		{"!=1.2.3", []string{"!=", "1.2.3"}},
		{"!1.2.3", []string{"!", "1.2.3"}},
		{"1.x.x", []string{"", "1.x.x"}},
		{"1.x", []string{"", "1.x"}},
		{">=1.x", []string{">=", "1.x"}},
		{">=v1.x", []string{">=", "v1.x"}},
		{"x", []string{"", "x"}},
		{"error", nil},
	}
	for _, tc := range tests {
		if op, v, err := splitComparatorVersion(tc.i); err != nil {
			if tc.p != nil {
				t.Errorf("Invalid for case %q: Expected %q, got error %q", tc.i, tc.p, err)
			}
		} else if op != tc.p[0] {
			t.Errorf("Invalid operator for case %q: Expected %q, got: %q", tc.i, tc.p[0], op)
		} else if v != tc.p[1] {
			t.Errorf("Invalid version for case %q: Expected %q, got: %q", tc.i, tc.p[1], v)
		}

	}
}

func TestBuildVersionRange(t *testing.T) {
	tests := []struct {
		opStr string
		vStr  string
		c     func(comparator) bool
		v     string
	}{
		{">", "1.2.3", testGT, "1.2.3"},
		{">=", "1.2.3", testGE, "1.2.3"},
		{"<", "1.2.3", testLT, "1.2.3"},
		{"<=", "1.2.3", testLE, "1.2.3"},
		{"", "1.2.3", testEQ, "1.2.3"},
		{"=", "1.2.3", testEQ, "1.2.3"},
		{"==", "1.2.3", testEQ, "1.2.3"},
		{"!=", "1.2.3", testNE, "1.2.3"},
		{"!", "1.2.3", testNE, "1.2.3"},
		{">>", "1.2.3", nil, ""},  // Invalid comparator
		{"=", "invalid", nil, ""}, // Invalid version
	}

	for _, tc := range tests {
		if r, err := buildVersionRange(tc.opStr, tc.vStr); err != nil {
			if tc.c != nil {
				t.Errorf("Invalid for case %q: Expected %q, got error %q", strings.Join([]string{tc.opStr, tc.vStr}, ""), tc.v, err)
			}
		} else if r == nil {
			t.Errorf("Invalid for case %q: got nil", strings.Join([]string{tc.opStr, tc.vStr}, ""))
		} else {
			// test version
			if tv := MustParse(tc.v); !r.v.EQ(tv) {
				t.Errorf("Invalid for case %q: Expected version %q, got: %q", strings.Join([]string{tc.opStr, tc.vStr}, ""), tv, r.v)
			}
			// test comparator
			if r.c == nil {
				t.Errorf("Invalid for case %q: got nil comparator", strings.Join([]string{tc.opStr, tc.vStr}, ""))
				continue
			}
			if !tc.c(r.c) {
				t.Errorf("Invalid comparator for case %q\n", strings.Join([]string{tc.opStr, tc.vStr}, ""))
			}
		}
	}

}

func TestVersionRangeToRange(t *testing.T) {
	vr := versionRange{
		v: MustParse("1.2.3"),
		c: compLT,
	}
	rf := vr.rangeFunc()
	if !rf(MustParse("1.2.2")) || rf(MustParse("1.2.3")) {
		t.Errorf("Invalid conversion to range func")
	}
}

func TestRangeAND(t *testing.T) {
	v := MustParse("1.2.2")
	v1 := MustParse("1.2.1")
	v2 := MustParse("1.2.3")
	rf1 := Range(func(v Version) bool {
		return v.GT(v1)
	})
	rf2 := Range(func(v Version) bool {
		return v.LT(v2)
	})
	rf := rf1.AND(rf2)
	if rf(v1) {
		t.Errorf("Invalid rangefunc, accepted: %s", v1)
	}
	if rf(v2) {
		t.Errorf("Invalid rangefunc, accepted: %s", v2)
	}
	if !rf(v) {
		t.Errorf("Invalid rangefunc, did not accept: %s", v)
	}
}

func TestRangeOR(t *testing.T) {
	tests := []struct {
		v Version
		b bool
	}{
		{MustParse("1.2.0"), true},
		{MustParse("1.2.2"), false},
		{MustParse("1.2.4"), true},
	}
	v1 := MustParse("1.2.1")
	v2 := MustParse("1.2.3")
	rf1 := Range(func(v Version) bool {
		return v.LT(v1)
	})
	rf2 := Range(func(v Version) bool {
		return v.GT(v2)
	})
	rf := rf1.OR(rf2)
	for _, tc := range tests {
		if r := rf(tc.v); r != tc.b {
			t.Errorf("Invalid for case %q: Expected %t, got %t", tc.v, tc.b, r)
		}
	}
}

func TestParseRange(t *testing.T) {
	type tv struct {
		v string
		b bool
	}
	tests := []struct {
		i string
		t []tv
	}{
		// Simple expressions
		{">1.2.3", []tv{
			{"1.2.2", false},
			{"1.2.3", false},
			{"1.2.4", true},
		}},
		{">=1.2.3", []tv{
			{"1.2.3", true},
			{"1.2.4", true},
			{"1.2.2", false},
		}},
		{"<1.2.3", []tv{
			{"1.2.2", true},
			{"1.2.3", false},
			{"1.2.4", false},
		}},
		{"<=1.2.3", []tv{
			{"1.2.2", true},
			{"1.2.3", true},
			{"1.2.4", false},
		}},
		{"1.2.3", []tv{
			{"1.2.2", false},
			{"1.2.3", true},
			{"1.2.4", false},
		}},
		{"=1.2.3", []tv{
			{"1.2.2", false},
			{"1.2.3", true},
			{"1.2.4", false},
		}},
		{"==1.2.3", []tv{
			{"1.2.2", false},
			{"1.2.3", true},
			{"1.2.4", false},
		}},
		{"!=1.2.3", []tv{
			{"1.2.2", true},
			{"1.2.3", false},
			{"1.2.4", true},
		}},
		{"!1.2.3", []tv{
			{"1.2.2", true},
			{"1.2.3", false},
			{"1.2.4", true},
		}},
		// Simple Expression errors
		{">>1.2.3", nil},
		{"!1.2.3", nil},
		{"1.0", nil},
		{"string", nil},
		{"", nil},
		{"fo.ob.ar.x", nil},
		// AND Expressions
		{">1.2.2 <1.2.4", []tv{
			{"1.2.2", false},
			{"1.2.3", true},
			{"1.2.4", false},
		}},
		{"<1.2.2 <1.2.4", []tv{
			{"1.2.1", true},
			{"1.2.2", false},
			{"1.2.3", false},
			{"1.2.4", false},
		}},
		{">1.2.2 <1.2.5 !=1.2.4", []tv{
			{"1.2.2", false},
			{"1.2.3", true},
			{"1.2.4", false},
			{"1.2.5", false},
		}},
		{">1.2.2 <1.2.5 !1.2.4", []tv{
			{"1.2.2", false},
			{"1.2.3", true},
			{"1.2.4", false},
			{"1.2.5", false},
		}},
		// OR Expressions
		{">1.2.2 || <1.2.4", []tv{
			{"1.2.2", true},
			{"1.2.3", true},
			{"1.2.4", true},
		}},
		{"<1.2.2 || >1.2.4", []tv{
			{"1.2.2", false},
			{"1.2.3", false},
			{"1.2.4", false},
		}},
		// Wildcard expressions
		{">1.x", []tv{
			{"0.1.9", false},
			{"1.2.6", false},
			{"1.9.0", false},
			{"2.0.0", true},
		}},
		{">1.2.x", []tv{
			{"1.1.9", false},
			{"1.2.6", false},
			{"1.3.0", true},
		}},
		// Combined Expressions
		{">1.2.2 <1.2.4 || >=2.0.0", []tv{
			{"1.2.2", false},
			{"1.2.3", true},
			{"1.2.4", false},
			{"2.0.0", true},
			{"2.0.1", true},
		}},
		{"1.x || >=2.0.x <2.2.x", []tv{
			{"0.9.2", false},
			{"1.2.2", true},
			{"2.0.0", true},
			{"2.1.8", true},
			{"2.2.0", false},
		}},
		{"1.* || >=2.0.* <2.2.*", []tv{
			{"0.9.2", false},
			{"1.2.2", true},
			{"2.0.0", true},
			{"2.1.8", true},
			{"2.2.0", false},
		}},
		{">1.2.2 <1.2.4 || >=2.0.0 <3.0.0", []tv{
			{"1.2.2", false},
			{"1.2.3", true},
			{"1.2.4", false},
			{"2.0.0", true},
			{"2.0.1", true},
			{"2.9.9", true},
			{"3.0.0", false},
		}},
		// Major version only, should be parsed as MAJOR.x
		{"10", []tv{
			{"9.2.2", false},
			{"11.2.3", false},
			{"10.2.4", true},
			{"10.0.0", true},
			{"10.99.99", true},
		}},
		{"0", []tv{
			{"0.2.2", true},
			{"0.0.1", true},
			{"2.0.0", false},
			{"1.0.0", false},
			{"0.99.9999", true},
		}},
		// *'s and X's as wildcards
		{"1.*", []tv{
			{"1.2.2", true},
			{"1.0.1", true},
			{"2.0.0", false},
			{"0.99.99", false},
		}},
		{"1.X", []tv{
			{"1.2.2", true},
			{"1.0.1", true},
			{"2.0.0", false},
			{"0.99.99", false},
		}},
		// wildcard only should match any version
		{"x", []tv{
			{"1.2.2", true},
			{"1.0.1", true},
			{"2.0.0", true},
			{"0.99.99", true},
			{"1021.99.99", true},
		}},
		{"*", []tv{
			{"1.2.2", true},
			{"1.0.1", true},
			{"2.0.0", true},
			{"0.99.99", true},
			{"1021.99.99", true},
		}},
		{"~10.1.2", []tv{
			{"10.1.4", true},
			{"10.1.1", false},
			{"10.2.0", false},
		}},
		{"~10.1.x", []tv{
			{"10.1.4", true},
			{"10.1.1", true},
			{"10.1.0", true},
			{"10.2.0", false},
		}},
		// Should act just like 10.x
		{"~10.x", []tv{
			{"10.1.4", true},
			{"10.1.1", true},
			{"10.2.0", true},
			{"9.9.9", false},
			{"10.99.99", true},
		}},
		// yes, someone wrote this. I don't know why
		{"~7.x || ~8.x || ~9.x", []tv{
			{"7.1.0", true},
			{"6.9.9", false},
			{"8.2.5", true},
			{"9.9.9", true},
			{"10.99.99", false},
			{"10.0.0", false},
		}},
		{"^10.1.2", []tv{
			{"10.1.4", true},
			{"10.1.1", false},
			{"10.2.0", true},
			{"10.99.99", true},
			{"11.0.0", false},
			{"9.0.0", false},
		}},
		{"^10.1.x", []tv{
			{"10.1.4", true},
			{"10.1.1", true},
			{"10.1.0", true},
			{"10.2.0", true},
			{"10.99.99", true},
			{"11.0.0", false},
			{"9.0.0", false},
		}},
		// Should act just like 10.x
		{"^10.x", []tv{
			{"10.1.4", true},
			{"10.1.1", true},
			{"10.2.0", true},
			{"9.9.9", false},
			{"10.99.99", true},
		}},
		{"^10.14.1 || ^8.15.0", []tv{
			{"10.1.4", false},
			{"10.1.1", false},
			{"10.2.0", false},
			{"10.14.0", false},
			{"10.14.1", true},
			{"10.16.1", true},
			{"8.14.0", false},
			{"8.15.0", true},
			{"8.95.0", true},
			{"10.0.0", false},
			{"9.0.0", false},
		}},
		// what would this even mean?
		{"^x", nil},
		// More range tests
		{">=11 <12", []tv{
			{"10.1.4", false},
			{"11.0.0", true},
			{"11.99.99", true},
			{"12.0.0", false},
		}},
		{">=11.x <12.x", []tv{
			{"10.1.4", false},
			{"11.0.0", true},
			{"11.99.99", true},
			{"12.0.0", false},
		}},
		{">=8.* <12", []tv{
			{"10.1.4", true},
			{"11.0.0", true},
			{"8.0.0", true},
			{"7.9.0", false},
			{"11.99.99", true},
			{"12.0.0", false},
		}},
		{">=8 <12", []tv{
			{"10.1.4", true},
			{"11.0.0", true},
			{"8.0.0", true},
			{"7.9.0", false},
			{"11.99.99", true},
			{"12.0.0", false},
		}},
		{">=8.9.1 <9.0", []tv{
			{"8.9.1", true},
			{"8.9.0", false},
			{"9.0.0", false},
			{"8.9.2", true},
			{"10.0.0", false},
		}},
		// This gets converted to >=1.0.0 <4.0.0
		{"1 - 3", []tv{
			{"1.0.0", true},
			{"4.0.0", false},
			{"3.0.0", true},
			{"3.9.2", true},
			{"2.1.3", true},
		}},
		// impossible range
		{">4 <3", nil},
		// Carets behave differently for major versions < 1
		// ^1.2.3 := >=1.2.3 <2.0.0
		// ^0.2.3 := >=0.2.3 <0.3.0
		{"^1.2.3", []tv{
			{"1.2.3", true},
			{"1.2.2", false},
			{"1.8.9", true},
			{"2.0.0", false},
		}},
		{"^0.2.3", []tv{
			{"0.2.3", true},
			{"0.2.2", false},
			{"0.3.1", false},
			{"1.0.0", false},
		}},
		{"v1.2.3", []tv{
			{"1.2.2", false},
			{"1.2.3", true},
			{"1.2.4", false},
		}},
		{"=v1.2.3", []tv{
			{"1.2.2", false},
			{"1.2.3", true},
			{"1.2.4", false},
		}},
		{"^v0.2.3", []tv{
			{"0.2.3", true},
			{"0.2.2", false},
			{"0.3.1", false},
			{"1.0.0", false},
		}},
		{"v1 - v3", []tv{
			{"1.0.0", true},
			{"4.0.0", false},
			{"3.0.0", true},
			{"3.9.2", true},
			{"2.1.3", true},
		}},
	}

	for _, tc := range tests {
		r, err := ParseRange(tc.i)
		if err != nil && tc.t != nil {
			t.Errorf("Error parsing range %q: %s", tc.i, err)
			continue
		}
		for _, tvc := range tc.t {
			v := MustParse(tvc.v)
			if res := r(v); res != tvc.b {
				t.Errorf("Invalid for case %q matching %q: Expected %t, got: %t", tc.i, tvc.v, tvc.b, res)
			}
		}

	}
}

func TestMustParseRange(t *testing.T) {
	testCase := ">1.2.2 <1.2.4 || >=2.0.0 <3.0.0"
	r := MustParseRange(testCase)
	if !r(MustParse("1.2.3")) {
		t.Errorf("Unexpected range behavior on MustParseRange")
	}
}

func TestMustParseRange_panic(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Errorf("Should have panicked")
		}
	}()
	_ = MustParseRange("invalid version")
}

func BenchmarkRangeParseSimple(b *testing.B) {
	const VERSION = ">1.0.0"
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ParseRange(VERSION)
	}
}

func BenchmarkRangeParseAverage(b *testing.B) {
	const VERSION = ">=1.0.0 <2.0.0"
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ParseRange(VERSION)
	}
}

func BenchmarkRangeParseComplex(b *testing.B) {
	const VERSION = ">=1.0.0 <2.0.0 || >=3.0.1 <4.0.0 !=3.0.3 || >=5.0.0"
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ParseRange(VERSION)
	}
}

func BenchmarkRangeMatchSimple(b *testing.B) {
	const VERSION = ">1.0.0"
	r, _ := ParseRange(VERSION)
	v := MustParse("2.0.0")
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		r(v)
	}
}

func BenchmarkRangeMatchAverage(b *testing.B) {
	const VERSION = ">=1.0.0 <2.0.0"
	r, _ := ParseRange(VERSION)
	v := MustParse("1.2.3")
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		r(v)
	}
}

func BenchmarkRangeMatchComplex(b *testing.B) {
	const VERSION = ">=1.0.0 <2.0.0 || >=3.0.1 <4.0.0 !=3.0.3 || >=5.0.0"
	r, _ := ParseRange(VERSION)
	v := MustParse("5.0.1")
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		r(v)
	}
}
