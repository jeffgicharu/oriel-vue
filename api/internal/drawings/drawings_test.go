package drawings

import (
	"encoding/xml"
	"strings"
	"testing"
)

// Each drawing is a whole SVG document that parses, with its title for
// screen readers.
func TestTheDrawingsAreWholeSVG(t *testing.T) {
	for name, svg := range map[string]string{"sun path": SunPath(), "bay section": BaySection(), "map": LocationMap()} {
		if !strings.HasPrefix(svg, "<svg") || !strings.HasSuffix(svg, "</svg>") {
			t.Errorf("the %s is not one SVG element", name)
		}
		if !strings.Contains(svg, "<title") {
			t.Errorf("the %s has no title", name)
		}
		dec := xml.NewDecoder(strings.NewReader(svg))
		for {
			if _, err := dec.Token(); err != nil {
				if err.Error() != "EOF" {
					t.Errorf("the %s does not parse: %v", name, err)
				}
				break
			}
		}
	}
	if strings.Count(SunPath(), "<polyline") != 3 {
		t.Error("the sun path should trace three days")
	}
}

// The elevation shows every bay on the south face: three on each of the
// fourteen typical floors and two on the penthouse floor.
func TestTheElevationDrawsEveryBay(t *testing.T) {
	if got := elevationBays(); got != 14*3+2 {
		t.Fatalf("the elevation draws %d bays", got)
	}
	strokes := ElevationStrokes()
	if len(strokes) < 20 || len(strokes) > 200 {
		t.Fatalf("the elevation is drawn in %d strokes", len(strokes))
	}
	for i := 1; i < len(strokes); i++ {
		if strokes[i].Order < strokes[i-1].Order {
			t.Fatal("the strokes do not rise storey by storey")
		}
	}
}
