package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"
	"unicode/utf8"

	termbox "github.com/nsf/termbox-go"
)

const (
	Width  = 100
	Height = 50
)

var (
	addr = flag.String("addr", "", "the address of the snek server to connect to")
	wrap = flag.Bool("wrap", false, "whether or not the snek should wrap around the board")

	keyMap = map[termbox.Key]Direction{
		termbox.KeyArrowUp:    Up,
		termbox.KeyArrowDown:  Down,
		termbox.KeyArrowLeft:  Left,
		termbox.KeyArrowRight: Right,
	}

	game *Game
)

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()

	termbox.SetInputMode(termbox.InputEsc)
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	termbox.Flush()

	evChan := make(chan *termbox.Event)
	go listenToTerm(evChan)
	// Our event loop
	run(evChan)
}

func run(evChan chan *termbox.Event) {
	game = newGame(*wrap)

	if *addr != "" {
		go game.goOnline(*addr)
	}

	t := time.NewTicker(75 * time.Millisecond)
	checkTerm()
	for {
		select {
		// Keyboard event
		case ev := <-evChan:
			die := handleEvent(ev)
			if die {
				return
			}
		case <-t.C:
			if !game.update() {
				t.Stop()
				game.clearSnek()
				return
			}
		}
	}
}

func handleEvent(ev *termbox.Event) (die bool) {
	switch ev.Type {
	case termbox.EventKey:
		if ev.Key <= termbox.KeyArrowUp && ev.Key >= termbox.KeyArrowRight {
			if d, ok := keyMap[ev.Key]; ok {
				game.addDirection(d)
			}
		}
		if ev.Key == termbox.KeyCtrlX || ev.Key == termbox.KeyCtrlC {
			die = true
		}
	case termbox.EventResize:
		checkTerm()
	}
	return
}

func checkTerm() {
	w, h := termbox.Size()
	if w < Width || h < Height {
		game.pause()
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
		drawString(w/2, h/2, "Your terminal is too small", fmt.Sprintf("It's currently %dx%d and needs to be %dx%d", w, h, Width, Height))
		termbox.Flush()
	} else {
		game.unpause()
	}
}

func listenToTerm(evChan chan *termbox.Event) {
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventError:
			panic(ev.Err)
		default:
			evChan <- &ev
		}
	}
}

func drawString(x, y int, ss ...string) {
	sy := y - len(ss)/2
	for i, str := range ss {
		for j := 0; j < len(str); j++ {
			sx := x - len(str)/2
			r, _ := utf8.DecodeLastRuneInString(str[j : j+1])
			termbox.SetCell(sx+j, sy+i, r, termbox.ColorWhite, termbox.ColorDefault)
		}
	}
}
