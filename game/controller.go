package game

import (
	"time"
)

const animationSpeed = 10 * time.Millisecond

func Go_termbox(c Client) {
	err := Init(c)
	if err != nil {
		panic(err)
	}
	defer Close(c)

	event_queue := make(chan Event)

	go func() {
		for {
			event_queue <- PollEvent()
		}
	}()

	g := NewGame()
	render(g, c)

	for {
		ev := <-event_queue
		if ev.Type == EventKey {
			switch {
			case ev.Key == KeyArrowUp || ev.Ch == 'k':
				g.move(UP)
			case ev.Key == KeyArrowDown || ev.Ch == 'j':
				g.move(DOWN)
			case ev.Key == KeyArrowLeft || ev.Ch == 'h':
				g.move(LEFT)
			case ev.Key == KeyArrowRight || ev.Ch == 'l':
				g.move(RIGHT)
			case ev.Ch == 'n':
				g.nextLevel()
			case ev.Ch == 'p':
				g.prevLevel()
			case ev.Ch == 'r':
				g.reset()
			case ev.Ch == 'd':
				g.toggleDebug()
			case ev.Key == KeyEsc:
				return
			}
		}
		render(g, c)
		time.Sleep(animationSpeed)
	}
}
