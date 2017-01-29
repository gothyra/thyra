package server

type Screen struct {
	width       int
	height      int
	promptBar   PromptBar
	mapCanvas   [][]rune
	screenRunes [][]rune
}

func (scr *Screen) init(p *Client) {

	scr.screenRunes = make([][]rune, p.w)
	for w := 0; w < p.w; w++ {
		scr.screenRunes[w] = make([]rune, p.h)
		for h := 0; h < p.h; h++ {
			scr.screenRunes[w][h] = ' '
		}
	}

}
