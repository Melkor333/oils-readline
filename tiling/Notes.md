
Border calc:
l 10 - borderwidth
1: 0, 5
2: 6, 4
3: 11, 4

	   . . . . . | . . . . | . . . .
	   . . . . . | . . . . | . . . .
	   . . . . . | . . . . | . . . .
	   . . . . . | . . . . | . . . .
	   . . . . . | . . . . | . . . .
	   . . . . . | . . . . | . . . .
	   . . . . . | . . . . | . . . .

We can't just set a lipgloss.Border, because then we're missing the edges.
Therefore we have a bitmask to tell which cell has what kinds of borders and merge them.
