package main

import (
	"context"
	"io"
	"log"
	"math/rand"
	"time"

	"github.com/nsf/termbox-go"
	"google.golang.org/grpc"

	pb "github.com/bcspragu/Snek/proto"
)

type Direction struct {
	X, Y int
}

var (
	Up    = Direction{0, -1}
	Down  = Direction{0, 1}
	Left  = Direction{-1, 0}
	Right = Direction{1, 0}

	oppMap = map[Direction]Direction{
		Up:    Down,
		Down:  Up,
		Left:  Right,
		Right: Left,
	}

	playerColors = []termbox.Attribute{
		termbox.ColorRed,
		termbox.ColorGreen,
		termbox.ColorYellow,
		termbox.ColorBlue,
		termbox.ColorMagenta,
		termbox.ColorCyan,
		termbox.ColorWhite,
	}
)

type Game struct {
	snek          *Snek
	wrap          bool
	food          Loc
	onlineFunc    func(*pb.UpdateRequest) error
	nextDirs      []Direction
	colors        map[int32]termbox.Attribute
	Width, Height int
}

func newGame(w, h int, wrap bool) *Game {
	g := &Game{
		snek:     newSnek(10),
		wrap:     wrap,
		nextDirs: []Direction{},
		colors:   make(map[int32]termbox.Attribute),
		Width:    w,
		Height:   h,
	}
	g.newFood()
	return g
}

func (g *Game) addDirection(d Direction) {
	// Get the last direction
	var ld Direction
	if len(g.nextDirs) == 0 {
		// If our queue is empty, the last direction is the current direction
		ld = g.snek.dir
	} else {
		// If our queue isn't empty, the last direction is at the end of the queue
		ld = g.nextDirs[len(g.nextDirs)-1]
	}

	// If the next direction isn't the opposite of the direction the player wants to go, add it to the queue
	if oppMap[d] != ld {
		g.nextDirs = append(g.nextDirs, d)
	}
}

func (g *Game) goOnline(addr string) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}
	defer conn.Close()

	client := pb.NewSnekClient(conn)
	stream, err := client.Update(context.Background())

	g.onlineFunc = func(req *pb.UpdateRequest) error {
		return stream.Send(req)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Printf("Error reading from server: %v", err)
		}
		if _, ok := g.colors[resp.Id]; !ok {
			g.colors[resp.Id] = playerColors[len(g.colors)%len(playerColors)]
		}
		g.drawSnek(resp)
	}
	stream.CloseSend()
}

func (g *Game) drawSnek(resp *pb.UpdateResponse) {
	// Draw the new head
	hx, hy := int(resp.NewHead.X), int(resp.NewHead.Y)
	c := g.colors[resp.Id]
	termbox.SetCell(hx*2, hy, '█', c, termbox.ColorDefault)
	termbox.SetCell(hx*2+1, hy, '█', c, termbox.ColorDefault)

	// Clear the old tail
	tx, ty := int(resp.OldTail.X), int(resp.OldTail.Y)
	termbox.SetCell(tx*2, ty, ' ', termbox.ColorDefault, termbox.ColorDefault)
	termbox.SetCell(tx*2+1, ty, ' ', termbox.ColorDefault, termbox.ColorDefault)
}

func (g *Game) clearSnek() {
	time.Sleep(time.Second)
	for _, l := range g.snek.body {
		termbox.SetCell(l.X*2, l.Y, ' ', termbox.ColorDefault, termbox.ColorDefault)
		termbox.SetCell(l.X*2+1, l.Y, ' ', termbox.ColorDefault, termbox.ColorDefault)
		termbox.Flush()
		time.Sleep(50 * time.Millisecond)
	}
}

func (g *Game) newFood() {
	g.food = Loc{X: rand.Intn(g.Width / 2), Y: rand.Intn(g.Height / 2)}
}

type Snek struct {
	body     []Loc // head is at body[len(body)-1]
	dir      Direction
	occupied map[Loc]struct{}
}

func newSnek(l int) *Snek {
	b := make([]Loc, l)
	o := make(map[Loc]struct{})
	for i := 0; i < l; i++ {
		b[i].X = i
		o[b[i]] = struct{}{}
	}
	return &Snek{
		body:     b,
		dir:      Right,
		occupied: o,
	}
}

type Loc struct {
	X, Y int
}

// Returns whether or not we were successful
func (s *Snek) addHead(l Loc) bool {
	if _, ok := s.occupied[l]; ok {
		// It's already occupied, fail it
		return false
	}
	s.occupied[l] = struct{}{}
	s.body = append(s.body, l)
	return true
}

func (s *Snek) removeTail() {
	delete(s.occupied, s.body[0])
	s.body = s.body[1:]
}

func (s *Snek) tail() Loc {
	return s.body[0]
}

func (s *Snek) grow() {
	s.body = append(s.body[0:1], s.body...)
}

func (s *Snek) head() Loc {
	return s.body[len(s.body)-1]
}

func (g *Game) addHead() (Loc, bool) {
	h := g.snek.head()
	nh := Loc{h.X + g.snek.dir.X, h.Y + g.snek.dir.Y}

	if g.wrap {
		if nh.X < 0 {
			nh.X = g.Width / 2
		}
		if nh.X > g.Width/2 {
			nh.X = 0
		}
		if nh.Y < 0 {
			nh.Y = g.Height
		}
		if nh.Y > g.Height {
			nh.Y = 0
		}
	} else {
		// We aren't wrapping, kill them if they go too far
		if nh.X < 0 || nh.Y < 0 || nh.X >= g.Width/2 || nh.Y >= g.Height {
			return nh, false
		}
	}
	return nh, g.snek.addHead(nh)
}

func (g *Game) updateDir() {
	if len(g.nextDirs) > 0 {
		g.snek.dir, g.nextDirs = g.nextDirs[0], g.nextDirs[1:]
	}
}

func (g *Game) update() bool {
	g.updateDir()

	h, ok := g.addHead()
	if !ok {
		return false
	}
	// draw the new head
	termbox.SetCell(h.X*2, h.Y, '█', termbox.ColorWhite, termbox.ColorDefault)
	termbox.SetCell(h.X*2+1, h.Y, '█', termbox.ColorWhite, termbox.ColorDefault)

	if h == g.food {
		g.snek.grow()
		// Clear the cell
		termbox.SetCell(g.food.X*2+1, g.food.Y, '█', termbox.ColorWhite, termbox.ColorDefault)
		g.newFood()
	}
	// draw food
	termbox.SetCell(g.food.X*2+1, g.food.Y, '◎', termbox.ColorWhite, termbox.ColorDefault)

	t := g.snek.tail()
	g.snek.removeTail()
	// clear the tail
	termbox.SetCell(t.X*2, t.Y, ' ', termbox.ColorDefault, termbox.ColorDefault)
	termbox.SetCell(t.X*2+1, t.Y, ' ', termbox.ColorDefault, termbox.ColorDefault)

	if g.onlineFunc != nil {
		g.onlineFunc(&pb.UpdateRequest{
			NewHead: &pb.Loc{X: int32(h.X), Y: int32(h.Y)},
			OldTail: &pb.Loc{X: int32(t.X), Y: int32(t.Y)},
		})
	}

	termbox.Flush()
	return true
}
