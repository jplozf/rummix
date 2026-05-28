package main

// ----------------------------------------------------------------------------
// IMPORTS
// ----------------------------------------------------------------------------
import (
	"fmt"
	"image/color"
	"regexp"
	"rummix/grummi"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ----------------------------------------------------------------------------
// TYPES
// ----------------------------------------------------------------------------
type HoverButton struct {
	widget.Button
	tooltip string
}

func NewHoverButton(tooltip string, icon fyne.Resource, tapped func()) *HoverButton {
	hb := &HoverButton{tooltip: tooltip}
	hb.Icon = icon
	hb.OnTapped = tapped
	hb.ExtendBaseWidget(hb)
	return hb
}

func (h *HoverButton) MouseIn(e *desktop.MouseEvent) {
	statusMsg.SetText(stripANSI(h.tooltip))
}

func (h *HoverButton) MouseOut() {
	statusMsg.SetText("")
}

func (h *HoverButton) MouseMoved(e *desktop.MouseEvent) {}

// ----------------------------------------------------------------------------
// CONSTANTS
// ----------------------------------------------------------------------------

// ----------------------------------------------------------------------------
// VARS
// ----------------------------------------------------------------------------
var (
	boardCellSize     fyne.Size
	rackCellSize      fyne.Size
	isPaused          bool
	humanTimerElapsed time.Duration
	confirmBtn        *HoverButton
	sortBtn           *HoverButton
	drawBtn           *HoverButton
	rollbackBtn       *HoverButton
	pauseBtn          *HoverButton
	turnLimitMinutes  int
)
var boardSlots []fyne.CanvasObject
var rackSlots []fyne.CanvasObject
var myWindow fyne.Window
var gameTable *fyne.Container
var playerRack *fyne.Container
var overlay *fyne.Container
var statusMsg *widget.Label
var statusNames []*canvas.Text
var statusLabel *widget.Label
var statusDrawLabel *widget.Label
var statusTimerLabel *canvas.Text
var statusLimitLabel *canvas.Text
var statusOpeningPointsLabel *canvas.Text
var timerStop chan bool // Channel to stop the human player's timer

// New variables for the statistics table
var statsPlayerLabels []*widget.Label
var statsWinsLabels []*widget.Label
var statsGamesLabels []*widget.Label
var statsPointsLabels []*widget.Label
var statusTiles []*widget.Label
var gameState grummi.GameState
var myApp fyne.App
var isGameLoading bool
var isTurnProcessing bool
var background *canvas.Rectangle
var aiLogEntry *widget.Label
var humanPool []grummi.Tile // Added for human player's temporary tiles
var aiLogScroll *container.Scroll

var rummixLogo *canvas.Image
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// playerStats is a helper struct to hold player statistics for sorting.
type playerStats struct {
	Name  string
	Wins  int
	Games int
	Score int
}

// stripANSI removes ANSI escape sequences (like colors) from a string.
func stripANSI(str string) string {
	str = ansiRegex.ReplaceAllString(str, "")

	// Replace emojis with text equivalents to avoid rendering issues on some platforms
	replacer := strings.NewReplacer(
		"😁", "",
		"🔴", ":RED",
		"🔵", ":BLUE",
		"🟢", ":GREEN",
		"🟠", ":ORANGE",
		"🤖", "",
		"🎲", "",
		"🧩", "",
		"📥", "",
		"🖐️", "",
		"✨", "",
		"⭐", "",
		"🎉", "",
		"🏆", "",
		"📊", "",
		"👤", "",
		"💻", "",
		"🃏", "",
		"✔", "",
		"✖", "",
		"►", "",
	)
	/*
		replacer := strings.NewReplacer(
			"😁", "[Joker]",
			"🔴", "(R)",
			"🔵", "(B)",
			"🟢", "(G)",
			"🟠", "(O)",
			"🤖", "[AI]",
			"🎲", "Roll:",
			"🧩", "Table",
			"📥", "Pool",
			"🖐️", "Hand",
			"✨", "*",
			"⭐", "!",
			"🎉", "!",
			"🏆", "WIN",
			"📊", "Stats",
			"👤", "P",
			"🃏", "Deck",
			"✔", "OK",
			"✖", "X",
			"►", ">",
		)
	*/
	return replacer.Replace(str)
}

// uiLogger implements the grummi.Logger interface to direct logs to the UI.
type uiLogger struct{}

// Log sends messages to the UI's status bar and log panel.
func (l *uiLogger) Log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fyne.Do(func() {
		SetStatus(msg)
		refreshTable()
		refreshRack()
	})
	time.Sleep(600 * time.Millisecond) // Short delay to let the user see the move
}

// ----------------------------------------------------------------------------
// main()
// ----------------------------------------------------------------------------
func main() {

	myApp = app.NewWithID(APP_ID)
	myApp.SetIcon(resourceRummixPng)
	myApp.Settings().SetTheme(&compactTheme{Theme: theme.DefaultTheme()})

	myWindow = myApp.NewWindow(APP_NAME)
	setMenu()
	myWindow.SetCloseIntercept(func() {
		confirmExit()
	})

	InitAudio()

	boardCellSize = fyne.NewSize(40, 52)
	rackCellSize = fyne.NewSize(30, 39)

	// Initialize the overlay and background containers early. refreshRack and refreshTable
	// are called during startup and create DragTiles that require these
	// references to be non-nil for rendering phantom tiles during drag operations.
	overlay = container.NewWithoutLayout()
	background = canvas.NewRectangle(color.Transparent)

	gameState = grummi.InitializeGame(2, &uiLogger{})
	humanPool = []grummi.Tile{} // Initialize the pool for the human player

	// The main game table
	gameTable = container.New(layout.NewGridLayoutWithColumns(28))
	for i := range 224 {
		cell := createCell(boardCellSize, true)
		gameTable.Add(cell)
		registerCell(cell, gameTable, i)
	}
	tableWidth := boardCellSize.Width * 28
	tableHeight := boardCellSize.Height * 8
	totalTableSize := fyne.NewSize(tableWidth, tableHeight)

	fixedTable := container.NewGridWrap(totalTableSize, gameTable)

	// The bottom area
	aiLogEntry = widget.NewLabel("")
	aiLogEntry.Wrapping = fyne.TextWrapWord
	aiLogScroll = container.NewScroll(aiLogEntry)
	aiLogScroll.SetMinSize(fyne.NewSize(250, 156)) // Match the player rack height

	playerRack = container.New(layout.NewGridLayoutWithColumns(20))
	for i := range 80 { // 4 rows * 20 columns
		cell := createCell(rackCellSize, false)
		playerRack.Add(cell)
		registerCell(cell, playerRack, i)
	}
	rackWidth := rackCellSize.Width * 20
	rackHeight := rackCellSize.Height * 4
	totalRackSize := fyne.NewSize(rackWidth, rackHeight)

	fixedRack := container.NewGridWrap(totalRackSize, playerRack)

	confirmBtn = NewHoverButton(grummi.T("btn_validate"), theme.ConfirmIcon(), func() {
		if isTurnProcessing {
			return
		}
		if syncUItoGameState() {
			PlayOkSound() // Play OK sound for valid move
			stopHumanTimer()
			humanTimerElapsed = 0
			gameState.ConsecutivePasses = 0 // Valid move, reset passes
			gameState.TurnNumber++          // Increment turn number after a valid human move
			updateStatusTiles()             // Refresh status display
			if checkGameEnd() {
				return // Game ended
			}
			gameState.CurrentPlayerID = (gameState.CurrentPlayerID + 1) % len(gameState.Players)
			playNextTurn()
		}
	})
	sortBtn = NewHoverButton(grummi.T("btn_arrange"), theme.ViewRefreshIcon(), func() {
		grummi.SortTiles(gameState.Players[0].Hand)
		refreshRack()
		SetStatus(grummi.T("status_sorting"))
	})
	drawBtn = NewHoverButton(grummi.T("btn_draw"), theme.ContentAddIcon(), func() {
		performHumanDraw()
	})
	rollbackBtn = NewHoverButton(grummi.T("btn_rollback"), theme.CancelIcon(), func() { // Rollback button
		if isTurnProcessing {
			return
		}
		refreshTable()
		refreshRack()
		PlayNogoodSound() // Play Nogood sound for invalid move
		SetStatus(grummi.T("move_cancelled"))
	})
	pauseBtn = NewHoverButton(grummi.T("btn_pause"), theme.MediaPauseIcon(), func() {
		togglePause()
	})

	buttons := container.NewVBox(
		confirmBtn,
		sortBtn,
		drawBtn,
		rollbackBtn,
		pauseBtn,
	)

	gapBetweenRackAndButtons := canvas.NewRectangle(color.Transparent)
	gapBetweenRackAndButtons.SetMinSize(fyne.NewSize(10, 0))

	// Logo alignment logic:
	// Width 140 to match the statusArea, height 156 to match the rack height (4 rows * 39px).
	rummixLogo = canvas.NewImageFromResource(resourceRummixPng)
	rummixLogo.FillMode = canvas.ImageFillContain
	rummixLogo.SetMinSize(fyne.NewSize(100, 100))

	logoSpacer := canvas.NewRectangle(color.Transparent)
	logoSpacer.SetMinSize(fyne.NewSize(140, 156))
	logoRightContainer := container.NewStack(logoSpacer, container.NewCenter(rummixLogo))

	rackAndButtonsContainer := container.NewHBox(
		gapBetweenRackAndButtons,
		fixedRack,
		gapBetweenRackAndButtons,
		gapBetweenRackAndButtons,
		container.NewPadded(buttons),
		gapBetweenRackAndButtons,
		logoRightContainer)

	statusMsg = widget.NewLabel("")

	// The status area on the right
	statusLabel = widget.NewLabelWithStyle("1", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
	statusDrawLabel = widget.NewLabelWithStyle("0", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
	// Initialize the timer here, as it's part of the status area
	statusTimerLabel = canvas.NewText("00:00", theme.ForegroundColor())
	statusTimerLabel.Alignment = fyne.TextAlignCenter
	statusTimerLabel.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}
	statusTimerLabel.TextSize = 24

	statusLimitLabel = canvas.NewText("", theme.ForegroundColor())
	statusLimitLabel.Alignment = fyne.TextAlignCenter
	statusLimitLabel.TextSize = 12

	statusOpeningPointsLabel = canvas.NewText("", theme.ForegroundColor())
	statusOpeningPointsLabel.Alignment = fyne.TextAlignCenter
	statusOpeningPointsLabel.TextSize = 12

	statusNames = make([]*canvas.Text, 4)
	for i := range 4 {
		statusNames[i] = canvas.NewText("-", theme.ForegroundColor())
		statusNames[i].TextStyle = fyne.TextStyle{Bold: true}
	}

	// Initialize new labels for the statistics table
	statsPlayerLabels = make([]*widget.Label, 4)
	statsWinsLabels = make([]*widget.Label, 4)
	statsGamesLabels = make([]*widget.Label, 4)
	statsPointsLabels = make([]*widget.Label, 4)
	for i := range 4 {
		statsPlayerLabels[i] = widget.NewLabelWithStyle("-", fyne.TextAlignLeading, fyne.TextStyle{})
		statsWinsLabels[i] = widget.NewLabelWithStyle("-", fyne.TextAlignTrailing, fyne.TextStyle{})
		statsGamesLabels[i] = widget.NewLabelWithStyle("-", fyne.TextAlignTrailing, fyne.TextStyle{})
		statsPointsLabels[i] = widget.NewLabelWithStyle("-", fyne.TextAlignTrailing, fyne.TextStyle{})
	}

	statusTiles = make([]*widget.Label, 4)
	for i := range 4 {
		statusTiles[i] = widget.NewLabelWithStyle("-", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
	}

	// shrink wraps an object in a fixed-height container to force tighter row spacing.
	shrink := func(obj fyne.CanvasObject, width float32) fyne.CanvasObject {
		return container.NewGridWrap(fyne.NewSize(width, 20), obj)
	}

	gameInfo := container.New(layout.NewFormLayout(),
		shrink(widget.NewLabel(grummi.T("label_turn")), 100), shrink(statusLabel, 40),
		shrink(widget.NewLabel(grummi.T("label_draw_pile")), 100), shrink(statusDrawLabel, 40),
	)

	handsGrid := container.New(layout.NewFormLayout())
	for i := 0; i < 4; i++ {
		handsGrid.Add(shrink(statusNames[i], 100))
		handsGrid.Add(shrink(statusTiles[i], 40))
	}

	// New statsGrid using GridLayoutWithColumns for a table-like display
	statsGrid := container.New(layout.NewGridLayoutWithColumns(4))
	// Add headers for the stats table
	statsGrid.Add(shrink(widget.NewLabelWithStyle(grummi.T("column_player"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), 60))
	statsGrid.Add(shrink(widget.NewLabelWithStyle(grummi.T("column_wins"), fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}), 25))
	statsGrid.Add(shrink(widget.NewLabelWithStyle(grummi.T("column_games"), fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}), 25))
	statsGrid.Add(shrink(widget.NewLabelWithStyle(grummi.T("column_score"), fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}), 30))

	// Add player stat labels to the grid
	for i := 0; i < 4; i++ {
		statsGrid.Add(shrink(statsPlayerLabels[i], 60))
		statsGrid.Add(shrink(statsWinsLabels[i], 25))
		statsGrid.Add(shrink(statsGamesLabels[i], 25))
		statsGrid.Add(shrink(statsPointsLabels[i], 30))
	}

	// Dissociate components using sections with headers and separators
	statusDetails := container.NewVBox( // This container holds the different sections
		gameInfo,
		widget.NewSeparator(),
		container.NewCenter(shrink(widget.NewLabelWithStyle(grummi.T("label_tiles_section"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true, Italic: true}), 140)),
		handsGrid,
		layout.NewSpacer(),
		widget.NewSeparator(),
		container.NewCenter(shrink(widget.NewLabelWithStyle(grummi.T("label_stats_section"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true, Italic: true}), 140)),
		statsGrid,
	)

	statusTitle := widget.NewLabelWithStyle(grummi.T("section_status"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	statusArea := container.NewBorder(
		container.NewVBox(
			container.NewCenter(shrink(statusTitle, 140)),
			statusDetails,
		),
		container.NewVBox(
			container.NewCenter(statusLimitLabel),
			container.NewCenter(statusOpeningPointsLabel),
			container.NewCenter(statusTimerLabel),
		), // Pushes the limit and timer to the very bottom
		nil, nil,
		layout.NewSpacer(), // Fills the middle space
	)

	refreshRack()
	readPreferences()
	refreshTable() // Display the initial (empty) table

	// Define a 3-column layout to ensure vertical alignment of side elements
	// Left Column: Status top, AI Log bottom
	leftTopSpacer := canvas.NewRectangle(color.Transparent)
	leftTopSpacer.SetMinSize(fyne.NewSize(250, 416)) // Match table height (8 rows * 52px)
	vGapLeft := canvas.NewRectangle(color.Transparent)
	vGapLeft.SetMinSize(fyne.NewSize(0, 20)) // 20px space
	leftColumn := container.NewBorder(container.NewVBox(container.NewStack(leftTopSpacer, statusArea), vGapLeft), nil, nil, nil, aiLogScroll)

	// Center Column: Table top, Rack/Buttons bottom
	vGapCenter := canvas.NewRectangle(color.Transparent)
	vGapCenter.SetMinSize(fyne.NewSize(0, 20))
	centerColumn := container.NewVBox(fixedTable, vGapCenter, rackAndButtonsContainer)

	// Assemble rows horizontally and center the whole board
	gameBoard := container.NewHBox(gapBetweenRackAndButtons, leftColumn, gapBetweenRackAndButtons, centerColumn, gapBetweenRackAndButtons)
	mainInterface := container.NewBorder(nil, statusMsg, nil, nil, container.NewCenter(gameBoard))

	windowContent := container.NewStack(background, mainInterface)
	finalStack := container.NewStack(windowContent, overlay)

	// Let's show the window and run the app
	updateBackgroundColor()
	myWindow.SetContent(finalStack)
	myWindow.Resize(fyne.NewSize(1400, 700))

	SetStatus(grummi.T("status_welcome"))
	showNewGameDialog(myWindow, onNewGame)
	// The game loop will start after the dialog is closed and onNewGame is called.
	myWindow.ShowAndRun()
}

// ----------------------------------------------------------------------------
// updateBackgroundColor()
// ----------------------------------------------------------------------------
func updateBackgroundColor() {
	if myApp.Preferences().StringWithFallback("AppTheme", "light") == "dark" {
		background.FillColor = ColorBackgroundGlobalDark
	} else {
		background.FillColor = ColorBackgroundGlobalLight
	}
	if statusTimerLabel != nil {
		statusTimerLabel.Color = theme.ForegroundColor()
		statusTimerLabel.Refresh()
	}
	if statusLimitLabel != nil {
		statusLimitLabel.Color = theme.ForegroundColor()
		statusLimitLabel.Refresh()
	}
	if statusOpeningPointsLabel != nil {
		statusOpeningPointsLabel.Color = theme.ForegroundColor()
		statusOpeningPointsLabel.Refresh()
	}
	// The new stats labels will automatically pick up the theme's text color.
	// statusNames are still used for the hands grid, and their color is set in updateStatusTiles.
	background.Refresh()
}

// ----------------------------------------------------------------------------
// refreshRack()
// ----------------------------------------------------------------------------
// refreshRack clears the player's rack and repopulates it with tiles from the current game state.
func refreshRack() {
	// Clear the current rack display
	for i := 0; i < len(playerRack.Objects); i++ {
		wrapper := playerRack.Objects[i].(*fyne.Container)
		cellStack := wrapper.Objects[0].(*fyne.Container)
		if len(cellStack.Objects) > 1 {
			delete(cellMap, cellStack.Objects[1]) // Remove the old tile from the cellMap
			cellStack.Objects = cellStack.Objects[:1]
			cellStack.Refresh()
		}
	}

	// Add tiles from the player's hand using a more compact 4-row layout
	for _, tile := range gameState.Players[0].Hand {
		if tile.Value != 0 {
			// Place regular tiles by color and value
			row := int(tile.Color)
			col := tile.Value - 1
			idx := row*20 + col

			if isCellOccupied(playerRack, idx) {
				// Handle duplicates: place in the 'overflow' area (cols 13-19)
				for c := 13; c < 20; c++ {
					altIdx := row*20 + c
					if !isCellOccupied(playerRack, altIdx) {
						setTileAt(playerRack, altIdx, tile.Value, tile.Color)
						break
					}
				}
			} else {
				setTileAt(playerRack, idx, tile.Value, tile.Color)
			}
		} else {
			// Joker: place in the first available overflow slot (cols 13-19) in any row
			placed := false
			for r := 0; r < 4 && !placed; r++ {
				for c := 13; c < 20; c++ {
					idx := r*20 + c
					if !isCellOccupied(playerRack, idx) {
						setTileAt(playerRack, idx, tile.Value, tile.Color)
						placed = true
						break
					}
				}
			}
		}
	}

	updateStatusTiles()
	playerRack.Refresh()
}

// ----------------------------------------------------------------------------
// refreshTable()
// ----------------------------------------------------------------------------
// refreshTable clears the game table and repopulates it with combinations from the current game state.
func refreshTable() {
	// Identify what was already on the table to avoid re-animating existing tiles
	existingTiles := make(map[string]int)
	for i := 0; i < 224; i++ {
		if t := getTileAtCell(gameTable, i); t != nil {
			key := fmt.Sprintf("%d-%d", t.Value, t.Color)
			existingTiles[key]++
		}
	}

	// Clear the current table display
	for i := 0; i < len(gameTable.Objects); i++ {
		wrapper := gameTable.Objects[i].(*fyne.Container)
		cellStack := wrapper.Objects[0].(*fyne.Container)
		if len(cellStack.Objects) > 1 {
			// Remove the old DragTile from the cellMap if it exists
			if dt, ok := cellStack.Objects[1].(*DragTile); ok {
				delete(cellMap, dt)
			}
			cellStack.Objects = cellStack.Objects[:1] // Keep only the background
			cellStack.Refresh()
		}
	}

	const maxCols = 28 // Number of columns in gameTable
	groupCount := 0

	// placeTile is a helper to set a tile and optionally animate it if it's new to the table
	placeTile := func(idx int, tile grummi.Tile) {
		if idx < 0 || idx >= 224 {
			return
		}
		setTileAt(gameTable, idx, tile.Value, tile.Color)

		key := fmt.Sprintf("%d-%d", tile.Value, tile.Color)
		if count := existingTiles[key]; count > 0 {
			existingTiles[key]--
		} else {
			// New tile! Animate it in from the player's side
			wrapper := gameTable.Objects[idx].(*fyne.Container)
			cellStack := wrapper.Objects[0].(*fyne.Container)
			if len(cellStack.Objects) > 1 {
				tileVisual := cellStack.Objects[1]
				tileVisual.Hide()
				animateTileIn(tile, cellStack, gameState.CurrentPlayerID, idx)
			}
		}
	}

	for _, combo := range gameState.Table {
		if grummi.IsValidRun(combo) {
			// Logic for Runs: Fixed positions by color and value on the left (cols 0-12)
			c := getRunColor(combo)
			// We assign 2 rows per color (8 rows total for 4 colors)
			row := int(c) * 2

			// Check if the first designated row for this color is already occupied
			// at the required positions to handle duplicate runs of the same color.
			isRow0Occupied := false
			for i := range combo {
				val := getTileValueInRun(combo, i)
				idx := row*maxCols + (val - 1)
				if idx >= 0 && idx < 224 && isCellOccupied(gameTable, idx) {
					isRow0Occupied = true
					break
				}
			}
			if isRow0Occupied {
				row++ // Use the second row for this color
			}

			for i, tile := range combo {
				val := getTileValueInRun(combo, i)
				if val >= 1 && val <= 13 {
					placeTile(row*maxCols+(val-1), tile)
				}
			}
		} else {
			// Logic for Groups: Place them on the right side (columns 14-27)
			// We fit 3 groups per row with 1-cell gaps: cols 14-17, 19-22, and 24-27.
			r := groupCount / 3
			cOffset := 14
			if groupCount%3 == 1 {
				cOffset = 19
			} else if groupCount%3 == 2 {
				cOffset = 24
			}

			if r < 8 {
				for i, tile := range combo {
					if i < 4 {
						placeTile(r*maxCols+cOffset+i, tile)
					}
				}
				groupCount++
			}
		}
	}
	gameTable.Refresh()
}

// ----------------------------------------------------------------------------
// getStartPos()
// ----------------------------------------------------------------------------
// getStartPos calculates the starting position of a tile animation based on the player ID.
func getStartPos(playerID int, targetPos fyne.Position, winSize fyne.Size) fyne.Position {
	switch playerID {
	case 0: // Human - Bottom
		return fyne.NewPos(targetPos.X, winSize.Height)
	case 1: // AI#1 - Left
		return fyne.NewPos(-boardCellSize.Width, targetPos.Y)
	case 2: // AI#2 - Top
		return fyne.NewPos(targetPos.X, -boardCellSize.Height)
	case 3: // AI#3 - Right
		return fyne.NewPos(winSize.Width, targetPos.Y)
	default:
		return targetPos
	}
}

// ----------------------------------------------------------------------------
// animateTileIn()
// ----------------------------------------------------------------------------
// animateTileIn handles the visual movement of a tile from the window border to its destination.
func animateTileIn(tile grummi.Tile, targetCell *fyne.Container, playerID int, idx int) {
	// 1. Render a phantom tile for animation
	phantom := container.NewStack(renderTile(&tile))
	phantom.Resize(boardCellSize) // Phantom tile should be the same size as a board cell
	overlay.Add(phantom)

	// 2. Determine positions relative to overlay
	c := myWindow.Canvas()
	if c == nil {
		overlay.Remove(phantom)
		// If the canvas is not ready, ensure the real tile is shown immediately
		if len(targetCell.Objects) > 1 {
			targetCell.Objects[1].Show()
			targetCell.Refresh()
		}
		return
	}

	winSize := myWindow.Content().Size()
	statusMsgHeight := statusMsg.MinSize().Height

	// These are the calculated minimum sizes for the gameBoard based on its children's minimum sizes.
	// Since gameBoard is centered, these are effectively its rendered dimensions.
	// gameBoardWidth = 10 (gap) + 250 (leftColumn) + 10 (gap) + 1120 (centerColumn) + 10 (gap) = 1400
	const gameBoardRenderedWidth = 1400.0
	// gameBoardHeight = max(leftColumn.Height (592), centerColumn.Height (594)) = 594
	const gameBoardRenderedHeight = 594.0

	// Calculate gameBoard's top-left position within the window content area (excluding statusMsg at bottom)
	gameBoardOffsetX := (winSize.Width - gameBoardRenderedWidth) / 2
	gameBoardOffsetY := (winSize.Height - statusMsgHeight - gameBoardRenderedHeight) / 2

	// The gameTable is located within the centerColumn, which is offset within the gameBoard HBox.
	// gameBoard HBox children: gap (10), leftColumn (250), gap (10), centerColumn (table starts here)
	tableGlobalOffsetX := gameBoardOffsetX + float32(10) + float32(250) + float32(10)
	tableGlobalOffsetY := gameBoardOffsetY // centerColumn is aligned to the top of gameBoard

	finalPos := fyne.NewPos(tableGlobalOffsetX+float32(idx%28)*boardCellSize.Width, tableGlobalOffsetY+float32(idx/28)*boardCellSize.Height)

	startPos := getStartPos(playerID, finalPos, winSize)

	phantom.Move(startPos)

	// 3. Create and start animation
	// playPoc() // Play sound once at the start of the movement
	PlayTileSound()
	anim := canvas.NewPositionAnimation(startPos, finalPos, 500*time.Millisecond, func(p fyne.Position) {
		phantom.Move(p)
		phantom.Refresh()
	})
	anim.Start()

	// Cleanup after animation completes (500ms)
	time.AfterFunc(500*time.Millisecond, func() {
		fyne.Do(func() {
			overlay.Remove(phantom)
			if len(targetCell.Objects) > 1 {
				targetCell.Objects[1].Show()
				targetCell.Refresh()
			}
		})
	})
}

// ----------------------------------------------------------------------------
// animateTileToRack()
// ----------------------------------------------------------------------------
// animateTileToRack handles the visual movement of a drawn tile from the right window border to its rack slot.
func animateTileToRack(tile grummi.Tile, targetCell *fyne.Container, idx int) {
	phantom := container.NewStack(renderTile(&tile))
	phantom.Resize(rackCellSize)
	overlay.Add(phantom)
	c := myWindow.Canvas()
	if c == nil {
		overlay.Remove(phantom)
		// If the canvas is not ready, ensure the real tile is shown immediately
		if len(targetCell.Objects) > 1 {
			targetCell.Objects[1].Show()
			targetCell.Refresh()
		}
		return
	}
	winSize := myWindow.Content().Size()
	statusMsgHeight := statusMsg.MinSize().Height

	const gameBoardRenderedWidth = 1400.0
	const gameBoardRenderedHeight = 594.0

	gameBoardOffsetX := (winSize.Width - gameBoardRenderedWidth) / 2
	gameBoardOffsetY := (winSize.Height - statusMsgHeight - gameBoardRenderedHeight) / 2

	// Calculate centerColumn's top-left position within the window content area
	centerColumnGlobalOffsetX := gameBoardOffsetX + float32(10) + float32(250) + float32(10)
	centerColumnGlobalOffsetY := gameBoardOffsetY

	// rackAndButtonsContainer is the 3rd child of centerColumn (index 2)
	// Its Y position relative to centerColumn is fixedTable.Size().Height + vGapCenter.MinSize().Height (416 + 20 = 436)
	rackAndButtonsContainerOffsetYInCenterColumn := (float32(8) * boardCellSize.Height) + float32(20)
	fixedRackOffsetXInRackAndButtonsContainer := float32(10) // fixedRack is the 2nd child of rackAndButtonsContainer (index 1), after a 10px gap
	finalPos := fyne.NewPos(centerColumnGlobalOffsetX+fixedRackOffsetXInRackAndButtonsContainer+float32(idx%20)*rackCellSize.Width, centerColumnGlobalOffsetY+rackAndButtonsContainerOffsetYInCenterColumn+float32(idx/20)*rackCellSize.Height)
	startPos := fyne.NewPos(winSize.Width, finalPos.Y) // Start from the right edge

	phantom.Move(startPos)

	// Create and start the position animation
	PlayTileSound() // Play sound once when drawing
	anim := canvas.NewPositionAnimation(startPos, finalPos, 500*time.Millisecond, func(p fyne.Position) {
		phantom.Move(p)
		phantom.Refresh()
	})
	anim.Start()

	// Cleanup: remove phantom and show the real tile in the rack
	time.AfterFunc(500*time.Millisecond, func() {
		fyne.Do(func() {
			overlay.Remove(phantom)
			if len(targetCell.Objects) > 1 {
				targetCell.Objects[1].Show()
				targetCell.Refresh()
			}
		})
	})
}

// ----------------------------------------------------------------------------
// syncUItoGameState()
// ----------------------------------------------------------------------------
// syncUItoGameState reads the current state of the UI grids and updates the underlying game logic.
// It returns true if the current table configuration is valid.
func syncUItoGameState() bool {
	// 1. Extract Hand from Rack
	newHand := []grummi.Tile{}
	for i := 0; i < 80; i++ {
		if t := getTileAtCell(playerRack, i); t != nil {
			newHand = append(newHand, *t)
		}
	}

	// 2. Extract Table combinations from GameTable
	// We scan row by row, grouping contiguous tiles into combinations.
	newTable := [][]grummi.Tile{}
	var currentCombo []grummi.Tile
	const cols = 28

	for i := 0; i < 224; i++ {
		t := getTileAtCell(gameTable, i)
		if t != nil {
			currentCombo = append(currentCombo, *t)
		}

		// A combination ends if we hit an empty cell or the end of a row
		isEndOfRow := (i+1)%cols == 0
		if (t == nil || isEndOfRow) && len(currentCombo) > 0 {
			newTable = append(newTable, currentCombo)
			currentCombo = nil
		}
	}

	// 3. Validation Logic
	// Check if all combinations on the table are valid
	for _, combo := range newTable {
		if !grummi.IsValidCombination(combo) {
			SetStatus(grummi.T("err_invalid_move"))
			PlayNogoodSound() // Play Nogood sound for invalid move
			return false
		}
	}

	// 4. Check if any action was taken (tiles played from hand or table modified)
	handSizeUnchanged := len(newHand) == len(gameState.Players[0].Hand)
	tableUnchanged := compareTables(gameState.Table, newTable) // New helper function

	if handSizeUnchanged && tableUnchanged {
		SetStatus(grummi.T("err_no_action"))
		PlayNogoodSound() // Play Nogood sound for invalid move
		return false
	}

	// 5. Handle the "Opening" rule (25,30 or 50 points minimum for the first play)
	if !gameState.Players[0].HasPlayedFirst {
		oldVal := calculateTableValue(gameState.Table)
		newVal := calculateTableValue(newTable)
		playedPoints := newVal - oldVal

		if playedPoints < gameState.RequiredOpeningPoints {
			PlayNogoodSound() // Play Nogood sound for invalid move
			SetStatus(grummi.T("err_opening_refused", playedPoints, gameState.RequiredOpeningPoints))
			return false
		}
		gameState.Players[0].HasPlayedFirst = true
		SetStatus(grummi.T("status_opening_ok"))
	}

	// 6. Update the game state if all checks pass
	gameState.Players[0].Hand = newHand
	gameState.Table = newTable
	return true
}

// ----------------------------------------------------------------------------
// compareTables()
// ----------------------------------------------------------------------------
// compareTables performs a deep comparison of two game tables,
// considering combinations and tiles within them.
// It sorts combinations and tiles to ensure that rearrangements don't
// falsely indicate a change.
func compareTables(table1, table2 [][]grummi.Tile) bool {
	if len(table1) != len(table2) {
		return false
	}

	// Create sortable copies of the tables
	sortedTable1 := make([][]grummi.Tile, len(table1))
	sortedTable2 := make([][]grummi.Tile, len(table2))

	for i, combo := range table1 {
		c := make([]grummi.Tile, len(combo))
		copy(c, combo)
		grummi.SortTiles(c) // Sort tiles within each combination
		sortedTable1[i] = c
	}
	for i, combo := range table2 {
		c := make([]grummi.Tile, len(combo))
		copy(c, combo)
		grummi.SortTiles(c) // Sort tiles within each combination
		sortedTable2[i] = c
	}

	// Sort the combinations themselves (e.g., by the first tile's value and color)
	sort.Slice(sortedTable1, func(i, j int) bool {
		return compareCombinationsForSorting(sortedTable1[i], sortedTable1[j]) < 0
	})
	sort.Slice(sortedTable2, func(i, j int) bool {
		return compareCombinationsForSorting(sortedTable2[i], sortedTable2[j]) < 0
	})

	// Now compare the sorted tables element by element
	for i := range sortedTable1 {
		if !areCombinationsEqual(sortedTable1[i], sortedTable2[i]) {
			return false
		}
	}
	return true
}

// ----------------------------------------------------------------------------
// areCombinationsEqual()
// ----------------------------------------------------------------------------
// areCombinationsEqual checks if two combinations are identical (after sorting their tiles).
func areCombinationsEqual(combo1, combo2 []grummi.Tile) bool {
	if len(combo1) != len(combo2) {
		return false
	}
	for i := range combo1 {
		if combo1[i] != combo2[i] {
			return false
		}
	}
	return true
}

// ----------------------------------------------------------------------------
// compareCombinationsForSorting()
// ----------------------------------------------------------------------------
// compareCombinationsForSorting compares two combinations for sorting purposes.
// Returns -1 if combo1 < combo2, 0 if equal, 1 if combo1 > combo2.
func compareCombinationsForSorting(combo1, combo2 []grummi.Tile) int {
	if len(combo1) == 0 && len(combo2) == 0 {
		return 0
	}
	if len(combo1) == 0 {
		return -1
	}
	if len(combo2) == 0 {
		return 1
	}

	// Compare by first tile
	if combo1[0].Color != combo2[0].Color {
		if combo1[0].Color < combo2[0].Color {
			return -1
		}
		return 1
	}
	if combo1[0].Value != combo2[0].Value {
		if combo1[0].Value < combo2[0].Value {
			return -1
		}
		return 1
	}

	// If first tiles are equal, compare by length
	if len(combo1) != len(combo2) {
		if len(combo1) < len(combo2) {
			return -1
		}
		return 1
	}

	// If all else equal, compare tile by tile
	for i := range combo1 {
		if combo1[i] != combo2[i] {
			// This should ideally not happen if tiles are unique and sorted,
			// but as a fallback for full comparison.
			if combo1[i].Color != combo2[i].Color {
				if combo1[i].Color < combo2[i].Color {
					return -1
				}
				return 1
			}
			if combo1[i].Value != combo2[i].Value {
				if combo1[i].Value < combo2[i].Value {
					return -1
				}
				return 1
			}
		}
	}
	return 0
}

// ----------------------------------------------------------------------------
// getTileAtCell()
// ----------------------------------------------------------------------------
// getTileAtCell is a helper to retrieve the grummi.Tile pointer from a specific cell in a grid.
func getTileAtCell(grid *fyne.Container, idx int) *grummi.Tile {
	if idx < 0 || idx >= len(grid.Objects) {
		return nil
	}
	// Structure: GridWrap -> Stack -> [HoverCell, DragTile]
	wrapper := grid.Objects[idx].(*fyne.Container)
	cellStack := wrapper.Objects[0].(*fyne.Container)
	if len(cellStack.Objects) > 1 {
		if dt, ok := cellStack.Objects[1].(*DragTile); ok {
			return dt.tile
		}
	}
	return nil
}

// ----------------------------------------------------------------------------
// calculateTableValue()
// ----------------------------------------------------------------------------
// calculateTableValue sums the points of all combinations currently on the table.
func calculateTableValue(table [][]grummi.Tile) int {
	total := 0
	for _, combo := range table {
		// We use the exported GetComboValueWithJoker from the grummi package
		total += grummi.GetComboValueWithJoker(combo, grummi.IsValidRun(combo))
	}
	return total
}

// ----------------------------------------------------------------------------
// isCellOccupied()
// ----------------------------------------------------------------------------
// isCellOccupied checks if a specific cell in the gameTable contains a tile.
func isCellOccupied(grid *fyne.Container, idx int) bool {
	if idx < 0 || idx >= len(grid.Objects) {
		return false
	}
	wrapper := grid.Objects[idx].(*fyne.Container)
	cellStack := wrapper.Objects[0].(*fyne.Container)
	return len(cellStack.Objects) > 1
}

// ----------------------------------------------------------------------------
// getRunColor()
// ----------------------------------------------------------------------------
// getRunColor returns the color of a run by finding the first non-joker tile.
func getRunColor(combo []grummi.Tile) grummi.Color {
	for _, t := range combo {
		if t.Value != 0 {
			return t.Color
		}
	}
	return grummi.Red
}

// ----------------------------------------------------------------------------
// getTileValueInRun()
// ----------------------------------------------------------------------------
// getTileValueInRun deduces the intended value of a tile (including jokers) within a run.
func getTileValueInRun(combo []grummi.Tile, index int) int {
	t := combo[index]
	if t.Value != 0 {
		return t.Value
	}

	// For a Joker (Value 0), we must determine its logical value in the run.
	// We scan the combination to find internal gaps and then fill ends, matching grummi's scoring logic.
	var realTiles []int
	jokerCount := 0
	for _, tile := range combo {
		if tile.Value == 0 {
			jokerCount++
		} else {
			realTiles = append(realTiles, tile.Value)
		}
	}

	if len(realTiles) == 0 {
		return 0
	}
	sort.Ints(realTiles)

	assignedValues := make(map[int]int)
	used := 0
	// 1. Fill internal gaps (e.g., between 1 and 3)
	for i := 0; i < len(realTiles)-1; i++ {
		for v := realTiles[i] + 1; v < realTiles[i+1]; v++ {
			if used < jokerCount {
				assignedValues[used] = v
				used++
			}
		}
	}
	// 2. Fill high end and then low end
	high := realTiles[len(realTiles)-1]
	for high < 13 && used < jokerCount {
		high++
		assignedValues[used] = high
		used++
	}
	low := realTiles[0]
	for low > 1 && used < jokerCount {
		low--
		assignedValues[used] = low
		used++
	}

	thisJokerIdx := 0
	for i := 0; i < index; i++ {
		if combo[i].Value == 0 {
			thisJokerIdx++
		}
	}
	return assignedValues[thisJokerIdx]
}

// ----------------------------------------------------------------------------
// setMenu()
// ----------------------------------------------------------------------------
func setMenu() {
	newItem := fyne.NewMenuItem(grummi.T("menu_new_game"), func() { showNewGameDialog(myWindow, onNewGame) })
	quitItem := fyne.NewMenuItem(grummi.T("menu_quit"), func() { confirmExit() })

	currentTheme := myApp.Preferences().StringWithFallback("AppTheme", "light")
	currentLang := myApp.Preferences().StringWithFallback("AppLanguage", "fr")

	// Settings Sub-menu: Appearance
	darkThemeItem := fyne.NewMenuItem(grummi.T("menu_theme_dark"), func() {
		SetStatus(grummi.T("status_theme_dark"))
		myApp.Settings().SetTheme(&compactTheme{Theme: theme.DarkTheme()})
		myApp.Preferences().SetString("AppTheme", "dark")
		updateBackgroundColor()
		setMenu()
		myWindow.Content().Refresh()
	})
	darkThemeItem.Checked = currentTheme == "dark"

	lightThemeItem := fyne.NewMenuItem(grummi.T("menu_theme_light"), func() {
		SetStatus(grummi.T("status_theme_light"))
		myApp.Settings().SetTheme(&compactTheme{Theme: theme.LightTheme()})
		myApp.Preferences().SetString("AppTheme", "light")
		updateBackgroundColor()
		setMenu()
		myWindow.Content().Refresh()
	})
	lightThemeItem.Checked = currentTheme == "light"

	displayItem := fyne.NewMenuItem(grummi.T("menu_display"), nil)
	displayItem.ChildMenu = fyne.NewMenu("", darkThemeItem, lightThemeItem)

	// Settings Sub-menu: Language
	enItem := fyne.NewMenuItem("English", func() {
		grummi.SetLanguage("en")
		myApp.Preferences().SetString("AppLanguage", "en")
		SetStatus(grummi.T("status_lang_changed", "English"))
		setMenu()
	})
	enItem.Checked = currentLang == "en"

	frItem := fyne.NewMenuItem("Français", func() {
		grummi.SetLanguage("fr")
		myApp.Preferences().SetString("AppLanguage", "fr")
		SetStatus(grummi.T("status_lang_changed", "Français"))
		setMenu()
	})
	frItem.Checked = currentLang == "fr"

	languageItem := fyne.NewMenuItem(grummi.T("menu_language"), nil)
	languageItem.ChildMenu = fyne.NewMenu("", enItem, frItem)

	settingsMenu := fyne.NewMenu(grummi.T("menu_settings"), displayItem, languageItem)

	// Add it to our menu bar
	mainMenu := fyne.NewMainMenu(
		fyne.NewMenu(grummi.T("menu_file"), newItem, quitItem),
		settingsMenu,
		fyne.NewMenu(grummi.T("menu_help"), fyne.NewMenuItem(grummi.T("menu_about"), func() { showAbout(myWindow) })),
	)
	myWindow.SetMainMenu(mainMenu)
}

// ----------------------------------------------------------------------------
// confirmExit()
// ----------------------------------------------------------------------------
func confirmExit() {
	SetStatus(grummi.T("dialog_confirm_title"))
	d := dialog.NewConfirm(grummi.T("dialog_confirm_title"), grummi.T("dialog_confirm_quit"), func(confirm bool) {
		if confirm {
			myApp.Quit()
		}
	}, myWindow)
	d.Show()
}

// ----------------------------------------------------------------------------
// showAbout()
// ----------------------------------------------------------------------------
func showAbout(win fyne.Window) {
	SetStatus(grummi.T("menu_about"))
	info := APP_NAME + "\n" +
		grummi.T("label_version") + " " + getFullVersion() + "\n\n" +
		grummi.T("app_description") + "\n\n" +
		APP_URL + "\n\n" +
		APP_COPYRIGHT
	dialog.ShowInformation(grummi.T("menu_about"), info, win)
}

// ----------------------------------------------------------------------------
// readPreferences()
// ----------------------------------------------------------------------------
func readPreferences() {
	SetStatus(grummi.T("status_reading_prefs"))
	langPref := myApp.Preferences().StringWithFallback("AppLanguage", "fr")
	grummi.SetLanguage(langPref)
	themePref := myApp.Preferences().StringWithFallback("AppTheme", "light")
	if themePref == "dark" {
		myApp.Settings().SetTheme(&compactTheme{Theme: theme.DarkTheme()})
	} else {
		myApp.Settings().SetTheme(&compactTheme{Theme: theme.LightTheme()})
	}
}

// ****************************************************************************
// getFullVersion()
// ****************************************************************************
func getFullVersion() string {
	return fmt.Sprintf("%s.%s", MAJOR, GitVersion)
}

// ****************************************************************************
// updateStatusTiles()
// ****************************************************************************
func updateStatusTiles() {
	if len(gameState.Players) > 0 {
		statusLabel.SetText(fmt.Sprintf("%d", gameState.TurnNumber)) // Display turn number
		statusDrawLabel.SetText(fmt.Sprintf("%d", len(gameState.Remaining)))
	}

	for i := 0; i < 4; i++ {
		if i < len(gameState.Players) {
			p := gameState.Players[i]
			statusNames[i].Text = p.Name // For the "TUILES" section
			if p.HasPlayedFirst {
				statusNames[i].Color = ColorRummyGreen
			} else {
				statusNames[i].Color = ColorRummyRed
			}
			statusNames[i].Refresh()
			statusTiles[i].SetText(fmt.Sprintf("%d", len(p.Hand)))
		} else {
			statusNames[i].Text = "-"
			statusNames[i].Color = theme.ForegroundColor()
			statusNames[i].Refresh()
			statusTiles[i].SetText("-")
		}
	}

	// Collect stats for sorting and display in the "STATS" section
	var currentStats []playerStats
	for _, p := range gameState.Players {
		wins := myApp.Preferences().Int(fmt.Sprintf("Stats_%s_Wins", p.Name))
		games := myApp.Preferences().Int(fmt.Sprintf("Stats_%s_Games", p.Name))
		score := myApp.Preferences().Int(fmt.Sprintf("Stats_%s_Score", p.Name))
		currentStats = append(currentStats, playerStats{
			Name:  p.Name,
			Wins:  wins,
			Games: games,
			Score: score,
		})
	}

	// Sort by score (descending)
	sort.Slice(currentStats, func(i, j int) bool {
		return currentStats[i].Score > currentStats[j].Score
	})

	// Update the stats grid labels with sorted data
	for i := 0; i < 4; i++ {
		if i < len(currentStats) {
			statsPlayerLabels[i].SetText(currentStats[i].Name)
			statsWinsLabels[i].SetText(fmt.Sprintf("%d", currentStats[i].Wins))
			statsGamesLabels[i].SetText(fmt.Sprintf("%d", currentStats[i].Games))
			statsPointsLabels[i].SetText(fmt.Sprintf("%d", currentStats[i].Score))
		} else {
			// Clear unused rows
			statsPlayerLabels[i].SetText("-")
			statsWinsLabels[i].SetText("-")
			statsGamesLabels[i].SetText("-")
			statsPointsLabels[i].SetText("-")
		}
		statsPlayerLabels[i].Refresh()
		statsWinsLabels[i].Refresh()
		statsGamesLabels[i].Refresh()
		statsPointsLabels[i].Refresh()
	}
}

// ****************************************************************************
// SetStatus()
// ****************************************************************************
func SetStatus(msg string) {
	msg = stripANSI(msg)
	statusMsg.SetText(msg)
	appendAIMessage(msg)
}

// ****************************************************************************
// showNewGameDialog()
// ****************************************************************************
func showNewGameDialog(win fyne.Window, startCallback func(playerName string, aiCount int, timeLimit int, openingPoints int)) {
	// 1. Champ pour le nom du joueur
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder(grummi.T("placeholder_name"))

	// Optionnel: On peut recharger le dernier nom utilisé depuis les préférences
	nameEntry.SetText(fyne.CurrentApp().Preferences().StringWithFallback("PlayerName", "Humain"))

	// 1bis. Sélecteur pour les points d'ouverture
	openingOptions := []string{"25", "30", "50"}
	openingSelect := widget.NewSelect(openingOptions, nil)
	openingSelect.SetSelected(fyne.CurrentApp().Preferences().StringWithFallback("OpeningPoints", "30"))

	// 2. Sélecteur pour le nombre d'adversaires (de 1 à 3)
	aiSelect := widget.NewSelect([]string{"1", "2", "3"}, nil)
	aiSelect.SetSelected(fyne.CurrentApp().Preferences().StringWithFallback("AICount", "3"))

	// 2bis. Sélecteur pour la limite de temps
	timeOptions := []string{grummi.T("option_no_limit"), "1 min", "2 min", "3 min", "4 min", "5 min"}
	timeSelect := widget.NewSelect(timeOptions, nil)
	timeSelect.SetSelected(fyne.CurrentApp().Preferences().StringWithFallback("TimeLimit", timeOptions[0]))

	// 3. Mise en page du formulaire
	form := widget.NewForm(
		widget.NewFormItem(grummi.T("label_your_name"), nameEntry),
		widget.NewFormItem(grummi.T("label_opening_points"), openingSelect),
		widget.NewFormItem(grummi.T("label_ai_opponents"), aiSelect),
		widget.NewFormItem(grummi.T("label_time_limit"), timeSelect),
	)

	// 4. Création du dialogue avec boutons Confirmer/Annuler
	dialog.ShowCustomConfirm(
		grummi.T("dialog_new_game_title"), // Titre
		grummi.T("btn_start"),             // Bouton de validation
		grummi.T("btn_cancel"),            // Bouton d'annulation
		form,                              // Le contenu du formulaire
		func(confirmed bool) {
			if confirmed {
				// On convertit le choix de l'IA en entier
				aiCount := 1
				switch aiSelect.Selected {
				case "2":
					aiCount = 2
				case "3":
					aiCount = 3
				}

				timeLimit := 0
				fmt.Sscanf(timeSelect.Selected, "%d", &timeLimit)

				openingPoints := 30
				fmt.Sscanf(openingSelect.Selected, "%d", &openingPoints)

				nomJoueur := nameEntry.Text
				if nomJoueur == "" {
					nomJoueur = "Humain" // Sécurité si le nom est vide
				}

				// On sauvegarde les préférences pour la prochaine fois
				fyne.CurrentApp().Preferences().SetString("PlayerName", nomJoueur)
				fyne.CurrentApp().Preferences().SetString("OpeningPoints", openingSelect.Selected)
				fyne.CurrentApp().Preferences().SetString("AICount", aiSelect.Selected)
				fyne.CurrentApp().Preferences().SetString("TimeLimit", timeSelect.Selected)

				// On lance le callback avec les données récupérées
				startCallback(nomJoueur, aiCount, timeLimit, openingPoints)
			} else {
				// If the user cancels the dialog, quit the application
				myApp.Quit()
			}
		},
		win,
	)
}

// ****************************************************************************
// onNewGame()
// ****************************************************************************
func onNewGame(name string, ais int, timeLimit int, openingPoints int) {
	if isGameLoading {
		return
	}
	isGameLoading = true
	stopHumanTimer()

	turnLimitMinutes = timeLimit

	go func() {
		defer func() { isGameLoading = false }()

		gameLogger := &uiLogger{}
		newGameState := grummi.InitializeGame(ais+1, gameLogger)
		newGameState.Players[0].Name = name
		newGameState.RequiredOpeningPoints = openingPoints

		// Swap global state immediately so logs/refreshes use new data
		fyne.Do(func() {
			gameState = newGameState
			gameState.TurnNumber = 1
			refreshRack()
			refreshTable()

			if turnLimitMinutes == 0 {
				statusLimitLabel.Text = grummi.T("label_time_limit") + " " + grummi.T("option_no_limit")
			} else {
				statusLimitLabel.Text = grummi.T("label_time_limit") + " " + fmt.Sprintf("%d min", turnLimitMinutes)
			}
			statusOpeningPointsLabel.Text = grummi.T("label_opening_points") + " " + fmt.Sprintf("%d", openingPoints)
			statusOpeningPointsLabel.Refresh()
			statusLimitLabel.Refresh()
		})

		// DetermineFirstPlayer contains sleeps/logs; now safe as global state is swapped
		firstPlayerID := gameState.DetermineFirstPlayer()
		gameState.CurrentPlayerID = firstPlayerID

		fyne.Do(func() {
			updateStatusTiles()
		})

		playNextTurn()
	}()
}

// ----------------------------------------------------------------------------
// performHumanDraw()
// ----------------------------------------------------------------------------
// performHumanDraw executes an automatic or manual draw for the human player and ends their turn.
func performHumanDraw() {
	if isTurnProcessing {
		return
	}

	fyne.Do(func() {
		stopHumanTimer()
		humanTimerElapsed = 0

		if len(gameState.Remaining) > 0 {
			drawnTile := gameState.Remaining[0]
			gameState.DrawTile()
			refreshRack()

			// Find where the new tile was placed in the rack to animate it
			targetIdx := -1
			for i := 0; i < 80; i++ {
				if t := getTileAtCell(playerRack, i); t != nil && *t == drawnTile {
					targetIdx = i
					break
				}
			}

			SetStatus(grummi.T("status_drawn", len(gameState.Remaining)))
			if targetIdx != -1 {
				cellStack := playerRack.Objects[targetIdx].(*fyne.Container).Objects[0].(*fyne.Container)
				cellStack.Objects[1].Hide() // Hide initially to let animation play
				animateTileToRack(drawnTile, cellStack, targetIdx)
			}
		} else {
			SetStatus(grummi.T("err_draw_pile_empty"))
		}

		gameState.ConsecutivePasses++ // Drawing counts as a pass for stalemate
		gameState.TurnNumber++        // Increment turn number after human draws
		updateStatusTiles()           // Refresh status display
		if checkGameEnd() {
			return // Game ended
		}
		gameState.CurrentPlayerID = (gameState.CurrentPlayerID + 1) % len(gameState.Players) // End human turn
		playNextTurn()
	})
}

// ****************************************************************************
// playNextTurn()
// ****************************************************************************
// playNextTurn manages the game flow, handling AI turns automatically
// and setting up for the human player's turn.
func playNextTurn() {
	if isTurnProcessing {
		return
	}
	isTurnProcessing = true

	go func() {
		defer func() { isTurnProcessing = false }()

		for gameState.Players[gameState.CurrentPlayerID].IsAI { // Loop through AI turns until it's the human player's turn
			if isPaused {
				time.Sleep(200 * time.Millisecond)
				continue
			}

			currentPlayer := &gameState.Players[gameState.CurrentPlayerID]

			// Initial "thinking" pause
			time.Sleep(1 * time.Second)

			// Execute AI turn
			// Note: The UI now refreshes inside IATurn via the Logger
			gameState.IATurn(currentPlayer)

			// Final refresh after turn
			fyne.Do(func() {
				refreshTable()
				refreshRack()
			})

			// Check if the AI won or stalemate occurred
			if checkGameEnd() {
				return
			}

			// Move to the next player
			gameState.CurrentPlayerID = (gameState.CurrentPlayerID + 1) % len(gameState.Players)
		}

		// It's now the human player's turn
		fyne.Do(func() {
			SetStatus(grummi.T("status_human_turn", gameState.Players[0].Name))
			humanTimerElapsed = 0
			startHumanTimer()
			flashTimer()
		})
	}()
}

// ****************************************************************************
// appendAIMessage()
// ****************************************************************************
func appendAIMessage(msg string) {
	currentText := aiLogEntry.Text
	if currentText != "" {
		aiLogEntry.SetText(currentText + "\n> " + msg)
	} else {
		aiLogEntry.SetText("> " + msg)
	}

	aiLogScroll.ScrollToBottom()
}

// ****************************************************************************
// checkGameEnd()
// ****************************************************************************
// checkGameEnd checks for win or stalemate conditions and displays a dialog if the game ends.
// Returns true if the game has ended, false otherwise.
func checkGameEnd() bool {
	currentPlayer := &gameState.Players[gameState.CurrentPlayerID]

	isWin := len(currentPlayer.Hand) == 0
	isStalemate := gameState.ConsecutivePasses >= len(gameState.Players) && len(gameState.Remaining) == 0

	if isWin || isStalemate {
		winnerID := -1
		if isWin {
			winnerID = currentPlayer.ID
		}

		// Calculate points to save in statistics
		playerPoints := make(map[int]int)
		totalOpponentPoints := 0
		for _, p := range gameState.Players {
			handPoints := grummi.CalculateHandPoints(p.Hand)
			if p.ID == winnerID {
				playerPoints[p.ID] = 0
			} else {
				playerPoints[p.ID] = -handPoints
				totalOpponentPoints += handPoints
			}
		}
		if winnerID != -1 {
			playerPoints[winnerID] = totalOpponentPoints
		}

		// Save statistics to Preferences
		prefs := myApp.Preferences()
		for _, p := range gameState.Players {
			gKey := fmt.Sprintf("Stats_%s_Games", p.Name)
			wKey := fmt.Sprintf("Stats_%s_Wins", p.Name)
			sKey := fmt.Sprintf("Stats_%s_Score", p.Name)

			prefs.SetInt(gKey, prefs.Int(gKey)+1)
			if p.ID == winnerID {
				prefs.SetInt(wKey, prefs.Int(wKey)+1)
			}
			prefs.SetInt(sKey, prefs.Int(sKey)+playerPoints[p.ID])
		}

		PlayWinnerSound() // Play a win sound effect

		fyne.Do(func() {
			if isWin {
				msgKey := "msg_win_human"
				if currentPlayer.IsAI {
					msgKey = "msg_win_ai"
				}
				winMsg := grummi.T(msgKey, currentPlayer.Name)
				SetStatus(winMsg)
				showGameOverDialog(winMsg, currentPlayer.ID)
			} else {
				SetStatus(grummi.T("msg_stalemate"))
				showGameOverDialog(grummi.T("msg_stalemate"), -1)
			}
			updateStatusTiles() // Refresh to show new stats immediately
		})
		return true
	}
	return false
}

// ****************************************************************************
// showGameOverDialog()
// ****************************************************************************
// showGameOverDialog displays a dialog with game over information.
func showGameOverDialog(message string, winnerID int) {
	// Prepare final scores and hands for display in the dialog
	scoreDetails := grummi.T("label_scores") + "\n"
	playerPoints := make(map[int]int)
	totalOpponentPoints := 0

	for _, p := range gameState.Players {
		handPoints := grummi.CalculateHandPoints(p.Hand)
		if p.ID == winnerID {
			playerPoints[p.ID] = 0
		} else {
			playerPoints[p.ID] = -handPoints
			totalOpponentPoints += handPoints
		}
	}

	if winnerID != -1 {
		playerPoints[winnerID] = totalOpponentPoints
	}

	for _, p := range gameState.Players {
		scoreDetails += fmt.Sprintf("  %s: %d points\n", p.Name, playerPoints[p.ID])
	}

	handDetails := "\n" + grummi.T("label_final_hands") + "\n"
	for _, p := range gameState.Players {
		grummi.SortTiles(p.Hand)
		handDetails += fmt.Sprintf("  %s: ", p.Name)
		for _, tile := range p.Hand {
			handDetails += stripANSI(grummi.FormatTile(tile)) // Strip ANSI here
		}
		handDetails += "\n"
	}

	content := container.NewVBox(
		widget.NewLabel(stripANSI(message)), // Strip ANSI here
		widget.NewLabel(scoreDetails),
		widget.NewLabel(handDetails),
	)

	dialog.ShowCustomConfirm(
		grummi.T("game_over"),
		grummi.T("menu_new_game"),
		"OK",
		content,
		func(confirmed bool) {
			if confirmed {
				showNewGameDialog(myWindow, onNewGame)
			}
		},
		myWindow,
	)
}

// ****************************************************************************
// startHumanTimer()
// ****************************************************************************
func startHumanTimer() {
	stopHumanTimer() // Clean up any existing timer

	timerStop = make(chan bool)
	localStop := timerStop
	startTime := time.Now().Add(-humanTimerElapsed)

	PlayBeepSound() // Play a beep at the start of the timer

	go func() {
		warningFlashed := false
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-localStop:
				return
			case <-ticker.C:
				elapsed := time.Since(startTime)
				humanTimerElapsed = elapsed
				minutes := int(elapsed.Minutes())
				seconds := int(elapsed.Seconds()) % 60

				fyne.Do(func() {
					if statusTimerLabel != nil {
						statusTimerLabel.Text = fmt.Sprintf("%02d:%02d", minutes, seconds)
						statusTimerLabel.Refresh()
					}
				})

				if turnLimitMinutes > 0 && !warningFlashed && int(elapsed.Seconds()) >= (turnLimitMinutes*60-10) {
					warningFlashed = true
					flashTimer()
				}

				if turnLimitMinutes > 0 && elapsed >= time.Duration(turnLimitMinutes)*time.Minute {
					fyne.Do(func() {
						SetStatus(grummi.T("status_auto_draw"))
					})
					performHumanDraw()
					return
				}
			}
		}
	}()
}

// ----------------------------------------------------------------------------
// togglePause()
// ----------------------------------------------------------------------------
// togglePause switches the game between paused and running states.
// It hides tiles and stops the clock when paused to prevent cheating.
func togglePause() {
	isPaused = !isPaused
	if isPaused {
		stopHumanTimer()
		gameTable.Hide()
		playerRack.Hide()
		pauseBtn.SetIcon(theme.MediaPlayIcon())
		pauseBtn.tooltip = grummi.T("btn_resume")

		confirmBtn.Disable()
		sortBtn.Disable()
		drawBtn.Disable()
		rollbackBtn.Disable()

		SetStatus(grummi.T("status_paused"))
	} else {
		gameTable.Show()
		playerRack.Show()
		pauseBtn.SetIcon(theme.MediaPauseIcon())
		pauseBtn.tooltip = grummi.T("btn_pause")

		confirmBtn.Enable()
		sortBtn.Enable()
		drawBtn.Enable()
		rollbackBtn.Enable()

		SetStatus(grummi.T("status_resumed"))
		if !gameState.Players[gameState.CurrentPlayerID].IsAI {
			startHumanTimer()
		}
	}
	gameTable.Refresh()
	playerRack.Refresh()
}

// ****************************************************************************
// stopHumanTimer()
// ****************************************************************************
func stopHumanTimer() {
	if timerStop != nil {
		close(timerStop)
		timerStop = nil
	}
	if statusTimerLabel != nil {
		fyne.Do(func() {
			statusTimerLabel.Text = "00:00"
			statusTimerLabel.Refresh()
		})
	}
}

// ----------------------------------------------------------------------------
// flashTimer()
// ----------------------------------------------------------------------------
// flashTimer creates a visual color animation on the timer to alert the player.
func flashTimer() {
	if statusTimerLabel == nil {
		return
	}

	// Manual flash to ensure compatibility with all Fyne v2 versions
	go func() {
		originalColor := theme.ForegroundColor()
		// Flash 3 times
		for i := 0; i < 3; i++ {
			fyne.Do(func() {
				statusTimerLabel.Color = ColorRummyRed
				statusTimerLabel.Refresh()
			})
			time.Sleep(300 * time.Millisecond)
			fyne.Do(func() {
				statusTimerLabel.Color = originalColor
				statusTimerLabel.Refresh()
			})
			time.Sleep(300 * time.Millisecond)
		}
	}()
}
