package main

import (
	"errors"
	"fmt"
	"github.com/fogleman/gg"
	"gopkg.in/yaml.v3"
	"image/color"
	"log"
	"os"
	"strings"
)

func main() {
	fmt.Println(len(os.Args), os.Args)
	fileName := "uml.yaml"
	for i, arg := range os.Args {
		if i == 0 {
			continue
		}
		if arg == "--file" {
			fileName = os.Args[i+1]
		}
	}
	d, err := DiagramFromYaml(fileName)
	if err != nil {
		log.Fatalf("err %v", err)
	}

	//defer os.Remove("test.png")
	//
	d.Render()
}

// Edge represents an Edge on a Sequence Diagram
type Edge struct {
	From  string `yaml:"from"`
	To    string `yaml:"to"`
	Label string `yaml:"label"`
	Type  string `yaml:"type"`
}

type config struct {
	Title        string   `yaml:"title"`
	Participants []string `yaml:"participants"`
	Edges        []Edge   `yaml:"edges"`
}

// DiagramFromYaml create a Diagram from a YAML file
func DiagramFromYaml(file string) (*Diagram, error) {
	var cfg config
	name := strings.TrimSuffix(file, ".yaml")

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	d := NewDiagram(name)

	d.SetTitle(cfg.Title)

	for _, partecipant := range cfg.Participants {
		d.AddParticipants(partecipant)
	}

	for _, edge := range cfg.Edges {
		switch edge.Type {
		case "->":
			err = d.AddDirectionalEdge(edge.From, edge.To, edge.Label)
		case "-":
			err = d.AddUndirectionalEdge(edge.From, edge.To, edge.Label)
		default:
			err = errors.New("edge type not valid")
		}
	}

	if err != nil {
		return nil, err
	}

	return d, nil
}

const (
	participantBoxWidth  = 90.0
	participantBoxHeight = 50.0
	participantsPadding  = 32

	rectangleStrokeWidth = 2.0
	lineStrokeWidth      = 1.0

	verticalSpaceBetweenEdges = 50

	width  = 1000
	height = 1000
)

// Diagram represents a diagram
type Diagram struct {
	participants         []participant
	edges                []edge
	renderedParticipants []*participant
	participantsCoordMap map[string]participantCoord

	dc       *gg.Context
	title    string
	filename string
}

// NewDiagram init function
func NewDiagram(filename string) *Diagram {
	coordMap := make(map[string]participantCoord)

	return &Diagram{
		participantsCoordMap: coordMap,
		filename:             filename,
	}
}

// Render generates an image from a `Diagram` object
func (d *Diagram) Render() {
	width := width
	if len(d.participants) > 5 {
		width = width + ((len(d.participants) - 5) * 150)
	}
	height := height
	if len(d.edges) > 15 {
		height = height + ((len(d.edges) - 15) * 50)
	}
	d.dc = gg.NewContext(width, height)
	d.dc.DrawRectangle(0, 0, float64(width), float64(height))
	d.dc.SetColor(color.White)
	d.dc.Fill()

	d.renderTitle()
	d.renderParticipants()
	d.renderEdges()

	d.dc.SavePNG(fmt.Sprintf("%s.png", d.filename))
}

func (d *Diagram) renderTitle() {
	s := d.title
	textWidth, _ := d.dc.MeasureString(s)
	centerX := float64(d.dc.Width())/2.0 - float64(textWidth)/2.0
	log.Printf("title: %s", d.title)
	d.dc.SetColor(color.Black)
	d.dc.DrawString(s, centerX, height*0.05)
	d.dc.Stroke()
}

func (d *Diagram) renderParticipants() {
	for idx := range d.participants {
		p := &d.participants[idx]

		for rIdx := range d.renderedParticipants {
			if d.renderedParticipants[rIdx].Name == p.Name {
				return
			}
		}
		spacePerBlock := float64(d.dc.Width() / len(d.participants))
		strWidth, strHeight := d.dc.MeasureString(p.Name)
		startX := spacePerBlock*float64(len(d.renderedParticipants)+1) - spacePerBlock/2 - participantsPadding - strWidth/2

		endX := startX + participantBoxWidth + strWidth
		startY := height * 0.1 // 10% from the top
		endY := startY + participantBoxHeight
		// draw the border
		d.dc.SetColor(color.Black)
		d.dc.SetLineWidth(rectangleStrokeWidth)
		d.dc.SetFillRule(gg.FillRuleEvenOdd)

		d.dc.DrawLine(startX, startY, endX, startY)
		d.dc.Stroke()

		d.dc.DrawLine(startX, endY, endX, endY)
		d.dc.Stroke()

		d.dc.DrawLine(startX, startY, startX, endY)
		d.dc.Stroke()

		d.dc.DrawLine(endX, startY, endX, endY)
		d.dc.Stroke()

		d.dc.SetColor(color.Gray{Y: 230})
		d.dc.DrawRectangle(startX, startY, participantBoxWidth+strWidth, participantBoxHeight)
		d.dc.SetColor(color.Black)
		centerStrWidth := startX + ((endX - startX) / 2) - strWidth/2
		centerStrHeight := (endY-startY)/2 + startY + (strHeight / 2)

		d.dc.DrawString(p.Name, centerStrWidth, centerStrHeight)
		d.dc.Stroke()

		// render vertical action line for each participant
		centerX := startX + (endX-startX)/2 - 2.5
		lineStartY := endY + 2.5
		lineEndY := float64(len(d.edges)*(verticalSpaceBetweenEdges)) + lineStartY + verticalSpaceBetweenEdges // padding

		d.dc.SetLineWidth(lineStrokeWidth)
		d.dc.DrawLine(centerX, lineStartY, centerX, lineEndY)
		d.dc.Stroke()
		d.renderedParticipants = append(d.renderedParticipants, p)

		d.participantsCoordMap[p.Name] = participantCoord{
			X: startX,
			Y: startY,
		}
	}
}

func (d *Diagram) renderEdges() {
	renderedEdges := 0

	for idx := range d.edges {
		e := &d.edges[idx]
		fromStrWidth, _ := d.dc.MeasureString(e.from.Name)
		toStrWidth, _ := d.dc.MeasureString(e.to.Name)
		fromCords := d.participantsCoordMap[e.from.Name]
		toCords := d.participantsCoordMap[e.to.Name]
		startX := fromCords.X + participantBoxWidth/2 - 2.5 + fromStrWidth/2 // 2.5 = half of stroke width
		startY := fromCords.Y + participantBoxHeight + 2.5 + float64((1+renderedEdges)*verticalSpaceBetweenEdges)
		endX := toCords.X + participantBoxWidth/2 - 2.5 + toStrWidth/2
		isReverseEdge := endX < startX

		d.dc.SetDash(6)
		d.dc.DrawLine(startX, startY, endX, startY)
		d.dc.Stroke()

		d.dc.SetDash()

		if e.directional {
			arrowTipStartX := endX
			var arrowTipEndX float64

			if isReverseEdge {
				arrowTipEndX = arrowTipStartX + 10
			} else {
				arrowTipEndX = arrowTipStartX - 10
			}
			d.dc.DrawLine(arrowTipStartX, startY, arrowTipEndX, startY-10)
			d.dc.DrawLine(arrowTipStartX, startY, arrowTipEndX, startY+10)
			d.dc.Stroke()
		}

		if e.Label != "" {
			textWidth, textHeight := d.dc.MeasureString(e.Label)
			textY := startY + textHeight
			textX := startX
			if isReverseEdge {
				textX -= participantsPadding / 2
				textX -= textWidth
			} else {
				textX += participantsPadding / 2
			}

			d.dc.DrawString(e.Label, textX, textY)
		}

		renderedEdges++
	}
}

// AddParticipants sets the `participant` array on the Diagram object
func (d *Diagram) AddParticipants(name ...string) {
	for _, n := range name {
		for i := range d.participants {
			if d.participants[i].Name == n {
				return
			}
		}
		d.participants = append(d.participants, participant{Name: n})
	}
}

// AddDirectionalEdge adds a connection (renders as an arrowed line) between two participants
func (d *Diagram) AddDirectionalEdge(from, to string, label string) error {
	var fromPar *participant
	var toPar *participant
	for i := range d.participants {
		if d.participants[i].Name == from {
			fromPar = &d.participants[i]
		}
		if d.participants[i].Name == to {
			toPar = &d.participants[i]
		}
	}
	if fromPar == nil {
		panic(fmt.Sprintf("participant \"%s\" not found", from))
	}
	if toPar == nil {
		panic(fmt.Sprintf("participant \"%s not found", to))
	}

	d.edges = append(d.edges, edge{from: *fromPar, to: *toPar, Label: label, directional: true})
	return nil
}

// AddUndirectionalEdge adds a connection (renders as a line) between two participants
func (d *Diagram) AddUndirectionalEdge(from, to string, label string) error {
	var fromPar *participant
	var toPar *participant
	for i := range d.participants {
		if d.participants[i].Name == from {
			fromPar = &d.participants[i]
		}
		if d.participants[i].Name == to {
			toPar = &d.participants[i]
		}
	}
	if fromPar == nil || toPar == nil {
		return fmt.Errorf("participant not found")
	}

	d.edges = append(d.edges, edge{from: *fromPar, to: *toPar, Label: label, directional: false})
	return nil
}

// SetTitle sets the diagram's title
func (d *Diagram) SetTitle(s string) {
	d.title = s
}

type participant struct {
	Name string
}

type edge struct {
	from        participant
	to          participant
	directional bool
	Label       string
}

func (e *edge) From() participant {
	return e.from
}

func (e *edge) To() participant {
	return e.to
}

type participantCoord struct {
	X float64
	Y float64
}
