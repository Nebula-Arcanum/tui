package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ZenNotes/tui/internal/vault"
	"github.com/ZenNotes/tui/internal/vim"
)

// calendarNow is the clock the calendar reads for "today", so tests can fix it.
var calendarNow = time.Now

// calSub is the calendar's layout: a month of day boxes, or a week or a day
// of columns.
type calSub int

const (
	calMonth calSub = iota
	calWeek
	calDay
)

// calFocus is the part of the calendar that takes the keys.
type calFocus int

const (
	calFocusGrid calFocus = iota
	calFocusItems
	calFocusStrip
)

// calMin holds the smallest body (width, height) each layout renders in.
// Below it the body only says the terminal is too small.
var calMin = map[calSub][2]int{
	calMonth: {72, 18},
	calWeek:  {64, 12},
	calDay:   {40, 10},
}

// calScheduledMark prefixes an entry placed by its scheduled date. It must
// stay one cell wide.
const calScheduledMark = "»"

// calRect is a region of the calendar body, in body cells.
type calRect struct{ x, y, w, h int }

func (r calRect) contains(x, y int) bool {
	return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h
}

// calDayBox is where one day sits in the grid.
type calDayBox struct {
	day  time.Time
	rect calRect
}

// calLayout is every region of the calendar body for one size. Rendering,
// clicks and menu placement all read the last one, so the geometry is
// written once.
type calLayout struct {
	w, h      int
	tooSmall  bool
	items     calRect
	strip     calRect
	boxes     []calDayBox
	itemStart int
	stripRows int
	// gutter is the width of the hour labels left of the week and day columns.
	gutter int
}

// taskCalendar is the Tasks tab's calendar: day entries, the items beside
// the grid, and the strip of tasks without a date.
type taskCalendar struct {
	sub    calSub
	focus  calFocus
	cursor int
	strip  int

	byDay   map[string][]calEntry
	undated []vault.Task
	layout  calLayout
}

func newTaskCalendar() *taskCalendar { return &taskCalendar{} }

// rebuild places the (already filtered) tasks and keeps the cursors inside
// the new lists.
func (c *taskCalendar) rebuild(tasks []vault.Task, day time.Time) {
	c.byDay, c.undated = bucketCalendarEntries(tasks)
	entries := c.dayEntries(day)
	if c.cursor >= len(entries) {
		c.cursor = max(0, len(entries)-1)
	}
	if c.focus == calFocusItems && len(entries) == 0 {
		c.focus = calFocusGrid
	}
}

func (c *taskCalendar) dayEntries(day time.Time) []calEntry {
	return c.byDay[day.Format("2006-01-02")]
}

// selected is the task the item panel or the strip has selected, if any.
func (c *taskCalendar) selected(day time.Time) *vault.Task {
	switch c.focus {
	case calFocusItems:
		entries := c.dayEntries(day)
		if c.cursor < len(entries) {
			t := entries[c.cursor].task
			return &t
		}
	case calFocusStrip:
		if c.strip < len(c.undated) {
			t := c.undated[c.strip]
			return &t
		}
	}
	return nil
}

// monthLayout computes the regions of the month view for a body of w x h.
func (c *taskCalendar) monthLayout(a *App, month time.Time, w, h int) calLayout {
	lay := calLayout{w: w, h: h}
	if m := calMin[calMonth]; w < m[0] || h < m[1] {
		lay.tooSmall = true
		return lay
	}
	panelW := clampInt(w*30/100, 26, 44)
	gw := w - panelW
	wnW := 0
	if a.prefs.CalendarShowWeekNumbers {
		wnW = 3
	}
	colW := (gw - wnW) / 7
	stripRows := clampInt(h/8, 2, 6)
	stripH := 1 + stripRows
	gridH := h - 2 - stripH
	first, weeks := monthGridStart(month, a.prefs.CalendarWeekStart != "sunday")
	rowH := gridH / weeks
	lay.items = calRect{x: gw, y: 0, w: panelW, h: h - stripH}
	lay.strip = calRect{x: 0, y: h - stripH, w: w, h: stripH}
	lay.stripRows = stripRows
	day := first
	for r := 0; r < weeks; r++ {
		for col := 0; col < 7; col++ {
			lay.boxes = append(lay.boxes, calDayBox{day: day, rect: calRect{x: wnW + col*colW, y: 2 + r*rowH, w: colW, h: rowH}})
			day = day.AddDate(0, 0, 1)
		}
	}
	return lay
}

// hourLabelWidth is the width of the widest hour label for the time format.
func hourLabelWidth(a *App) int {
	if a.prefs.TimeFormat == "12h" {
		return 5
	}
	return 2
}

// hourLabel is the label of hour row h (0-23).
func hourLabel(a *App, h int) string {
	if a.prefs.TimeFormat == "12h" {
		n := h % 12
		if n == 0 {
			n = 12
		}
		suffix := "AM"
		if h >= 12 {
			suffix = "PM"
		}
		return fmt.Sprintf("%2d %s", n, suffix)
	}
	return fmt.Sprintf("%02d", h)
}

// columnDays is the days a week or day layout shows as columns, in order.
func (c *taskCalendar) columnDays(a *App, day time.Time) []time.Time {
	if c.sub == calDay {
		return []time.Time{day}
	}
	offset := int(day.Weekday())
	if a.prefs.CalendarWeekStart != "sunday" {
		offset = (offset + 6) % 7
	}
	first := time.Date(day.Year(), day.Month(), day.Day()-offset, 0, 0, 0, 0, time.Local)
	days := make([]time.Time, 7)
	for i := range days {
		days[i] = first.AddDate(0, 0, i)
	}
	return days
}

// columnLayout computes the regions of the week or day view for a body of
// w x h. The hour list sits under the header rows with one column per day
// beside it, and the strip is pinned to the bottom of the body.
func (c *taskCalendar) columnLayout(a *App, day time.Time, w, h int) calLayout {
	lay := calLayout{w: w, h: h}
	if m := calMin[c.sub]; w < m[0] || h < m[1] {
		lay.tooSmall = true
		return lay
	}
	days := c.columnDays(a, day)
	headRows := 1
	if c.sub == calWeek {
		headRows = 2
	}
	stripRows := clampInt(h/8, 2, 6)
	stripH := 1 + stripRows
	lay.stripRows = stripRows
	lay.strip = calRect{x: 0, y: h - stripH, w: w, h: stripH}
	lay.gutter = hourLabelWidth(a) + 2
	hourRows := clampInt(h-stripH-headRows, 0, 24)
	colW := (w - lay.gutter) / len(days)
	for i, d := range days {
		lay.boxes = append(lay.boxes, calDayBox{day: d, rect: calRect{x: lay.gutter + i*colW, y: headRows, w: colW, h: hourRows}})
	}
	return lay
}

// renderColumns draws the week or day view: a header, the hour list with the
// day columns beside it, and the strip. Entries are listed from the top of
// their column, not placed by time of day.
func (c *taskCalendar) renderColumns(a *App, day time.Time, lay calLayout, w, h int, focused bool) string {
	th := a.theme
	rows := make([]string, 0, h)
	blank := strings.Repeat(" ", lay.gutter)
	first, last := lay.boxes[0].day, lay.boxes[len(lay.boxes)-1].day
	if c.sub == calDay {
		rows = append(rows, padRight(" "+th.Title.Render(day.Format("Monday, January 2, 2006")), w))
	} else {
		title := first.Format("January 2") + " – " + last.Format("January 2, 2006")
		rows = append(rows, padRight(" "+th.Title.Render(title), w))
		head := blank
		for _, b := range lay.boxes {
			cell := padRight(" "+b.day.Format("Mon")[:2]+b.day.Format(" 2"), b.rect.w)
			if sameDay(b.day, day) {
				cell = th.Selected.Render(cell)
			} else {
				cell = th.Dim.Render(cell)
			}
			head += cell
		}
		rows = append(rows, padRight(head, w))
	}
	hourRows := lay.boxes[0].rect.h
	for i := 0; i < hourRows; i++ {
		var sb strings.Builder
		sb.WriteString(th.Muted.Render(padRight(hourLabel(a, i), lay.gutter-2)) + th.Dim.Render(" │"))
		for _, b := range lay.boxes {
			sb.WriteString(c.columnLine(a, b, i, focused && sameDay(b.day, day)))
		}
		rows = append(rows, padRight(sb.String(), w))
	}
	for len(rows) < lay.strip.y {
		rows = append(rows, strings.Repeat(" ", w))
	}
	rows = append(rows, c.stripLines(a, lay, w, focused)...)
	return fitBlock(strings.Join(rows, "\n"), w, h)
}

// columnLine is line li of one day column: entries from the top, with
// "+N more" on the last line when they do not all fit.
func (c *taskCalendar) columnLine(a *App, b calDayBox, li int, focusedDay bool) string {
	th := a.theme
	w := b.rect.w
	entries := c.dayEntries(b.day)
	shown := len(entries)
	if shown > b.rect.h {
		shown = b.rect.h - 1
	}
	switch {
	case li < shown:
		e := entries[li]
		lead := " "
		if e.scheduled {
			lead = calScheduledMark
		}
		text := padRight(lead+truncateCells(e.task.Content, w-2), w)
		if focusedDay && c.focus == calFocusItems && li == c.cursor {
			return th.SelectedFocus.Render(text)
		}
		return text
	case li == shown && shown < len(entries):
		return th.Dim.Render(padRight(fmt.Sprintf(" +%d more", len(entries)-shown), w))
	}
	return strings.Repeat(" ", w)
}

// monthGridStart is the first day shown for a month and how many week rows
// cover it.
func monthGridStart(month time.Time, mondayFirst bool) (time.Time, int) {
	first := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.Local)
	offset := int(first.Weekday())
	if mondayFirst {
		offset = (offset + 6) % 7
	}
	days := first.AddDate(0, 1, -1).Day()
	return first.AddDate(0, 0, -offset), (offset + days + 6) / 7
}

// render draws the calendar into a w x h body.
func (c *taskCalendar) render(a *App, month, day time.Time, w, h int, focused bool) string {
	var lay calLayout
	if c.sub == calMonth {
		lay = c.monthLayout(a, month, w, h)
	} else {
		lay = c.columnLayout(a, day, w, h)
	}
	c.layout = lay
	th := a.theme
	if lay.tooSmall {
		m := calMin[c.sub]
		lines := make([]string, max(h, 0))
		if len(lines) > 0 {
			msg := fmt.Sprintf("Terminal too small: needs %d×%d", m[0], m[1])
			lines[len(lines)/2] = centerLine(th.Muted.Render(msg), w)
		}
		return fitBlock(strings.Join(lines, "\n"), w, h)
	}

	if c.sub != calMonth {
		return c.renderColumns(a, day, lay, w, h, focused)
	}
	weeks := len(lay.boxes) / 7
	gw := lay.items.x
	wnW := lay.boxes[0].rect.x
	colW := lay.boxes[0].rect.w
	rowH := lay.boxes[0].rect.h
	left := make([]string, lay.items.h)
	left[0] = padRight(" "+th.Title.Render(month.Format("January 2006")), gw)
	names := []string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"}
	if a.prefs.CalendarWeekStart == "sunday" {
		names = []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}
	}
	head := strings.Repeat(" ", wnW)
	for _, n := range names {
		head += padRight(" "+th.Dim.Render(n), colW)
	}
	left[1] = padRight(head, gw)
	today := calendarNow()
	for r := 0; r < weeks; r++ {
		for li := 0; li < rowH; li++ {
			var sb strings.Builder
			if wnW > 0 {
				if li == 0 {
					anchor := lay.boxes[r*7].day
					if a.prefs.CalendarWeekStart == "sunday" {
						anchor = anchor.AddDate(0, 0, 1)
					}
					_, wk := anchor.ISOWeek()
					sb.WriteString(th.Muted.Render(fmt.Sprintf("%2d ", wk)))
				} else {
					sb.WriteString(strings.Repeat(" ", wnW))
				}
			}
			for col := 0; col < 7; col++ {
				box := lay.boxes[r*7+col]
				sb.WriteString(c.boxLine(a, box, li, sameDay(box.day, day), sameDay(box.day, today), box.day.Month() != month.Month(), focused))
			}
			left[2+r*rowH+li] = padRight(sb.String(), gw)
		}
	}
	for i := range left {
		if left[i] == "" {
			left[i] = strings.Repeat(" ", gw)
		}
	}

	panel := c.itemPanel(a, day, lay, focused)
	rows := make([]string, 0, h)
	for i := 0; i < lay.items.h; i++ {
		rows = append(rows, left[i]+panel[i])
	}
	rows = append(rows, c.stripLines(a, lay, w, focused)...)
	return fitBlock(strings.Join(rows, "\n"), w, h)
}

// boxLine is line li of one day box: the date number on top, then one line
// per entry, with "+N more" on the last line when they do not all fit.
func (c *taskCalendar) boxLine(a *App, box calDayBox, li int, selected, today, outside, focused bool) string {
	th := a.theme
	w := box.rect.w
	text := ""
	style := th.Base
	if outside {
		style = th.Muted
	}
	if li == 0 {
		text = fmt.Sprintf(" %d", box.day.Day())
		if today && !selected {
			style = th.Base.Foreground(th.Accent).Bold(true)
		}
	} else {
		entries := c.dayEntries(box.day)
		avail := box.rect.h - 1
		shown := len(entries)
		if shown > avail {
			shown = avail - 1
		}
		switch {
		case li-1 < shown:
			e := entries[li-1]
			lead := " "
			if e.scheduled {
				lead = calScheduledMark
			}
			text = lead + truncateCells(e.task.Content, w-1)
		case li-1 == shown && shown < len(entries):
			text = " " + fmt.Sprintf("+%d more", len(entries)-shown)
			if !selected {
				style = th.Dim
			}
		}
	}
	if selected {
		if focused && c.focus == calFocusGrid {
			style = th.SelectedFocus
		} else {
			style = th.Selected
		}
	}
	return style.Render(padRight(text, w))
}

// itemPanel lists every entry of the selected day beside the grid.
func (c *taskCalendar) itemPanel(a *App, day time.Time, lay calLayout, focused bool) []string {
	th := a.theme
	pw := lay.items.w
	sep := th.Dim.Render("│")
	entries := c.dayEntries(day)
	out := make([]string, lay.items.h)
	out[0] = padRight(sep+" "+th.Bold.Render(day.Format("Monday, January 2"))+th.Muted.Render(fmt.Sprintf(" (%d)", len(entries))), pw)
	rows := lay.items.h - 1
	start := 0
	if c.cursor >= rows {
		start = c.cursor - rows + 1
	}
	c.layout.itemStart = start
	for i := 1; i < lay.items.h; i++ {
		line := sep
		idx := start + i - 1
		switch {
		case idx < len(entries):
			e := entries[idx]
			lead := " "
			if e.scheduled {
				lead = calScheduledMark
			}
			body := " " + lead + " " + truncateCells(e.task.Content, pw-4)
			if focused && c.focus == calFocusItems && idx == c.cursor {
				body = th.SelectedFocus.Render(padRight(body, pw-1))
			}
			line += body
		case len(entries) == 0 && i == 1:
			line += " " + th.Muted.Render(truncateCells("Nothing here. n adds a task.", pw-2))
		}
		out[i] = padRight(line, pw)
	}
	return out
}

// stripLines draws the strip of tasks without a date under the grid.
func (c *taskCalendar) stripLines(a *App, lay calLayout, w int, focused bool) []string {
	th := a.theme
	out := make([]string, lay.strip.h)
	out[0] = padRight(" "+th.Bold.Render("No date")+th.Muted.Render(fmt.Sprintf(" (%d)", len(c.undated))), w)
	rows := lay.stripRows
	start := 0
	if c.focus == calFocusStrip && c.strip >= rows {
		start = c.strip - rows + 1
	}
	for i := 0; i < rows; i++ {
		idx := start + i
		line := ""
		if idx < len(c.undated) {
			line = " " + truncateCells(c.undated[idx].Content, w-2)
			if focused && c.focus == calFocusStrip && idx == c.strip {
				line = th.SelectedFocus.Render(padRight(line, w))
			}
		}
		out[1+i] = padRight(line, w)
	}
	return out
}

// click handles a mouse press at body cell (x, y). It reports whether the

func (c *taskCalendar) click(a *App, v *tasksView, x, y int, right, double bool) {
	lay := c.layout
	if lay.tooSmall {
		return
	}
	for _, b := range lay.boxes {
		if b.rect.contains(x, y) {
			v.selectDay(a, b.day)
			c.focus = calFocusGrid
			if double {
				v.calendarKey(a, vim.KeyEnter)
			}
			return
		}
	}
	if lay.items.contains(x, y) && y > lay.items.y {
		idx := lay.itemStart + y - lay.items.y - 1
		entries := c.dayEntries(v.day)
		if idx < len(entries) {
			c.focus = calFocusItems
			c.cursor = idx
			t := entries[idx].task
			switch {
			case right:
				a.taskMenu(t)
			case double:
				a.openTaskSource(t)
			}
		}
	}
}

// selectionCell is the body cell menus open next to.
func (c *taskCalendar) selectionCell(day time.Time) (int, int) {
	lay := c.layout
	if c.focus == calFocusItems {
		return lay.items.x + 3, lay.items.y + 1 + c.cursor - lay.itemStart
	}
	for _, b := range lay.boxes {
		if sameDay(b.day, day) {
			return b.rect.x + 1, b.rect.y
		}
	}
	return 0, 0
}
