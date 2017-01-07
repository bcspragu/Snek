package main

import (
	"flag"
	"math/rand"
	"time"

	termbox "github.com/nsf/termbox-go"
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
	go listenForKeyboard(evChan)
	// Our event loop
	run(evChan)
}

func run(evChan chan *termbox.Event) {
	w, h := termbox.Size()
	g := newGame(w, h, *wrap)

	if *addr != "" {
		go g.goOnline(*addr)
	}

	t := time.NewTicker(75 * time.Millisecond)
	for {
		select {
		// Keyboard event
		case ev := <-evChan:
			if d, ok := keyMap[ev.Key]; ok {
				g.addDirection(d)
			}
			if ev.Key == termbox.KeyCtrlX || ev.Key == termbox.KeyCtrlC {
				return
			}
		// Update interval
		case <-t.C:
			if !g.update() {
				t.Stop()
				g.clearSnek()
				return
			}
		}
	}
}

func listenForKeyboard(evChan chan *termbox.Event) {
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
}
