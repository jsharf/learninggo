package main

import (
	"bufio"
	"fmt"
	"os"
)

type Color int

const (
	none Color = iota
	white
	black
	scored
	unscored
	tie
)

const (
	width  = 3
	height = 3
)

const (
	not_done = iota
	done
)

type Position struct {
	x int
	y int
}

type GameState struct {
	board           [width][height]Color
	black_prisoners int
	white_prisoners int
	state           int
	kohs            map[Position]bool
}

func (s GameState) lookup(pos Position) Color {
	return s.board[pos.x][pos.y]
}

func (s GameState) print_board() {
	for _, row := range s.board {
		for _, elem := range row {
			switch elem {
			case white:
				fmt.Print("W")
			case black:
				fmt.Print("B")
			default:
				fmt.Print("+")
			}
		}
		fmt.Println()
	}
}

func (c Color) invert() Color {
	var opposite_color Color
	switch c {
	case black:
		opposite_color = white
	case white:
		opposite_color = black
	default:
		panic("FUCK YOU JACOB")
	}
	return opposite_color
}

func (s GameState) can_place(pos Position, c Color) bool {
	if s.lookup(pos) != none {
		return false
	}

	s.board[pos.x][pos.y] = c
	count, _ := liberties(&s, pos, map[Position]bool{})

	return count != 0
}

// Go through all possible moves on a particular gamestate. Return list of
// resulting gamestates.
func (s GameState) get_children(c Color) []GameState {
	var children []GameState
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			pos := Position{x, y}
			if s.kohs[pos] {
				continue
			}
			if s.lookup(pos) == none {
				if !s.can_place(pos, c) {
					continue
				}
				child := s
				child.kohs = map[Position]bool{}
				child.board[x][y] = c
				children = append(children, child)
			}
		}
	}
	for i := range children {
		for x := 0; x < width; x++ {
			for y := 0; y < height; y++ {
				count := remove_dead_region(&children[i], Position{x, y}, c)
				if count == 1 {
					children[i].kohs[Position{x, y}] = true
				}
			}
		}
	}
	return children
}

func position_valid(pos Position) bool {
	return pos.x >= 0 && pos.x < width && pos.y >= 0 && pos.y < height
}

func (pos Position) neighbors() []Position {
	up := Position{pos.x, pos.y + 1}
	down := Position{pos.x, pos.y - 1}
	left := Position{pos.x - 1, pos.y}
	right := Position{pos.x + 1, pos.y}
	potential_neighbors := []Position{up, down, left, right}
	var neighbor_positions []Position
	for _, potential_neighbor := range potential_neighbors {
		if position_valid(potential_neighbor) {
			neighbor_positions = append(neighbor_positions, potential_neighbor)
		}
	}
	return neighbor_positions
}

// If Position pos indicates a "none" location on the board, count the size of
// that territory and return the color it belongs to (none if it's no one's
// territory). Mark the territory as "scored".
func explore_territory(s GameState, pos Position) (result_count int, result_color Color) {
	point := s.lookup(pos)
	if point != none {
		return 0, point
	}

	result_color = unscored
	result_count = 1

	neighbor_positions := pos.neighbors()

	for _, neighbor := range neighbor_positions {
		neighbor_count, neighbor_color := explore_territory(s, neighbor)
		if result_color == neighbor_color || result_color == unscored {
			result_color = neighbor_color
		} else {
			result_color = none
		}
		result_count += neighbor_count
	}
	if result_color == unscored {
		result_color = none
	}
	s.board[pos.x][pos.y] = scored
	return
}

// Returns (# liberties, objects in group)
func liberties(s *GameState, pos Position, visited map[Position]bool) (int, []Position) {
	c := s.lookup(pos)
	visited[pos] = true
	var sum int = 0
	var positions []Position
	positions = append(positions, pos)
	for _, neighbor := range pos.neighbors() {
		if visited[neighbor] {
			continue
		}
		switch s.lookup(neighbor) {
		case c:
			count, region := liberties(s, neighbor, visited)
			sum += count
			positions = append(positions, region...)
		case none:
			visited[neighbor] = true
			sum++
		}
	}
	return sum, positions
}

func remove_dead_region(s *GameState, pos Position, c Color) int {
	point := s.lookup(pos)
	if point != c {
		return 0
	}

	count, positions := liberties(s, pos, map[Position]bool{})

	if count == 0 {
		for _, pos := range positions {
			s.board[pos.x][pos.y] = none
		}
	}

	return len(positions)
}

// Returns winning color, score.
func score(s GameState) (Color, int) {
	white_score := 0
	black_score := 0
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			pos := Position{x, y}
			if s.lookup(pos) != scored {
				count, color := explore_territory(s, pos)
				if color == white {
					white_score += count
				}
				if color == black {
					black_score += count
				}
			}
		}
	}
	white_score += s.black_prisoners
	black_score += s.white_prisoners
	if white_score > black_score {
		return white, white_score
	}
	if black_score > white_score {
		return black, black_score
	}
	return tie, black_score
}

func max(i, j int) int {
	if i > j {
		return i
	}
	return j
}

func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}

func minimax(s GameState, depth int, c Color, orig Color) int {
	children := s.get_children(c)
	if len(children) == 0 || depth == 0 {
		winner, count := score(s)
		if winner == black {
			count *= -1
		}
		return count
	}

	if c == orig {
		best := -(1<<31 - 1)
		for _, child := range children {
			v := minimax(child, depth-1, c.invert(), orig)
			best = max(best, v)
		}
		return best
	}

	worst := (1<<31 - 1)
	for _, child := range children {
		v := minimax(child, depth-1, c.invert(), orig)
		worst = min(worst, v)
	}
	return worst
}

const depth = 1000

func make_turn(s GameState, c Color) GameState {
	children := s.get_children(c)
	if len(children) == 0 {
		return s
	}
	min_move := children[0]
	min_score := minimax(children[0], depth, c, c)
	for _, child := range children {
		v := minimax(child, depth, c, c)
		if v < min_score {
			min_move = child
			min_score = v
		}
	}
	return min_move
}

//func main() {
//	var s GameState
//	s.board = [width][height]Color{
//		{white, white, white},
//		{black, black, black},
//		{none, none, none},
//	}
//	s.kohs = map[Position]bool{}
//	s.print_board()
//	fmt.Println(liberties(&s, Position{0, 0}, map[Position]bool{}))
//}

func main() {
	var s GameState
	s.kohs = map[Position]bool{}
	var color_to_move Color = black
	for {
		s.print_board()
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
		next_move := make_turn(s, color_to_move)
		if next_move.board == s.board {
			break
		}
		color_to_move = color_to_move.invert()
		s = next_move
	}
	fmt.Println(score(s))
}
