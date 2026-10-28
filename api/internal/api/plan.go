package api

import (
	"math"

	"github.com/jeffgicharu/oriel-vue/api/internal/floorplan"
)

// Plan is a home's floor plan as geometry, for the site to draw. Rectangles
// and lines are [x0, y0, x1, y1] in metres, with y running from the main
// façade inwards.
type Plan struct {
	Type        string       `json:"type"`
	W           float64      `json:"w"`
	D           float64      `json:"d"`
	Rooms       []PlanRoom   `json:"rooms"`
	Doors       []PlanDoor   `json:"doors"`
	Openings    [][4]float64 `json:"openings"`
	Windows     [][4]float64 `json:"windows"`
	Bay         [4]float64   `json:"bay"`
	BayCorner   bool         `json:"bayCorner"`
	Outdoor     [4]float64   `json:"outdoor"`
	OutdoorName string       `json:"outdoorName"`
	OutdoorMore [][4]float64 `json:"outdoorMore"`
	Items       []PlanItem   `json:"items"`
}

// PlanRoom is one room, made of one rectangle or a few.
type PlanRoom struct {
	Name  string       `json:"name"`
	Kind  string       `json:"kind"`
	Rects [][4]float64 `json:"rects"`
	Area  float64      `json:"area"`
	Size  string       `json:"size"`
}

// PlanDoor is a door in a wall: on the wall at x (vertical) or y, from its
// start along the wall for its width.
type PlanDoor struct {
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width"`
	Vertical   bool    `json:"vertical"`
	Into       int     `json:"into"`
	HingeAtEnd bool    `json:"hingeAtEnd"`
	Entrance   bool    `json:"entrance"`
	Service    bool    `json:"service"`
}

// PlanItem is a piece of furniture or a fitting.
type PlanItem struct {
	Kind string     `json:"kind"`
	R    [4]float64 `json:"r"`
}

func cm(v float64) float64 { return math.Round(v*100) / 100 }

func rectOf(r floorplan.Rect) [4]float64 { return [4]float64{cm(r.X0), cm(r.Y0), cm(r.X1), cm(r.Y1)} }

func lineOf(l floorplan.Line) [4]float64 { return [4]float64{cm(l.X0), cm(l.Y0), cm(l.X1), cm(l.Y1)} }

func planOf(p floorplan.Plan) Plan {
	out := Plan{
		Type: p.Type, W: cm(p.W), D: cm(p.D),
		Rooms: []PlanRoom{}, Doors: []PlanDoor{}, Openings: [][4]float64{}, Windows: [][4]float64{},
		Bay: rectOf(p.Bay), BayCorner: p.BayCorner, Outdoor: rectOf(p.Outdoor), OutdoorName: p.OutdoorName,
		OutdoorMore: [][4]float64{}, Items: []PlanItem{},
	}
	for _, r := range p.Rooms {
		room := PlanRoom{Name: r.Name, Kind: r.Kind, Rects: [][4]float64{}, Area: math.Round(r.Area()*10) / 10, Size: r.Size()}
		for _, rect := range r.Rects {
			room.Rects = append(room.Rects, rectOf(rect))
		}
		out.Rooms = append(out.Rooms, room)
	}
	for _, d := range p.Doors {
		out.Doors = append(out.Doors, PlanDoor{X: cm(d.X), Y: cm(d.Y), Width: cm(d.Width), Vertical: d.Vertical, Into: d.Into, HingeAtEnd: d.HingeAtEnd, Entrance: d.Entrance, Service: d.Service})
	}
	for _, l := range p.Openings {
		out.Openings = append(out.Openings, lineOf(l))
	}
	for _, l := range p.Windows {
		out.Windows = append(out.Windows, lineOf(l))
	}
	for _, r := range p.OutdoorMore {
		out.OutdoorMore = append(out.OutdoorMore, rectOf(r))
	}
	for _, it := range p.Items {
		out.Items = append(out.Items, PlanItem{Kind: it.Kind, R: rectOf(it.R)})
	}
	return out
}
