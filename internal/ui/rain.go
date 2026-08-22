package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Rain is presentation only (the timer ruling): the tick in model.go
// advances Model.phase and nothing else — World never sees weather,
// turns.md still governs game state. rainField is a pure function of
// its arguments, so any single frame is testable without the timer.

const (
	rainColsPerDrop = 10 // gentle: about one drop per this many columns
	rainTrailLen    = 2  // fading cells trailing above a drop's head
	rainMinSpeed    = 2  // fastest drop: one row per this many frames
	rainMaxSpeed    = 4  // slowest drop: one row per this many frames
)

// rainField draws height rows of drops falling through the room
// panel's dead space. Each drop's column, fall offset, and speed come
// from a hash of the room ID — every room gets its own rain. Drops
// fall at staggered per-drop speeds (a row per 2–4 frames, offset
// from each other) so the field never steps in lockstep: at a fast
// frame rate that reads as smooth motion, not a slideshow. A drop
// fades upward: bright head, grey mid, faint tail. level picks a
// rung of the opacity ladder (styles.go); 0 renders nothing.
func rainField(roomID string, phase, width, height, level int) []string {
	return themedRainField(roomID, phase, width, height, level,
		defaultOverworldStyles().rainShades)
}

func themedRainField(roomID string, phase, width, height, level int, shades [5][3]lipgloss.Style) []string {
	if width < 1 || height < 1 || level < 1 {
		return nil
	}
	if level > maxRainLevel {
		level = maxRainLevel
	}
	shade := shades[level]
	// Brightness level per cell: 0 empty, 1 tail, 2 mid, 3 head.
	grid := make([][]int, height)
	for r := range grid {
		grid[r] = make([]int, width)
	}
	seed := uint32(2166136261)
	for _, r := range roomID {
		seed = (seed ^ uint32(r)) * 16777619
	}
	cycle := height + rainTrailLen
	for d := 0; d < width/rainColsPerDrop; d++ {
		seed = seed*1664525 + 1013904223
		col := int(seed>>16) % width
		off := int(seed>>4) % cycle
		speed := rainMinSpeed + int(seed>>2)%(rainMaxSpeed-rainMinSpeed+1)
		// (phase+off)/speed staggers *when* each drop steps, so drops
		// with the same speed still don't move on the same frame.
		head := (off + (phase+off)/speed) % cycle
		for k := 0; k <= rainTrailLen; k++ {
			if r := head - k; r >= 0 && r < height {
				grid[r][col] = 3 - k
			}
		}
	}
	rows := make([]string, height)
	var b strings.Builder
	for r := 0; r < height; r++ {
		b.Reset()
		for c := 0; c < width; c++ {
			// All periods — the fade alone carries the motion.
			if lvl := grid[r][c]; lvl == 0 {
				b.WriteByte(' ')
			} else {
				b.WriteString(shade[3-lvl].Render("·"))
			}
		}
		rows[r] = strings.TrimRight(b.String(), " ")
	}
	return rows
}
