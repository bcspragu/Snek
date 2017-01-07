package main

import (
	"github.com/nsf/termbox-go"
	"math/rand"
	"time"
)

type Direction struct {
	X, Y int
}

var (
	Up    = Direction{0, -1}
	Down  = Direction{0, 1}
	Left  = Direction{-1, 0}
	Right = Direction{1, 0}

	keyMap = map[termbox.Key]Direction{
		termbox.KeyArrowUp:    Up,
		termbox.KeyArrowDown:  Down,
		termbox.KeyArrowLeft:  Left,
		termbox.KeyArrowRight: Right,
	}

	oppMap = map[Direction]Direction{
		Up:    Down,
		Down:  Up,
		Left:  Right,
		Right: Left,
	}
)

type Game struct {
	snek          *Snake
	food          Loc
	nextDirs      []Direction
	Width, Height int
}

func newGame(w, h int) *Game {
	g := &Game{
		snek:     newSnake(25),
		nextDirs: []Direction{},
		Width:    w,
		Height:   h,
	}
	g.newFood()
	return g
}

func (g *Game) newFood() {
	g.food = Loc{X: rand.Intn(g.Width / 2), Y: rand.Intn(g.Height / 2)}
}

type Snake struct {
	body     []Loc // head is at body[len(body)-1]
	dir      Direction
	occupied map[Loc]struct{}
}

func newSnake(l int) *Snake {
	b := make([]Loc, l)
	o := make(map[Loc]struct{})
	for i := 0; i < l; i++ {
		b[i].X = i
		o[b[i]] = struct{}{}
	}
	return &Snake{
		body:     b,
		dir:      Right,
		occupied: o,
	}
}

type Loc struct {
	X, Y int
}

// Returns whether or not we were successful
func (s *Snake) addHead(l Loc) bool {
	if _, ok := s.occupied[l]; ok {
		// It's already occupied, fail it
		return false
	}
	s.occupied[l] = struct{}{}
	s.body = append(s.body, l)
	return true
}

func (s *Snake) removeTail() {
	delete(s.occupied, s.body[0])
	s.body = s.body[1:]
}

func (s *Snake) tail() Loc {
	return s.body[0]
}

func (s *Snake) grow() {
	s.body = append(s.body[0:1], s.body...)
}

func (s *Snake) head() Loc {
	return s.body[len(s.body)-1]
}

func (s *Snake) nextHead() Loc {
	h := s.head()
	return Loc{h.X + s.dir.X, h.Y + s.dir.Y}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()
	w, h := termbox.Size()
	g := newGame(w, h)

	termbox.SetInputMode(termbox.InputEsc)
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	termbox.Flush()
	evChan := make(chan *termbox.Event)
	go func() {
		for {
			switch ev := termbox.PollEvent(); ev.Type {
			case termbox.EventKey:
				if ev.Key == termbox.KeyCtrlX || ev.Key == termbox.KeyCtrlC || (ev.Key <= termbox.KeyArrowUp && ev.Key >= termbox.KeyArrowRight) {
					evChan <- &ev
				}
			case termbox.EventError:
				panic(ev.Err)
			}
		}
	}()
	t := time.NewTicker(75 * time.Millisecond)
loop:
	for {
		select {
		case ev := <-evChan:
			if d, ok := keyMap[ev.Key]; ok {
				var nd Direction
				if len(g.nextDirs) == 0 {
					nd = g.snek.dir
				} else {
					nd = g.nextDirs[len(g.nextDirs)-1]
				}

				if oppMap[d] != nd {
					g.addDirection(d)
				}
			}
			if ev.Key == termbox.KeyCtrlX || ev.Key == termbox.KeyCtrlC {
				break loop
			}
		case <-t.C:
			if !g.update() {
				t.Stop()
				g.clearSnake()
				break loop
			}
		}
	}
}

func (g *Game) addDirection(d Direction) {
	g.nextDirs = append(g.nextDirs, d)
}

func (g *Game) clearSnake() {
	time.Sleep(time.Second)
	for _, l := range g.snek.body {
		termbox.SetCell(l.X*2, l.Y, ' ', termbox.ColorDefault, termbox.ColorDefault)
		termbox.SetCell(l.X*2+1, l.Y, ' ', termbox.ColorDefault, termbox.ColorDefault)
		termbox.Flush()
		time.Sleep(50 * time.Millisecond)
	}
}

func (g *Game) update() bool {
	if len(g.nextDirs) > 0 {
		g.snek.dir, g.nextDirs = g.nextDirs[0], g.nextDirs[1:]
	}
	h := g.snek.nextHead()
	if h.X < 0 || h.Y < 0 || h.X >= g.Width/2 || h.Y >= g.Height || !g.snek.addHead(h) {
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

	termbox.Flush()
	return true
}
