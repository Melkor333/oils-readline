package tiling

// various split calculation functions
// TODO: Make them user selectable

type SplitMode int
type SplitFunc func(count int, available rec) []rec

const (
	Vertical SplitMode = iota
	Horizontal
)

type rec struct {
	x, y, width, height int
}

func SplitVertical(c int, available rec) (positions []rec) {
	// we want a border between each 2 nodes
	w := available.width - c + 1
	extra := w % c
	w = w / c
	for range c {
		r := available
		r.width = w
		// width + border
		available.x += w + 1

		if extra > 0 {
			// +1 extra
			r.width += 1
			available.x += 1
			extra--
		}
		positions = append(positions, r)
	}
	return positions
}

func SplitHorizontal(c int, available rec) (positions []rec) {
	// we want a border between each 2 nodes
	h := available.height - c + 1
	extra := h % c
	h = h / c
	for range c {
		r := available
		r.height = h
		// height + border
		available.y += h + 1

		if extra > 0 {
			// +1 extra
			r.height += 1
			// height + border + extra
			available.y += 1
			extra--
		}
		positions = append(positions, r)
	}
	return positions
}

func SplitHorizontalWithMain(c int, available rec) (positions []rec) {
	big := SplitHorizontal(2, available)
	positions = append(positions, big[0])
	small := SplitVertical(c-1, big[1])
	return append(positions, small...)
}

func SplitVerticalWithMain(c int, available rec) (positions []rec) {
	big := SplitVertical(2, available)
	positions = append(positions, big[0])
	small := SplitHorizontal(c-1, big[1])
	return append(positions, small...)
}
