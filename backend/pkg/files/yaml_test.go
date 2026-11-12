package files

import (
	"github.com/ramanmalinouski/Elevators/backend/internal/domain"
	"strings"
	"testing"
)

func TestYAMLRejectsUnknownKeysDuplicateKeysAndDocuments(t *testing.T) {
	for _, text := range []string{"floors: 10\nunknown: true", "floors: 10\nfloors: 12", "floors: 10\n---\nfloors: 12"} {
		var c domain.Config
		if DecodeYAML(strings.NewReader(text), &c) == nil {
			t.Fatalf("accepted malformed config: %s", text)
		}
	}
}

func TestYAMLRoundTrip(t *testing.T) {
	want := domain.Config{Floors: 10, Strategy: domain.MinimumWait, Elevators: []domain.ElevatorConfig{{Capacity: 6, MaxSpeed: 2, Acceleration: 1, InitialFloor: 5}}}
	data, err := EncodeYAML(want)
	if err != nil {
		t.Fatal(err)
	}
	var got domain.Config
	if err = DecodeYAML(strings.NewReader(string(data)), &got); err != nil {
		t.Fatal(err)
	}
	if got.Floors != 10 || got.Elevators[0].InitialFloor != 5 {
		t.Fatalf("bad roundtrip: %+v", got)
	}
	if err = got.Validate(); err != nil {
		t.Fatal(err)
	}
}
