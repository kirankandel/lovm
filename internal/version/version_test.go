package version

import (
	"encoding/json"
	"testing"
)

func TestParseBuild(t *testing.T) {
	tests := []struct {
		in      string
		want    Build
		wantErr bool
	}{
		{in: "24.8.4.2", want: Build{24, 8, 4, 2}},
		{in: "6.0.7.3", want: Build{6, 0, 7, 3}},
		{in: "24.8.4", wantErr: true},
		{in: "24.8.4.2.1", wantErr: true},
		{in: "24.x.4.2", wantErr: true},
		{in: "24..4.2", wantErr: true},
		{in: "+24.8.4.2", wantErr: true},
		{in: "", wantErr: true},
	}
	for _, tt := range tests {
		got, err := ParseBuild(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseBuild(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseBuild(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestBuildFormatting(t *testing.T) {
	b := Build{24, 8, 4, 2}
	if got := b.String(); got != "24.8.4.2" {
		t.Errorf("String() = %q", got)
	}
	if got := b.Triple(); got != "24.8.4" {
		t.Errorf("Triple() = %q", got)
	}
	if got := b.Branch(); got != "24.8" {
		t.Errorf("Branch() = %q", got)
	}
}

func TestBuildCompare(t *testing.T) {
	tests := []struct {
		a, b Build
		want int
	}{
		{Build{24, 8, 4, 2}, Build{24, 8, 4, 2}, 0},
		{Build{24, 8, 4, 2}, Build{24, 8, 10, 1}, -1},
		{Build{25, 2, 0, 1}, Build{24, 8, 7, 2}, 1},
	}
	for _, tt := range tests {
		if got := tt.a.Compare(tt.b); got != tt.want {
			t.Errorf("%v.Compare(%v) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestBuildJSONRoundTrip(t *testing.T) {
	in := map[Build]bool{{24, 8, 4, 2}: true}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"24.8.4.2":true}` {
		t.Fatalf("Marshal = %s", data)
	}
	var out map[Build]bool
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !out[Build{24, 8, 4, 2}] {
		t.Fatalf("Unmarshal = %v", out)
	}
}

func TestParseSpec(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "latest", want: "latest"},
		{in: "24.8", want: "24.8"},
		{in: " 24.8.4\n", want: "24.8.4"},
		{in: "24.8.4.2", want: "24.8.4.2"},
		{in: "24", wantErr: true},
		{in: "v24.8", wantErr: true},
		{in: "24.8.4.2.1", wantErr: true},
		{in: "", wantErr: true},
	}
	for _, tt := range tests {
		got, err := ParseSpec(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseSpec(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if err == nil && got.String() != tt.want {
			t.Errorf("ParseSpec(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSpecMatches(t *testing.T) {
	b := Build{24, 8, 4, 2}
	tests := []struct {
		spec string
		want bool
	}{
		{"latest", true},
		{"24.8", true},
		{"24.8.4", true},
		{"24.8.4.2", true},
		{"24.8.5", false},
		{"24.2", false},
		{"24.8.4.1", false},
	}
	for _, tt := range tests {
		if got := mustSpec(t, tt.spec).Matches(b); got != tt.want {
			t.Errorf("%s.Matches(%v) = %v, want %v", tt.spec, b, got, tt.want)
		}
	}
}

func TestSpecProperties(t *testing.T) {
	if !mustSpec(t, "24.8.4.2").IsExact() || mustSpec(t, "24.8").IsExact() {
		t.Error("IsExact should be true only for four-part specs")
	}
	if !mustSpec(t, "5.4").BelowFloor() || mustSpec(t, "6.0").BelowFloor() || mustSpec(t, "latest").BelowFloor() {
		t.Error("BelowFloor should be true only below 6.0")
	}
	if got := mustSpec(t, "24.9").Lower(); got != (Build{24, 9, 0, 0}) {
		t.Errorf("Lower() = %v", got)
	}
}

func mustSpec(t *testing.T, s string) Spec {
	t.Helper()
	spec, err := ParseSpec(s)
	if err != nil {
		t.Fatal(err)
	}
	return spec
}
