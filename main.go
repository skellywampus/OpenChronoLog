// OpenChronoLog (Go/Fyne)
// Requires Go 1.20+
// go get fyne.io/fyne/v2@latest
// go run main.go
package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	UserDBFile    = "user_database.json"
	ScanLogFile   = "scan_log.json"
	ApprovalsFile = "anomalies_approvals.json"
	TimeLayout    = "2006-01-02 15:04:05"
	DateLayout    = "2006-01-02"

	EntryScan   = "Scan"
	EntryManual = "Manual"
	StatusIn    = "clocked_in"
	StatusOut   = "clocked_out"
)

type User struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type LogEntry struct {
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
	Time      string `json:"time"`
	Status    string `json:"status"`
	EntryType string `json:"entry_type"`
}

var (
	userDB           []User
	scanLog          []LogEntry
	anomalyApprovals map[string]bool
	clockLbl         *widget.Label
	successLb        *widget.Label
	successTimer     *time.Timer
	idEntry          *widget.Entry
	mainWin          fyne.Window
	appRef           fyne.App
	contentHolder    *fyne.Container
	clockBar         *fyne.Container
	storageDir       string
	dataLoadErrors   []string
	popupBindings    []popupKeyboardBinding
	popupBindingID   uint64
	responsiveTables []struct {
		tbl          *widget.Table
		cols         []string
		rowsProvider func() [][]string
	}
)

type popupKeyboardBinding struct {
	id      uint64
	primary func()
}

var (
	popupReturnShortcut = &desktop.CustomShortcut{KeyName: fyne.KeyReturn}
	popupEnterShortcut  = &desktop.CustomShortcut{KeyName: fyne.KeyEnter}
)

var (
	inkColor       = color.NRGBA{R: 23, G: 34, B: 55, A: 255}
	mutedColor     = color.NRGBA{R: 71, G: 84, B: 103, A: 255}
	navyColor      = color.NRGBA{R: 16, G: 24, B: 40, A: 255}
	primaryColor   = color.NRGBA{R: 21, G: 94, B: 239, A: 255}
	surfaceColor   = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	canvasColor    = color.NRGBA{R: 246, G: 248, B: 251, A: 255}
	borderColor    = color.NRGBA{R: 228, G: 231, B: 236, A: 255}
	cardFrameColor = color.Black
)

type workplaceTheme struct {
	base fyne.Theme
}

func (t workplaceTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return canvasColor
	case theme.ColorNameButton:
		return color.NRGBA{R: 234, G: 241, B: 255, A: 255}
	case theme.ColorNameForeground:
		return inkColor
	case theme.ColorNameForegroundOnPrimary:
		return color.White
	case theme.ColorNameHover:
		return color.NRGBA{R: 255, G: 255, B: 255, A: 36}
	case theme.ColorNamePressed:
		return color.NRGBA{R: 16, G: 24, B: 40, A: 32}
	case theme.ColorNameFocus:
		return color.NRGBA{R: 21, G: 94, B: 239, A: 46}
	case theme.ColorNameSelection:
		return color.NRGBA{R: 21, G: 94, B: 239, A: 38}
	case theme.ColorNameInputBackground:
		return surfaceColor
	case theme.ColorNameInputBorder, theme.ColorNameSeparator:
		return borderColor
	case theme.ColorNamePrimary, theme.ColorNameHyperlink:
		return primaryColor
	case theme.ColorNameShadow:
		return color.NRGBA{R: 16, G: 24, B: 40, A: 24}
	}
	return t.base.Color(name, theme.VariantLight)
}

func (t workplaceTheme) Font(style fyne.TextStyle) fyne.Resource    { return t.base.Font(style) }
func (t workplaceTheme) Icon(name fyne.ThemeIconName) fyne.Resource { return t.base.Icon(name) }
func (t workplaceTheme) Size(name fyne.ThemeSizeName) float32 {
	return t.base.Size(name)
}

func surface(content fyne.CanvasObject) fyne.CanvasObject {
	frame := canvas.NewRectangle(cardFrameColor)
	card := canvas.NewRectangle(surfaceColor)
	inner := container.NewMax(card, container.NewPadded(content))
	return container.NewMax(frame, container.New(layout.NewCustomPaddedLayout(5, 5, 5, 5), inner))
}

func sectionHeading(title, detail string) fyne.CanvasObject {
	heading := canvas.NewText(title, inkColor)
	heading.TextStyle = fyne.TextStyle{Bold: true}
	heading.TextSize = 24
	subtitle := canvas.NewText(detail, mutedColor)
	subtitle.TextSize = 14
	return container.NewVBox(heading, subtitle)
}

func actionTile(title, detail string, button *widget.Button) fyne.CanvasObject {
	heading := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	description := widget.NewLabel(detail)
	description.Wrapping = fyne.TextWrapWord
	return surface(container.NewBorder(nil, button, nil, nil, container.NewVBox(heading, description)))
}

func twoPanelWorkspace(title, detail string, left, right fyne.CanvasObject) fyne.CanvasObject {
	header := container.NewVBox(sectionHeading(title, detail), widget.NewSeparator())
	panels := container.NewGridWithColumns(2, surface(left), surface(right))
	return container.NewBorder(header, nil, nil, nil, panels)
}

// ---------- Data IO ----------

func loadData() {
	userDB = []User{}
	scanLog = []LogEntry{}
	anomalyApprovals = map[string]bool{}
	dataLoadErrors = nil

	storageDir = applicationDataDir()
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		dataLoadErrors = append(dataLoadErrors, fmt.Sprintf("Create data folder: %v", err))
	}

	if err := loadJSON(dataFile(UserDBFile), &userDB); err != nil {
		dataLoadErrors = append(dataLoadErrors, err.Error())
	}
	if err := loadJSON(dataFile(ScanLogFile), &scanLog); err != nil {
		dataLoadErrors = append(dataLoadErrors, err.Error())
	}
	if err := loadJSON(dataFile(ApprovalsFile), &anomalyApprovals); err != nil {
		dataLoadErrors = append(dataLoadErrors, err.Error())
	}
}

func loadJSON(path string, destination any) error {
	recoverInterruptedWrite(path)
	contents, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("Read %s: %w", filepath.Base(path), err)
	}
	if len(strings.TrimSpace(string(contents))) == 0 {
		return nil
	}
	if err := json.Unmarshal(contents, destination); err != nil {
		return fmt.Errorf("Read %s: invalid JSON: %w", filepath.Base(path), err)
	}
	return nil
}

func recoverInterruptedWrite(path string) {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return
	}
	_ = os.Rename(path+".previous-write", path)
}

func applicationDataDir() string {
	executable, err := os.Executable()
	if err != nil {
		return "."
	}
	executableDir := filepath.Dir(executable)
	if runtime.GOOS == "darwin" {
		contentsDir := filepath.Dir(executableDir)
		bundleDir := filepath.Dir(contentsDir)
		if filepath.Ext(bundleDir) == ".app" {
			return filepath.Dir(bundleDir)
		}
	}
	return executableDir
}

func dataFile(name string) string {
	if storageDir == "" {
		return name
	}
	return filepath.Join(storageDir, name)
}

func saveData() bool {
	if err := saveJSON(dataFile(ScanLogFile), scanLog); err != nil {
		showSaveError(err)
		return false
	}
	if err := saveJSON(dataFile(UserDBFile), userDB); err != nil {
		showSaveError(err)
		return false
	}
	if anomalyApprovals != nil {
		if err := saveJSON(dataFile(ApprovalsFile), anomalyApprovals); err != nil {
			showSaveError(err)
			return false
		}
	}
	return true
}

func saveJSON(path string, v any) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "    ")
	if err := enc.Encode(v); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return replaceFile(tmp, path)
}

func replaceFile(source, destination string) error {
	previous := destination + ".previous-write"
	_ = os.Remove(previous)
	hadPrevious := false
	if err := os.Rename(destination, previous); err == nil {
		hadPrevious = true
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(source, destination); err != nil {
		if hadPrevious {
			_ = os.Rename(previous, destination)
		}
		_ = os.Remove(source)
		return err
	}
	if hadPrevious {
		_ = os.Remove(previous)
	}
	return nil
}

// ---------- Core Helpers ----------

func findUserByID(id string) *User {
	for i := range userDB {
		if strings.EqualFold(userDB[i].ID, id) {
			return &userDB[i]
		}
	}
	return nil
}

func countClockedIn() int {
	n := 0
	for _, u := range userDB {
		if u.Status == StatusIn {
			n++
		}
	}
	return n
}

func updateClockedInCount() {
	clockLbl.SetText(fmt.Sprintf("%d members currently clocked in", countClockedIn()))
}

func showSuccess(msg string) {
	successLb.SetText(msg)
	successLb.Show()
	if successTimer != nil {
		successTimer.Stop()
	}
	successTimer = time.AfterFunc(2*time.Second, func() {
		// Schedule GUI work on the main thread (works across Fyne versions)
		fyne.Do(func() {
			successLb.Hide()
		})
	})
}

func showSaveError(err error) {
	if mainWin == nil {
		return
	}
	fyne.Do(func() {
		showError(fmt.Errorf("Could not save your changes: %w", err))
	})
}

func focusScanEntry() {
	if mainWin == nil || idEntry == nil {
		return
	}
	mainWin.Canvas().Focus(idEntry)
}

func refreshPopupShortcuts() {
	if mainWin == nil {
		return
	}
	canvas := mainWin.Canvas()
	canvas.RemoveShortcut(popupReturnShortcut)
	canvas.RemoveShortcut(popupEnterShortcut)
	if len(popupBindings) == 0 {
		return
	}
	binding := popupBindings[len(popupBindings)-1]
	handler := func(fyne.Shortcut) {
		if len(popupBindings) == 0 || popupBindings[len(popupBindings)-1].id != binding.id {
			return
		}
		canvas.RemoveShortcut(popupReturnShortcut)
		canvas.RemoveShortcut(popupEnterShortcut)
		binding.primary()
		refreshPopupShortcuts()
	}
	canvas.AddShortcut(popupReturnShortcut, handler)
	canvas.AddShortcut(popupEnterShortcut, handler)
}

func bindPopupEnter(primary func()) uint64 {
	popupBindingID++
	binding := popupKeyboardBinding{id: popupBindingID, primary: primary}
	popupBindings = append(popupBindings, binding)
	refreshPopupShortcuts()
	return binding.id
}

func clearPopupEnter(id uint64) {
	for index := len(popupBindings) - 1; index >= 0; index-- {
		if popupBindings[index].id != id {
			continue
		}
		popupBindings = append(popupBindings[:index], popupBindings[index+1:]...)
		break
	}
	refreshPopupShortcuts()
}

func showDismissDialog(popup dialog.Dialog) {
	id := bindPopupEnter(popup.Dismiss)
	popup.SetOnClosed(func() { clearPopupEnter(id) })
	popup.Show()
}

func showError(err error) {
	showDismissDialog(dialog.NewError(err, mainWin))
}

func showInformation(title, message string) {
	showDismissDialog(dialog.NewInformation(title, message, mainWin))
}

func showConfirm(title, message string, callback func(bool)) {
	popup := dialog.NewConfirm(title, message, callback, mainWin)
	id := bindPopupEnter(popup.Confirm)
	popup.SetOnClosed(func() { clearPopupEnter(id) })
	popup.Show()
}

func nowString() string {
	return time.Now().Format(TimeLayout)
}

func parseLogTime(value string) (time.Time, error) {
	return time.ParseInLocation(TimeLayout, value, time.Local)
}

// ---------- Actions ----------

func submitAction() {
	id := strings.TrimSpace(idEntry.Text)
	if id == "" {
		showError(fmt.Errorf("Member ID cannot be empty"))
		return
	}
	u := findUserByID(id)
	if u == nil {
		showError(fmt.Errorf("Member with ID %s not found", id))
		idEntry.SetText("")
		focusScanEntry()
		return
	}
	wasClockedIn := u.Status == StatusIn
	handleScan(u)
	if !saveData() {
		focusScanEntry()
		return
	}
	updateClockedInCount()
	action := "Clocked in"
	if wasClockedIn {
		action = "Clocked out"
	}
	showSuccess(fmt.Sprintf("%s: %s at %s", action, u.Name, time.Now().Format("3:04 PM")))
	idEntry.SetText("")
	focusScanEntry()
}

func handleScan(u *User) {
	current := nowString()
	// toggle status
	if u.Status == StatusIn {
		u.Status = StatusOut
		calculateDuration(u.ID, current)
	} else {
		u.Status = StatusIn
	}
	scanLog = append(scanLog, LogEntry{
		UserID:    u.ID,
		Name:      u.Name,
		Time:      current,
		Status:    u.Status,
		EntryType: EntryScan,
	})
}

func calculateDuration(userID, clockOut string) {
	// Find last "clocked_in" for user by scanning backwards
	for i := len(scanLog) - 1; i >= 0; i-- {
		e := scanLog[i]
		if e.UserID == userID && e.Status == StatusIn {
			ci, err1 := parseLogTime(e.Time)
			co, err2 := parseLogTime(clockOut)
			if err1 == nil && err2 == nil && co.After(ci) {
				d := co.Sub(ci)
				fmt.Printf("%s was clocked in for %v\n", userID, d)
			}
			break
		}
	}
}

func manualEntryAction() {
	// Transition to a full-page Manual Entry view similar to ManualAdd.py
	setMainContent(buildManualEntryView())
}

func showUserLookup() {
	search := widget.NewEntry()
	search.SetPlaceHolder("Search by ID or name")
	matchedUsers := append([]User(nil), userDB...)
	sort.Slice(matchedUsers, func(i, j int) bool {
		return strings.ToLower(matchedUsers[i].Name) < strings.ToLower(matchedUsers[j].Name)
	})
	selectedIndex := -1
	selection := widget.NewLabel("Select a member from the results below.")

	results := widget.NewList(
		func() int { return len(matchedUsers) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(index widget.ListItemID, object fyne.CanvasObject) {
			user := matchedUsers[index]
			object.(*widget.Label).SetText(fmt.Sprintf("%s  •  %s", user.Name, user.ID))
		},
	)
	results.OnSelected = func(index widget.ListItemID) {
		selectedIndex = index
		user := matchedUsers[index]
		selection.SetText(fmt.Sprintf("Selected: %s (%s)", user.Name, user.ID))
	}

	applySearch := func() {
		query := strings.ToLower(strings.TrimSpace(search.Text))
		matchedUsers = matchedUsers[:0]
		for _, user := range userDB {
			if query == "" || strings.Contains(strings.ToLower(user.ID), query) || strings.Contains(strings.ToLower(user.Name), query) {
				matchedUsers = append(matchedUsers, user)
			}
		}
		sort.Slice(matchedUsers, func(i, j int) bool {
			return strings.ToLower(matchedUsers[i].Name) < strings.ToLower(matchedUsers[j].Name)
		})
		selectedIndex = -1
		selection.SetText(fmt.Sprintf("%d matching members", len(matchedUsers)))
		results.UnselectAll()
		results.Refresh()
	}
	var lookup dialog.Dialog
	var activateSelection func()
	search.OnChanged = func(string) { applySearch() }
	search.OnSubmitted = func(string) {
		if selectedIndex >= 0 {
			activateSelection()
			return
		}
		applySearch()
		if len(matchedUsers) == 1 {
			selectedIndex = 0
			results.Select(0)
			activateSelection()
		}
	}
	searchButton := widget.NewButton("Search", applySearch)
	searchButton.Importance = widget.HighImportance

	activateSelection = func() {
		if selectedIndex < 0 || selectedIndex >= len(matchedUsers) {
			showInformation("Choose a Member", "Select a member from the results list first.")
			return
		}
		idEntry.SetText(matchedUsers[selectedIndex].ID)
		lookup.Hide()
		focusScanEntry()
	}
	useSelected := widget.NewButton("Use Selected Member", activateSelection)
	useSelected.Importance = widget.HighImportance
	closeButton := widget.NewButton("Close", func() {
		lookup.Hide()
		focusScanEntry()
	})
	content := container.NewBorder(
		container.NewVBox(widget.NewLabel("Find a member by name or ID."), container.NewBorder(nil, nil, nil, searchButton, search)),
		container.NewVBox(selection, container.NewHBox(closeButton, useSelected)),
		nil, nil,
		results,
	)
	lookup = dialog.NewCustomWithoutButtons("Find Member", content, mainWin)
	lookup.Resize(fyne.NewSize(500, 420))
	lookupID := bindPopupEnter(func() {
		if selectedIndex >= 0 {
			activateSelection()
			return
		}
		applySearch()
		if len(matchedUsers) == 1 {
			selectedIndex = 0
			results.Select(0)
			activateSelection()
		}
	})
	lookup.SetOnClosed(func() { clearPopupEnter(lookupID) })
	lookup.Show()
	mainWin.Canvas().Focus(search)
}

// buildManualEntryView shows a page to add manual entries
// Fields: User (select), Date (YYYY-MM-DD), Time In/Out (HH:MM:SS)
func buildManualEntryView() fyne.CanvasObject {
	// Build user select options
	opts := make([]string, 0, len(userDB))
	idByDisplay := map[string]string{}
	for _, u := range userDB {
		disp := fmt.Sprintf("%s - %s", u.ID, u.Name)
		opts = append(opts, disp)
		idByDisplay[disp] = u.ID
	}
	userSel := widget.NewSelect(opts, nil)
	userSel.PlaceHolder = "Select Member"

	// Date selection via drop-downs (Month/Day/Year)
	// This avoids relying on a DatePicker widget that may not be available
	months := make([]string, 12)
	for i := 1; i <= 12; i++ {
		months[i-1] = fmt.Sprintf("%02d", i)
	}
	days := make([]string, 31)
	for i := 1; i <= 31; i++ {
		days[i-1] = fmt.Sprintf("%02d", i)
	}
	years := make([]string, 0, 52)
	for y := 2000; y <= 2051; y++ {
		years = append(years, fmt.Sprintf("%d", y))
	}
	now := time.Now()
	selMonth := widget.NewSelect(months, nil)
	selDay := widget.NewSelect(days, nil)
	selYear := widget.NewSelect(years, nil)
	selMonth.SetSelected(now.Format("01"))
	selDay.SetSelected(now.Format("02"))
	selYear.SetSelected(now.Format("2006"))

	// Hours / Minutes duration
	hours := make([]string, 24)
	for i := 0; i < 24; i++ {
		hours[i] = fmt.Sprintf("%02d", i)
	}
	minutes := make([]string, 60)
	for i := 0; i < 60; i++ {
		minutes[i] = fmt.Sprintf("%02d", i)
	}
	selHours := widget.NewSelect(hours, nil)
	selMins := widget.NewSelect(minutes, nil)
	selHours.SetSelected("08")
	selMins.SetSelected("00")

	info := widget.NewLabel("")

	saveBtn := widget.NewButton("Save Entry", func() {
		disp := strings.TrimSpace(userSel.Selected)
		if disp == "" {
			showError(fmt.Errorf("Please select a member"))
			return
		}
		uid := idByDisplay[disp]
		u := findUserByID(uid)
		if u == nil {
			showError(fmt.Errorf("Member not found"))
			return
		}

		// Build midnight time for selected date and add hours/minutes
		y, _ := strconv.Atoi(selYear.Selected)
		mth, _ := strconv.Atoi(selMonth.Selected)
		dy, _ := strconv.Atoi(selDay.Selected)
		dy = clampDay(y, time.Month(mth), dy)
		midnight := time.Date(y, time.Month(mth), dy, 0, 0, 0, 0, time.Local)
		h, _ := strconv.Atoi(selHours.Selected)
		m, _ := strconv.Atoi(selMins.Selected)
		if h == 0 && m == 0 {
			showError(fmt.Errorf("Duration cannot be 0h 0m"))
			return
		}
		timeIn := midnight
		timeOut := midnight.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute)
		timeInFull := timeIn.Format(TimeLayout)
		timeOutFull := timeOut.Format(TimeLayout)

		// Apply entries
		scanLog = append(scanLog,
			LogEntry{UserID: u.ID, Name: u.Name, Time: timeInFull, Status: StatusIn, EntryType: EntryManual},
			LogEntry{UserID: u.ID, Name: u.Name, Time: timeOutFull, Status: StatusOut, EntryType: EntryManual},
		)
		// For manual entry we do not alter live status, leave it as-is
		calculateDuration(u.ID, timeOutFull)
		if !saveData() {
			return
		}
		updateClockedInCount()
		info.SetText("Manual entry saved.")
	})
	saveBtn.Importance = widget.HighImportance

	details := container.NewVBox(
		sectionHeading("Entry details", "Choose the member, date, and total duration."),
		widget.NewForm(
			widget.NewFormItem("Member", userSel),
			widget.NewFormItem("Date", container.NewHBox(selMonth, selDay, selYear)),
			widget.NewFormItem("Hours", selHours),
			widget.NewFormItem("Minutes", selMins),
		),
		saveBtn,
		info,
	)
	guidance := container.NewVBox(
		sectionHeading("Duration-only entry", "Add time without changing live clock status."),
		widget.NewLabel("Use this workspace for meetings, projects, or time that was not recorded with a badge scan."),
		widget.NewSeparator(),
		widget.NewLabel("The selected duration is recorded from midnight on the chosen date for reporting purposes."),
		widget.NewLabel("A manual duration creates matching clock-in and clock-out records and leaves the member's current kiosk status unchanged."),
	)
	for _, object := range guidance.Objects {
		if label, ok := object.(*widget.Label); ok {
			label.Wrapping = fyne.TextWrapWord
		}
	}
	return twoPanelWorkspace("Manual entry", "Record member time that was not captured by the kiosk.", guidance, details)
}

func addUserAction() {
	setMainContent(buildAddUserView())
}

func buildAddUserView() fyne.CanvasObject {
	id := widget.NewEntry()
	name := widget.NewEntry()
	id.SetPlaceHolder("Member ID")
	name.SetPlaceHolder("Member Name")

	addMember := func() {
		uid := strings.TrimSpace(id.Text)
		un := strings.TrimSpace(name.Text)
		if uid == "" || un == "" {
			showError(fmt.Errorf("ID and Name required"))
			return
		}
		if findUserByID(uid) != nil {
			showError(fmt.Errorf("Member with this ID already exists"))
			return
		}
		userDB = append(userDB, User{ID: uid, Name: un, Status: StatusOut})
		if !saveData() {
			return
		}
		updateClockedInCount()
		showInformation("Success", "Member added successfully.")
		reviewAction()
	}
	submit := widget.NewButton("Add Member", addMember)
	submit.Importance = widget.HighImportance
	id.OnSubmitted = func(string) { mainWin.Canvas().Focus(name) }
	name.OnSubmitted = func(string) { addMember() }
	cancel := widget.NewButton("Cancel", func() { reviewAction() })

	details := container.NewVBox(
		sectionHeading("Member details", "Create a unique record for this member."),
		widget.NewForm(
			widget.NewFormItem("Member ID", id),
			widget.NewFormItem("Member Name", name),
		),
		container.NewGridWithColumns(2, cancel, submit),
	)
	guidanceText := widget.NewLabel("Member IDs should match the value encoded on the member's badge. IDs are not case-sensitive, but each ID must be unique.")
	guidanceText.Wrapping = fyne.TextWrapWord
	guidance := container.NewVBox(
		sectionHeading("Member directory", "Add a member before their first kiosk scan."),
		guidanceText,
		widget.NewSeparator(),
		widget.NewLabel("After saving, the member is immediately available in badge scanning and Find a Member."),
	)
	return twoPanelWorkspace("Add member", "Create a new member record for kiosk access.", guidance, details)
}

// ---------- Review / Data Views ----------

// ---------- Main Content Swapping ----------

func setMainContent(c fyne.CanvasObject) {
	if contentHolder == nil {
		return
	}
	// Reset responsive tables registry on view change
	responsiveTables = nil
	contentHolder.RemoveAll()
	contentHolder.Add(container.NewPadded(c))
	contentHolder.Refresh()
}

func showHome() {
	setMainContent(buildHomeView())
	focusScanEntry()
}

func reviewAction() {
	setMainContent(buildReviewView())
}

func buildReviewView() fyne.CanvasObject {
	btnUsers := widget.NewButton("Member Directory", func() { displayUserList() })
	btnLog := widget.NewButton("Scan Log", func() { displayScanLog() })
	btnTime := widget.NewButton("Time Data", func() { showTimeDataView() })
	btnExport := widget.NewButton("Export Scan Log", func() { exportDataCSV() })
	btnSummary := widget.NewButton("Export Time Summary", func() { showTimeSummaryExportView() })
	btnRefresh := widget.NewButton("Refresh Current Statuses", refreshCurrentStatuses)
	btnAnom := widget.NewButton("Anomalies", func() { anomaliesAction() })
	for _, button := range []*widget.Button{btnUsers, btnLog, btnTime, btnExport, btnSummary, btnRefresh, btnAnom} {
		button.Importance = widget.HighImportance
	}
	tools := container.NewGridWithColumns(4,
		actionTile("Members", "Directory and member details", btnUsers),
		actionTile("Scan history", "Review or correct time entries", btnLog),
		actionTile("Hours", "Calculate totals by date range", btnTime),
		actionTile("Raw export", "Save the complete scan log", btnExport),
		actionTile("Hours export", "Save totals by member", btnSummary),
		actionTile("Refresh", "Rebuild current member statuses", btnRefresh),
		actionTile("Exceptions", "Review long or open sessions", btnAnom),
	)
	dataLocation := widget.NewLabel("Data files: " + storageDir)
	dataLocation.Wrapping = fyne.TextWrapWord
	return container.NewBorder(
		container.NewVBox(
			sectionHeading("Coordinator workspace", "Review members, time records, and exports."),
			widget.NewSeparator(),
		),
		dataLocation,
		nil, nil,
		tools,
	)
}

func displayUserList() {
	cols := []string{"member_id", "name"}
	data := make([][]string, 0, len(userDB))
	for _, u := range userDB {
		data = append(data, []string{u.ID, u.Name})
	}
	setMainContent(buildTableView("Member Directory", cols, data, nil, nil))
}

func displayScanLog() {
	cols := []string{"member_id", "name", "time", "status", "entry_type"}

	data := make([][]string, 0, len(scanLog))
	indexMap := make([]int, 0, len(scanLog))
	for i, e := range scanLog {
		row := []string{e.UserID, e.Name, e.Time, e.Status, e.EntryType}
		data = append(data, row)
		indexMap = append(indexMap, i)
	}

	selected := -1
	var table *widget.Table

	// Build filter dropdowns from existing data
	uidSet := map[string]struct{}{}
	nameSet := map[string]struct{}{}
	dateSet := map[string]struct{}{}
	for _, e := range scanLog {
		uidSet[e.UserID] = struct{}{}
		nameSet[e.Name] = struct{}{}
		if len(e.Time) >= 10 {
			dateSet[e.Time[:10]] = struct{}{}
		}
	}
	uidOpts := []string{"All"}
	for k := range uidSet {
		uidOpts = append(uidOpts, k)
	}
	sort.Strings(uidOpts)
	nameOpts := []string{"All"}
	for k := range nameSet {
		nameOpts = append(nameOpts, k)
	}
	sort.Strings(nameOpts)
	dateOpts := []string{"All"}
	for k := range dateSet {
		dateOpts = append(dateOpts, k)
	}
	sort.Strings(dateOpts)

	uidSel := widget.NewSelect(uidOpts, nil)
	nameSel := widget.NewSelect(nameOpts, nil)
	dateSel := widget.NewSelect(dateOpts, nil)
	uidSel.SetSelected("All")
	nameSel.SetSelected("All")
	dateSel.SetSelected("All")

	// Sorting controls (via dropdowns)
	sortFields := []string{"Member ID", "Name", "Date"}
	sortField := widget.NewSelect(sortFields, nil)
	sortField.SetSelected("Date")
	sortOrder := widget.NewSelect([]string{"Ascending", "Descending"}, nil)
	sortOrder.SetSelected("Descending")

	// Holder to swap table on filter/sort
	tableHolder := container.NewMax()

	rebuild := func() {
		// Build filtered rows
		baseData := make([][]string, 0)
		baseIdx := make([]int, 0)
		for i, e := range scanLog {
			if uidSel.Selected != "All" && e.UserID != uidSel.Selected {
				continue
			}
			if nameSel.Selected != "All" && e.Name != nameSel.Selected {
				continue
			}
			if dateSel.Selected != "All" {
				if len(e.Time) < 10 || e.Time[:10] != dateSel.Selected {
					continue
				}
			}
			baseData = append(baseData, []string{e.UserID, e.Name, e.Time, e.Status, e.EntryType})
			baseIdx = append(baseIdx, i)
		}
		// Sort
		col := 2
		switch sortField.Selected {
		case "Member ID":
			col = 0
		case "Name":
			col = 1
		default:
			col = 2
		}
		asc := sortOrder.Selected == "Ascending"
		n := len(baseData)
		ord := make([]int, n)
		for i := 0; i < n; i++ {
			ord[i] = i
		}
		sort.SliceStable(ord, func(i, j int) bool { return typeAwareLess(cols, baseData, col, ord[i], ord[j], asc) })
		data = make([][]string, n)
		indexMap = make([]int, n)
		for i := 0; i < n; i++ {
			data[i] = baseData[ord[i]]
			indexMap[i] = baseIdx[ord[i]]
		}

		selected = -1
		table = newTableWithData(cols, &data, func(row, col int) { selected = row }, nil, &selected)
		adjustTableColumnWidths(table, cols, data)
		tableHolder.RemoveAll()
		tableHolder.Add(table)
		tableHolder.Refresh()
	}

	// Initial table
	rebuild()
	registerResponsiveTable(table, cols, func() [][]string { return data })

	buttons := []fyne.CanvasObject{}
	// Delete button
	buttons = append(buttons, widget.NewButton("Delete Entry", func() {
		if selected < 1 {
			showInformation("Select an Entry", "Select a scan-log entry before deleting it.")
			return
		}
		idx := indexMap[selected-1]
		showConfirm("Confirm Delete", "Delete the selected entry?", func(ok bool) {
			if !ok {
				return
			}
			if idx >= 0 && idx < len(scanLog) {
				scanLog = append(scanLog[:idx], scanLog[idx+1:]...)
				if !saveData() {
					return
				}
			}
			displayScanLog()
		})
	}))

	// Modify button
	buttons = append(buttons, widget.NewButton("Modify Entry", func() {
		if selected < 1 {
			showInformation("Select an Entry", "Select a scan-log entry before modifying it.")
			return
		}
		idx := indexMap[selected-1]
		if idx < 0 || idx >= len(scanLog) {
			return
		}
		showModifyScanLogEntry(idx)
	}))

	// Controls
	filters := container.NewHBox(
		widget.NewLabel("Member ID:"), uidSel,
		widget.NewLabel("Name:"), nameSel,
		widget.NewLabel("Date:"), dateSel,
		widget.NewButton("Apply Filters", func() { rebuild() }),
	)
	sortControls := container.NewHBox(
		widget.NewLabel("Sort by:"), sortField,
		widget.NewLabel("Order:"), sortOrder,
		widget.NewButton("Apply Sort", func() { rebuild() }),
	)
	header := container.NewVBox(widget.NewLabelWithStyle("Scan Log", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), filters, sortControls)

	footer := container.NewHBox(buttons...)
	setMainContent(container.NewBorder(header, footer, nil, nil, tableHolder))
}

func showTimeDataView() {

	months := make([]string, 12)
	for i := 1; i <= 12; i++ {
		months[i-1] = fmt.Sprintf("%02d", i)
	}
	days := make([]string, 31)
	for i := 1; i <= 31; i++ {
		days[i-1] = fmt.Sprintf("%02d", i)
	}
	years := make([]string, 0, 52)
	for y := 2000; y <= 2051; y++ {
		years = append(years, fmt.Sprintf("%d", y))
	}

	now := time.Now()
	startMonth := widget.NewSelect(months, nil)
	startDay := widget.NewSelect(days, nil)
	startYear := widget.NewSelect(years, nil)
	endMonth := widget.NewSelect(months, nil)
	endDay := widget.NewSelect(days, nil)
	endYear := widget.NewSelect(years, nil)

	startMonth.SetSelected("01")
	startDay.SetSelected("01")
	startYear.SetSelected(fmt.Sprintf("%d", now.Year()))
	endMonth.SetSelected(now.Format("01"))
	endDay.SetSelected(now.Format("02"))
	endYear.SetSelected(now.Format("2006"))

	setRange := func(start, end time.Time) {
		startMonth.SetSelected(start.Format("01"))
		startDay.SetSelected(start.Format("02"))
		startYear.SetSelected(start.Format("2006"))
		endMonth.SetSelected(end.Format("01"))
		endDay.SetSelected(end.Format("02"))
		endYear.SetSelected(end.Format("2006"))
	}
	btnToday := widget.NewButton("Today", func() { setRange(now, now) })
	btnWeek := widget.NewButton("This Week", func() {
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		setRange(now.AddDate(0, 0, -(weekday-1)), now)
	})
	btnMonth := widget.NewButton("This Month", func() {
		setRange(time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local), now)
	})
	for _, button := range []*widget.Button{btnToday, btnWeek, btnMonth} {
		button.Importance = widget.HighImportance
	}

	btnCalc := widget.NewButton("Calculate", func() {
		sy, _ := strconv.Atoi(startYear.Selected)
		sm, _ := strconv.Atoi(startMonth.Selected)
		sd, _ := strconv.Atoi(startDay.Selected)
		ey, _ := strconv.Atoi(endYear.Selected)
		em, _ := strconv.Atoi(endMonth.Selected)
		ed, _ := strconv.Atoi(endDay.Selected)

		sd = clampDay(sy, time.Month(sm), sd)
		ed = clampDay(ey, time.Month(em), ed)

		start := time.Date(sy, time.Month(sm), sd, 0, 0, 0, 0, time.Local)
		end := time.Date(ey, time.Month(em), ed, 23, 59, 59, 0, time.Local)

		userTotals, total := computeTimeDataRange(start, end)

		cols := []string{"member_id", "name", "time_worked"}
		data := make([][]string, 0, len(userTotals))
		for uid, dur := range userTotals {
			u := findUserByID(uid)
			name := ""
			if u != nil {
				name = u.Name
			}
			data = append(data, []string{uid, name, formatDurationHours(dur)})
		}
		sort.Slice(data, func(i, j int) bool { return data[i][0] < data[j][0] })
		displayTableWithTotal("Time Data", cols, data, total)
	})
	btnCalc.Importance = widget.HighImportance

	startRow := container.NewHBox(widget.NewLabel("Start:"), startMonth, startDay, startYear)
	endRow := container.NewHBox(widget.NewLabel("End:"), endMonth, endDay, endYear)
	quickRanges := container.NewVBox(
		sectionHeading("Quick ranges", "Start with a commonly used reporting period."),
		btnToday,
		btnWeek,
		btnMonth,
	)
	dateRange := container.NewVBox(
		sectionHeading("Custom range", "Choose inclusive start and end dates."),
		startRow,
		endRow,
		widget.NewSeparator(),
		btnCalc,
	)
	setMainContent(twoPanelWorkspace("Time data", "Calculate and review member hours for any period.", quickRanges, dateRange))
}

// Full-window modify form for scan log entries
func showModifyScanLogEntry(idx int) {
	if idx < 0 || idx >= len(scanLog) {
		return
	}
	entry := scanLog[idx]

	newTime := widget.NewEntry()
	newTime.SetText(entry.Time)
	statusSel := widget.NewSelect([]string{StatusIn, StatusOut}, nil)
	statusSel.SetSelected(entry.Status)

	submit := widget.NewButton("Submit", func() {
		if _, err := parseLogTime(newTime.Text); err != nil {
			showError(fmt.Errorf("Invalid time format"))
			return
		}
		ns := strings.TrimSpace(statusSel.Selected)
		if ns != StatusIn && ns != StatusOut {
			showError(fmt.Errorf("Invalid status"))
			return
		}
		scanLog[idx].Time = newTime.Text
		scanLog[idx].Status = ns
		if !saveData() {
			return
		}
		displayScanLog()
	})
	submit.Importance = widget.HighImportance
	cancel := widget.NewButton("Cancel", func() { displayScanLog() })

	form := widget.NewForm(
		widget.NewFormItem("New time (YYYY-MM-DD HH:MM:SS)", newTime),
		widget.NewFormItem("Status", statusSel),
	)

	details := container.NewVBox(
		sectionHeading("Entry details", "Update the timestamp or clock status."),
		form,
		container.NewGridWithColumns(2, cancel, submit),
	)
	explanation := widget.NewLabel("Changes affect time totals and may affect the member's current status. Use Refresh Current Statuses afterward if this is the member's most recent record.")
	explanation.Wrapping = fyne.TextWrapWord
	guidance := container.NewVBox(
		sectionHeading("Administrative correction", "Correct an individual scan-log record."),
		explanation,
	)
	setMainContent(twoPanelWorkspace("Modify scan-log entry", "Review the record carefully before saving.", guidance, details))
}

// Robust, ordered, overlap-aware calculation
func computeTimeDataRange(rangeStart, rangeEnd time.Time) (map[string]time.Duration, time.Duration) {
	entries := make([]LogEntry, len(scanLog))
	copy(entries, scanLog)
	sort.Slice(entries, func(i, j int) bool {
		ti, _ := parseLogTime(entries[i].Time)
		tj, _ := parseLogTime(entries[j].Time)
		return ti.Before(tj)
	})

	type state struct {
		in      bool
		clockIn time.Time
	}

	states := map[string]*state{}
	totals := map[string]time.Duration{}

	for _, e := range entries {
		t, err := parseLogTime(e.Time)
		if err != nil {
			continue
		}
		s := states[e.UserID]
		if s == nil {
			s = &state{}
			states[e.UserID] = s
		}

		switch e.Status {
		case StatusIn:
			if !s.in {
				s.in = true
				s.clockIn = t
			}
		case StatusOut:
			if s.in {
				start := s.clockIn
				end := t
				if end.After(rangeStart) && start.Before(rangeEnd) {
					overlapStart := maxTime(start, rangeStart)
					overlapEnd := minTime(end, rangeEnd)
					if overlapEnd.After(overlapStart) {
						totals[e.UserID] += overlapEnd.Sub(overlapStart)
					}
				}
				s.in = false
			}
		}
	}

	for uid, s := range states {
		if s.in {
			start := s.clockIn
			end := minTime(rangeEnd, time.Now())
			if end.After(rangeStart) && start.Before(rangeEnd) {
				overlapStart := maxTime(start, rangeStart)
				overlapEnd := minTime(end, rangeEnd)
				if overlapEnd.After(overlapStart) {
					totals[uid] += overlapEnd.Sub(overlapStart)
				}
			}
		}
	}

	var grand time.Duration
	for _, d := range totals {
		grand += d
	}
	return totals, grand
}

func formatDurationHours(d time.Duration) string {
	h := d.Hours()
	return fmt.Sprintf("%.2f hours", h)
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// clampDay ensures the day is valid for the given month/year
func clampDay(year int, month time.Month, day int) int {
	last := time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local).Day()
	if day < 1 {
		return 1
	}
	if day > last {
		return last
	}
	return day
}

// adjustTableColumnWidths sets reasonable widths with a preference for widening the "name" column.
func adjustTableColumnWidths(table *widget.Table, cols []string, rows [][]string) {
	if table == nil || len(cols) == 0 || mainWin == nil || mainWin.Canvas() == nil {
		return
	}
	cw := mainWin.Canvas().Size().Width
	if cw <= 0 {
		cw = 900
	}
	// Dynamic per-column width based on longest content
	minW := float32(100)
	maxW := float32(cw) * 0.7
	charW := float32(9) // rough average character width in px

	longest := func(col int) int {
		max := len(columnHeading(cols[col]))
		for _, r := range rows {
			if col < len(r) && len(r[col]) > max {
				max = len(r[col])
			}
		}
		return max
	}

	for i, c := range cols {
		l := longest(i)
		w := float32(l)*charW + 24 // padding
		if w < minW {
			w = minW
		}
		if w > maxW {
			w = maxW
		}
		lc := strings.ToLower(c)
		if lc == "user_id" || lc == "member_id" || lc == "status" || lc == "entry_type" {
			if w > 220 {
				w = 220
			}
		}
		table.SetColumnWidth(i, w)
	}
}

func columnHeading(value string) string {
	parts := strings.Split(value, "_")
	for i, part := range parts {
		if strings.EqualFold(part, "id") {
			parts[i] = "ID"
			continue
		}
		if part != "" {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}

func displayCellValue(column, value string) string {
	if column == "status" {
		switch value {
		case StatusIn:
			return "Clocked In"
		case StatusOut:
			return "Clocked Out"
		}
	}
	return value
}

// registerResponsiveTable tracks a table so we can recompute widths when the window resizes.
func registerResponsiveTable(table *widget.Table, cols []string, rowsProvider func() [][]string) {
	responsiveTables = append(responsiveTables, struct {
		tbl          *widget.Table
		cols         []string
		rowsProvider func() [][]string
	}{table, cols, rowsProvider})
}

func adjustAllTables() {
	for _, rt := range responsiveTables {
		if rt.tbl != nil {
			adjustTableColumnWidths(rt.tbl, rt.cols, rt.rowsProvider())
		}
	}
}

// typeAwareSort sorts rows in place based on column header semantics.
func typeAwareSort(cols []string, rows *[][]string, col int, asc bool) {
	r := *rows
	order := make([]int, len(r))
	for i := range r {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool { return typeAwareLess(cols, r, col, order[i], order[j], asc) })
	nr := make([][]string, len(r))
	for i := range nr {
		nr[i] = r[order[i]]
	}
	*rows = nr
}

// typeAwareLess compares two row indices considering numeric/time columns.
func typeAwareLess(cols []string, rows [][]string, col, i, j int, asc bool) bool {
	hdr := strings.ToLower(cols[col])
	a := rows[i][col]
	b := rows[j][col]
	switch hdr {
	case "time":
		ta, ea := parseLogTime(a)
		tb, eb := parseLogTime(b)
		if ea == nil && eb == nil {
			if asc {
				return ta.Before(tb)
			}
			return tb.Before(ta)
		}
	case "time_worked":
		// format "X.XX hours"
		af := parseHoursFloat(a)
		bf := parseHoursFloat(b)
		if asc {
			return af < bf
		}
		return bf < af
	default:
		// Try integer compare first
		ai, ea := strconv.Atoi(a)
		bi, eb := strconv.Atoi(b)
		if ea == nil && eb == nil {
			if asc {
				return ai < bi
			}
			return bi < ai
		}
	}
	// Fallback to case-insensitive string compare
	if asc {
		return strings.ToLower(a) < strings.ToLower(b)
	}
	return strings.ToLower(a) > strings.ToLower(b)
}

func parseHoursFloat(s string) float64 {
	f := strings.Fields(s)
	if len(f) == 0 {
		return 0
	}
	v, err := strconv.ParseFloat(f[0], 64)
	if err != nil {
		return 0
	}
	return v
}

// ---------- Export / Merge ----------

func refreshCurrentStatuses() {
	entries := append([]LogEntry(nil), scanLog...)
	sort.SliceStable(entries, func(i, j int) bool {
		left, leftErr := parseLogTime(entries[i].Time)
		right, rightErr := parseLogTime(entries[j].Time)
		if leftErr != nil || rightErr != nil {
			return entries[i].Time < entries[j].Time
		}
		return left.Before(right)
	})

	latestStatus := make(map[string]string, len(userDB))
	for _, entry := range entries {
		if entry.Status == StatusIn || entry.Status == StatusOut {
			latestStatus[entry.UserID] = entry.Status
		}
	}
	for i := range userDB {
		userDB[i].Status = latestStatus[userDB[i].ID]
		if userDB[i].Status == "" {
			userDB[i].Status = StatusOut
		}
	}
	if !saveData() {
		return
	}
	updateClockedInCount()
	showInformation("Statuses Refreshed", "Current clock-in statuses now match the most recent scan-log entries.")
}

func showTimeSummaryExportView() {
	now := time.Now()
	months := make([]string, 12)
	for month := 1; month <= 12; month++ {
		months[month-1] = fmt.Sprintf("%02d", month)
	}
	days := make([]string, 31)
	for day := 1; day <= 31; day++ {
		days[day-1] = fmt.Sprintf("%02d", day)
	}
	years := make([]string, 0, 52)
	for year := 2000; year <= 2051; year++ {
		years = append(years, strconv.Itoa(year))
	}

	startMonth, startDay, startYear := widget.NewSelect(months, nil), widget.NewSelect(days, nil), widget.NewSelect(years, nil)
	endMonth, endDay, endYear := widget.NewSelect(months, nil), widget.NewSelect(days, nil), widget.NewSelect(years, nil)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	startMonth.SetSelected(monthStart.Format("01"))
	startDay.SetSelected(monthStart.Format("02"))
	startYear.SetSelected(monthStart.Format("2006"))
	endMonth.SetSelected(now.Format("01"))
	endDay.SetSelected(now.Format("02"))
	endYear.SetSelected(now.Format("2006"))

	exportBtn := widget.NewButton("Choose File and Export", func() {
		sy, _ := strconv.Atoi(startYear.Selected)
		sm, _ := strconv.Atoi(startMonth.Selected)
		sd, _ := strconv.Atoi(startDay.Selected)
		ey, _ := strconv.Atoi(endYear.Selected)
		em, _ := strconv.Atoi(endMonth.Selected)
		ed, _ := strconv.Atoi(endDay.Selected)
		start := time.Date(sy, time.Month(sm), clampDay(sy, time.Month(sm), sd), 0, 0, 0, 0, time.Local)
		end := time.Date(ey, time.Month(em), clampDay(ey, time.Month(em), ed), 23, 59, 59, 0, time.Local)
		if end.Before(start) {
			showError(fmt.Errorf("End date must not be before start date"))
			return
		}
		exportTimeSummaryCSV(start, end)
	})
	exportBtn.Importance = widget.HighImportance

	details := container.NewVBox(
		sectionHeading("Export range", "Choose the dates included in the summary."),
		container.NewHBox(widget.NewLabel("Start:"), startMonth, startDay, startYear),
		container.NewHBox(widget.NewLabel("End:"), endMonth, endDay, endYear),
		widget.NewSeparator(),
		exportBtn,
	)
	explanation := widget.NewLabel("The exported CSV contains one row per member, including Member ID, name, total hours, and the selected reporting dates.")
	explanation.Wrapping = fyne.TextWrapWord
	guidance := container.NewVBox(
		sectionHeading("Member hours summary", "Create a clean report for coordinators."),
		explanation,
		widget.NewSeparator(),
		widget.NewLabel("The current month is selected by default."),
	)
	setMainContent(twoPanelWorkspace("Export time summary", "Create a per-member hours report for a selected period.", guidance, details))
}

func exportTimeSummaryCSV(start, end time.Time) {
	save := dialog.NewFileSave(func(uc fyne.URIWriteCloser, err error) {
		if err != nil || uc == nil {
			return
		}
		defer uc.Close()
		totals, _ := computeTimeDataRange(start, end)
		writer := csv.NewWriter(uc)
		defer writer.Flush()
		_ = writer.Write([]string{"Member ID", "Name", "Hours", "Start Date", "End Date"})
		ids := make([]string, 0, len(totals))
		for id := range totals {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			name := ""
			if user := findUserByID(id); user != nil {
				name = user.Name
			}
			_ = writer.Write([]string{id, name, fmt.Sprintf("%.2f", totals[id].Hours()), start.Format(DateLayout), end.Format(DateLayout)})
		}
	}, mainWin)
	save.SetFileName(fmt.Sprintf("time_summary_%s_to_%s.csv", start.Format(DateLayout), end.Format(DateLayout)))
	save.Show()
}

func exportDataCSV() {
	save := dialog.NewFileSave(func(uc fyne.URIWriteCloser, err error) {
		if err != nil || uc == nil {
			return
		}
		defer uc.Close()
		w := csv.NewWriter(uc)
		defer w.Flush()
		_ = w.Write([]string{"Member ID", "Name", "Time", "Status", "Entry Type"})
		for _, e := range scanLog {
			_ = w.Write([]string{e.UserID, e.Name, e.Time, e.Status, e.EntryType})
		}
	}, mainWin)
	save.SetFileName("scan_log_export.csv")
	save.Show()
}

func mergeExternal() {
	open := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil || r == nil {
			return
		}
		defer r.Close()
		b, _ := io.ReadAll(r)
		var ext []LogEntry
		if err := json.Unmarshal(b, &ext); err != nil {
			showError(fmt.Errorf("Invalid JSON log"))
			return
		}
		scanLog = append(scanLog, ext...)
		if !saveData() {
			return
		}
		showInformation("Merge Successful", "External log has been merged into the main scan log.")
	}, mainWin)
	open.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	open.Show()
}

// ---------- Anomalies (sessions > 8h or open > 8h) ----------

type Anomaly struct {
	UserID   string
	Name     string
	Start    string
	End      string // empty for open
	Duration time.Duration
	Open     bool
	InIndex  int
	OutIndex int // -1 if open
}

func anomalyKey(a Anomaly) string { return a.UserID + "|" + a.Start }

func detectAnomalies(threshold time.Duration) []Anomaly {
	// Work on a time-ordered copy while tracking original indices
	type item struct {
		idx int
		e   LogEntry
	}
	items := make([]item, len(scanLog))
	for i := range scanLog {
		items[i] = item{i, scanLog[i]}
	}
	sort.Slice(items, func(i, j int) bool {
		ti, _ := parseLogTime(items[i].e.Time)
		tj, _ := parseLogTime(items[j].e.Time)
		return ti.Before(tj)
	})

	type state struct {
		in      bool
		clockIn time.Time
		inIdx   int
	}
	states := map[string]*state{}
	var res []Anomaly

	for _, it := range items {
		e := it.e
		t, err := parseLogTime(e.Time)
		if err != nil {
			continue
		}
		s := states[e.UserID]
		if s == nil {
			s = &state{}
			states[e.UserID] = s
		}
		switch e.Status {
		case StatusIn:
			if s.in {
				// Open session lingering; check if exceeds threshold
				if d := t.Sub(s.clockIn); d > threshold {
					a := Anomaly{UserID: e.UserID, Name: e.Name, Start: s.clockIn.Format(TimeLayout), End: "", Duration: d, Open: true, InIndex: s.inIdx, OutIndex: -1}
					if !anomalyApprovals[anomalyKey(a)] {
						res = append(res, a)
					}
				}
			}
			s.in = true
			s.clockIn = t
			s.inIdx = it.idx
		case StatusOut:
			if s.in {
				if d := t.Sub(s.clockIn); d > threshold {
					a := Anomaly{UserID: e.UserID, Name: e.Name, Start: s.clockIn.Format(TimeLayout), End: t.Format(TimeLayout), Duration: d, Open: false, InIndex: s.inIdx, OutIndex: it.idx}
					if !anomalyApprovals[anomalyKey(a)] {
						res = append(res, a)
					}
				}
				s.in = false
			}
		}
	}
	// Remaining open sessions
	now := time.Now()
	for uid, s := range states {
		if s.in {
			if d := now.Sub(s.clockIn); d > threshold {
				// Need name
				name := ""
				if u := findUserByID(uid); u != nil {
					name = u.Name
				}
				a := Anomaly{UserID: uid, Name: name, Start: s.clockIn.Format(TimeLayout), End: "", Duration: d, Open: true, InIndex: s.inIdx, OutIndex: -1}
				if !anomalyApprovals[anomalyKey(a)] {
					res = append(res, a)
				}
			}
		}
	}
	return res
}

func anomaliesAction() {
	anomalies := detectAnomalies(8 * time.Hour)
	cols := []string{"member_id", "name", "start_time", "end_time", "duration", "type"}
	rows := make([][]string, 0, len(anomalies))
	for _, a := range anomalies {
		typ := "completed"
		if a.Open {
			typ = "open"
		}
		end := a.End
		if end == "" {
			end = "(open)"
		}
		rows = append(rows, []string{a.UserID, a.Name, a.Start, end, formatDurationHours(a.Duration), typ})
	}

	selected := -1
	var table *widget.Table
	table = newTableWithData(cols, &rows, func(row, col int) { selected = row }, nil, &selected)
	adjustTableColumnWidths(table, cols, rows)

	btnApprove := widget.NewButton("Approve", func() {
		if selected < 1 {
			showInformation("Select an Anomaly", "Select an anomaly before approving it.")
			return
		}
		a := anomalies[selected-1]
		if anomalyApprovals == nil {
			anomalyApprovals = map[string]bool{}
		}
		anomalyApprovals[anomalyKey(a)] = true
		if !saveData() {
			return
		}
		anomaliesAction()
	})

	btnDisapprove := widget.NewButton("Disapprove / Fix", func() {
		if selected < 1 {
			showInformation("Select an Anomaly", "Select an anomaly before fixing it.")
			return
		}
		a := anomalies[selected-1]
		showFixAnomaly(a)
	})

	footer := container.NewHBox(btnApprove, btnDisapprove)
	setMainContent(container.NewBorder(widget.NewLabelWithStyle("Anomalies (> 8h)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), footer, nil, nil, table))
}

func showFixAnomaly(a Anomaly) {
	// Replace explicit clock-out time with Hours/Minutes worked inputs.
	// Out time will be Start + (hours:minutes).
	date := a.Start[:10]

	// Parse start time
	startTime, err := parseLogTime(a.Start)
	if err != nil {
		showError(fmt.Errorf("Invalid start time in entry"))
		return
	}

	// Hours / Minutes dropdowns (like Manual Entry)
	hours := make([]string, 24)
	for i := 0; i < 24; i++ {
		hours[i] = fmt.Sprintf("%02d", i)
	}
	minutes := make([]string, 60)
	for i := 0; i < 60; i++ {
		minutes[i] = fmt.Sprintf("%02d", i)
	}
	selHours := widget.NewSelect(hours, nil)
	selMins := widget.NewSelect(minutes, nil)
	// Preselect 08:00 as a helpful default
	selHours.SetSelected("08")
	selMins.SetSelected("00")

	info := widget.NewLabel("")

	submit := widget.NewButton("Save", func() {
		h, _ := strconv.Atoi(selHours.Selected)
		m, _ := strconv.Atoi(selMins.Selected)
		if h == 0 && m == 0 {
			showError(fmt.Errorf("Duration cannot be 0h 0m"))
			return
		}
		outTime := startTime.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute)
		outFull := outTime.Format(TimeLayout)

		if a.Open {
			// Append a new OUT entry
			u := findUserByID(a.UserID)
			name := ""
			if u != nil {
				name = u.Name
			}
			scanLog = append(scanLog, LogEntry{UserID: a.UserID, Name: name, Time: outFull, Status: StatusOut, EntryType: "Adjustment"})
		} else {
			// Modify existing OUT entry time
			if a.OutIndex >= 0 && a.OutIndex < len(scanLog) {
				scanLog[a.OutIndex].Time = outFull
				scanLog[a.OutIndex].EntryType = "Adjustment"
			}
		}
		if !saveData() {
			return
		}
		updateClockedInCount()
		info.SetText("Saved. Rebuilding list...")
		anomaliesAction()
	})
	submit.Importance = widget.HighImportance
	cancel := widget.NewButton("Cancel", func() { anomaliesAction() })

	form := widget.NewForm(
		widget.NewFormItem("Member", widget.NewLabel(fmt.Sprintf("%s - %s", a.UserID, a.Name))),
		widget.NewFormItem("Date", widget.NewLabel(date)),
		widget.NewFormItem("Hours", selHours),
		widget.NewFormItem("Minutes", selMins),
	)
	details := container.NewVBox(
		sectionHeading("Corrected duration", "Enter the member's actual time worked."),
		form,
		container.NewGridWithColumns(2, cancel, submit),
		info,
	)
	explanation := widget.NewLabel("This creates or updates the matching clock-out record while preserving the original clock-in time.")
	explanation.Wrapping = fyne.TextWrapWord
	guidance := container.NewVBox(
		sectionHeading("Resolve exception", "Review a long or open session."),
		explanation,
	)
	setMainContent(twoPanelWorkspace("Fix anomaly", "Correct the duration before returning to the exception list.", guidance, details))
}

// ---------- Clocked-In Users ----------

func viewClockedInUsers() { // now transitions into main window
	var list [][]string
	for _, u := range userDB {
		if u.Status == StatusIn {
			list = append(list, []string{u.ID, u.Name})
		}
	}
	if len(list) == 0 {
		showInformation("No Members Clocked In", "There are no members currently clocked in.")
		return
	}

	cols := []string{"member_id", "name"}
	selectedRow := -1

	var table *widget.Table
	table = newTableWithData(cols, &list, func(row, col int) { selectedRow = row }, nil, &selectedRow)
	adjustTableColumnWidths(table, cols, list)

	btnClockOut := widget.NewButton("Clock Out Selected Member", func() {
		if selectedRow < 1 {
			showError(fmt.Errorf("No member selected"))
			return
		}
		uid := list[selectedRow-1][0]
		u := findUserByID(uid)
		if u == nil {
			showError(fmt.Errorf("Member not found"))
			return
		}
		current := nowString()
		u.Status = StatusOut
		calculateDuration(uid, current)
		scanLog = append(scanLog, LogEntry{
			UserID:    u.ID,
			Name:      u.Name,
			Time:      current,
			Status:    StatusOut,
			EntryType: EntryManual,
		})
		if !saveData() {
			return
		}
		updateClockedInCount()
		list = removeRow(list, selectedRow-1)
		selectedRow = -1
		table.Refresh()
		showInformation("Success", fmt.Sprintf("%s has been clocked out.", u.Name))
	})

	// Place the table in center, and actions in bottom to use full vertical space
	setMainContent(container.NewBorder(nil, container.NewHBox(btnClockOut), nil, nil, table))
}

func removeRow(s [][]string, i int) [][]string {
	return append(s[:i], s[i+1:]...)
}

// ---------- Mass Clock-Out ----------

func clockOutAllUsers() {
	showConfirm("Confirm Clock Out", "Are you sure you want to clock out all members?", func(ok bool) {
		if !ok {
			return
		}
		current := nowString()
		for i := range userDB {
			if userDB[i].Status == StatusIn {
				userDB[i].Status = StatusOut
				scanLog = append(scanLog, LogEntry{
					UserID:    userDB[i].ID,
					Name:      userDB[i].Name,
					Time:      current,
					Status:    StatusOut,
					EntryType: EntryManual,
				})
			}
		}
		if !saveData() {
			return
		}
		updateClockedInCount()
		showInformation("All Members Clocked Out", "All members have been clocked out.")
	})
}

// ---------- Generic Table Display ----------

func displayTableWithTotal(title string, cols []string, rows [][]string, total time.Duration) {
	lbl := widget.NewLabel("Total Time Worked: " + formatDurationHours(total))
	table := newTableWithData(cols, &rows, nil, nil, nil)
	adjustTableColumnWidths(table, cols, rows)
	setMainContent(container.NewBorder(container.NewVBox(widget.NewLabel(title), lbl), nil, nil, nil, table))
}

func buildTableView(title string, cols []string, rows [][]string, onDelete func(selectedRow int), onModify func(selectedRow int)) fyne.CanvasObject {
	selected := -1
	var table *widget.Table

	onSort := func(col int, asc bool) {
		if len(rows) == 0 {
			return
		}
		typeAwareSort(cols, &rows, col, asc)
		table.Refresh()
	}

	table = newTableWithData(cols, &rows, func(row, col int) { selected = row }, onSort, &selected)
	adjustTableColumnWidths(table, cols, rows)

	buttons := []fyne.CanvasObject{}
	if onDelete != nil {
		buttons = append(buttons, widget.NewButton("Delete Entry", func() { onDelete(selected) }))
	}
	if onModify != nil {
		buttons = append(buttons, widget.NewButton("Modify Entry", func() { onModify(selected) }))
	}
	footer := container.NewHBox(buttons...)

	return container.NewBorder(widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), footer, nil, nil, table)
}

// newTableWithData builds a table with a clickable header row that sorts columns.
func newTableWithData(cols []string, rows *[][]string, onSelect func(row, col int), onSort func(col int, asc bool), selectedRow *int) *widget.Table {
	sortCol := -1
	sortAsc := true

	var table *widget.Table
	table = widget.NewTable(
		func() (int, int) {
			return len(*rows), len(cols)
		},
		func() fyne.CanvasObject {
			background := canvas.NewRectangle(surfaceColor)
			label := widget.NewLabel("")
			label.Truncation = fyne.TextTruncateEllipsis
			return container.NewMax(background, label)
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			if i.Row >= 0 && i.Row < len(*rows) && i.Col >= 0 && i.Col < len((*rows)[i.Row]) {
				cell := o.(*fyne.Container)
				background := cell.Objects[0].(*canvas.Rectangle)
				label := cell.Objects[1].(*widget.Label)
				rowColor := surfaceColor
				if i.Row%2 == 1 {
					rowColor = color.NRGBA{R: 248, G: 250, B: 252, A: 255}
				}
				if selectedRow != nil && *selectedRow == i.Row+1 {
					rowColor = color.NRGBA{R: 214, G: 228, B: 255, A: 255}
				}
				background.FillColor = rowColor
				background.Refresh()
				label.SetText(displayCellValue(cols[i.Col], (*rows)[i.Row][i.Col]))
			}
		},
	)
	table.ShowHeaderRow = true
	if onSort == nil {
		table.CreateHeader = func() fyne.CanvasObject {
			return widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		}
		table.UpdateHeader = func(id widget.TableCellID, object fyne.CanvasObject) {
			if id.Row == -1 && id.Col >= 0 && id.Col < len(cols) {
				object.(*widget.Label).SetText(columnHeading(cols[id.Col]))
			}
		}
	} else {
		table.CreateHeader = func() fyne.CanvasObject { return widget.NewButton("", nil) }
		table.UpdateHeader = func(id widget.TableCellID, object fyne.CanvasObject) {
			if id.Row != -1 || id.Col < 0 || id.Col >= len(cols) {
				return
			}
			column := id.Col
			label := columnHeading(cols[column])
			if sortCol == column {
				if sortAsc {
					label += " ▲"
				} else {
					label += " ▼"
				}
			}
			button := object.(*widget.Button)
			button.SetText(label)
			button.OnTapped = func() {
				if selectedRow != nil {
					*selectedRow = -1
				}
				table.UnselectAll()
				if sortCol == column {
					sortAsc = !sortAsc
				} else {
					sortCol = column
					sortAsc = true
				}
				onSort(column, sortAsc)
				table.Refresh()
			}
		}
	}
	table.OnSelected = func(id widget.TableCellID) {
		row := id.Row + 1
		if selectedRow != nil {
			*selectedRow = row
		}
		if onSelect != nil {
			onSelect(row, id.Col)
		}
		table.UnselectAll()
		table.Refresh()
	}
	return table
}

// ---------- Main UI ----------

func main() {
	appRef = app.NewWithID("OpenChronoLog")
	appRef.Settings().SetTheme(workplaceTheme{base: theme.LightTheme()})
	mainWin = appRef.NewWindow("OpenChronoLog")
	mainWin.Resize(fyne.NewSize(900, 620))
	mainWin.SetFullScreen(true)
	mainWin.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyF11}, func(fyne.Shortcut) {
		mainWin.SetFullScreen(!mainWin.FullScreen())
	})
	mainWin.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyEscape}, func(fyne.Shortcut) {
		mainWin.SetFullScreen(false)
	})
	mainWin.CenterOnScreen()
	mainWin.SetMaster()

	loadData()

	// Try to set an application icon if an icon file is present alongside the binary
	trySetAppIcon()

	idEntry = widget.NewEntry()
	idEntry.SetPlaceHolder("Scan badge or enter Member ID")
	idEntry.OnSubmitted = func(s string) { submitAction() }

	successLb = widget.NewLabel("")
	successLb.Alignment = fyne.TextAlignCenter
	successLb.TextStyle = fyne.TextStyle{Bold: true}
	successLb.Hide()

	clockLbl = widget.NewLabel("")
	clockLbl.Alignment = fyne.TextAlignCenter
	clockLbl.TextStyle = fyne.TextStyle{Bold: true}
	updateClockedInCount()
	// Center the clocked-in user count label
	clockBar = container.NewCenter(clockLbl)

	btnHome := widget.NewButton("Time Clock", showHome)
	btnManual := widget.NewButton("Manual Entry", manualEntryAction)
	btnAddUser := widget.NewButton("Add Member", addUserAction)
	btnReview := widget.NewButton("Coordinator", reviewAction)
	for _, button := range []*widget.Button{btnHome, btnManual, btnAddUser, btnReview} {
		button.Importance = widget.HighImportance
	}
	bottomBar := container.NewGridWithColumns(4, btnHome, btnManual, btnAddUser, btnReview)

	closeBtn := widget.NewButton("Close", func() { handleCloseRequest() })
	topBar := buildAppHeader(closeBtn)

	// Main content holder swaps between views
	contentHolder = container.NewMax()
	// Initialize with the Home view
	showHome()

	root := container.NewBorder(topBar, bottomBar, nil, nil, contentHolder)
	mainWin.SetContent(root)

	// Poll window size periodically; adjust table widths on change
	go func() {
		last := mainWin.Canvas().Size()
		ticker := time.NewTicker(400 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			sz := mainWin.Canvas().Size()
			if sz.Width != last.Width || sz.Height != last.Height {
				last = sz
				fyne.Do(func() { adjustAllTables() })
			}
		}
	}()

	// Confirm on close with additional warning if users are clocked in
	mainWin.SetCloseIntercept(func() { handleCloseRequest() })

	mainWin.Show()
	focusScanEntry()
	if len(dataLoadErrors) > 0 {
		showError(fmt.Errorf("Some data files could not be loaded:\n\n%s", strings.Join(dataLoadErrors, "\n")))
	}
	appRef.Run()
}

// trySetAppIcon loads an icon from common paths and sets it for the app/window.
// This affects window/taskbar icons at runtime. For embedding the executable icon
// on Windows, use `fyne package -os windows -icon icon.png` during packaging.
func trySetAppIcon() {
	candidates := []string{"icon.png", "assets/icon.png"}
	for _, p := range candidates {
		if b, err := os.ReadFile(p); err == nil && len(b) > 0 {
			res := fyne.NewStaticResource(filepath.Base(p), b)
			appRef.SetIcon(res)
			if mainWin != nil {
				mainWin.SetIcon(res)
			}
			break
		}
	}
}

// buildHomeView composes the landing view
func buildHomeView() fyne.CanvasObject {
	prompt := widget.NewLabel("Present your badge to the scanner, or enter your Member ID below.")
	primary := widget.NewButton("Clock In / Clock Out", submitAction)
	primary.Importance = widget.HighImportance
	lookup := widget.NewButton("Find a Member", showUserLookup)
	lookup.Importance = widget.HighImportance
	clockedIn := widget.NewButton("View Clocked-In Members", viewClockedInUsers)
	clockOutAll := widget.NewButton("Clock Out All Members", clockOutAllUsers)
	clockedIn.Importance = widget.HighImportance
	clockOutAll.Importance = widget.DangerImportance

	scanCard := surface(container.NewVBox(
		sectionHeading("Member check-in", "Ready for the next member"),
		prompt,
		container.NewBorder(nil, nil, nil, primary, idEntry),
		successLb,
	))
	statusCard := surface(container.NewVBox(
		sectionHeading("Live status", "Current activity at a glance"),
		clockBar,
		clockedIn,
		lookup,
	))
	toolsCard := surface(container.NewBorder(nil, nil, nil, clockOutAll, container.NewVBox(
		sectionHeading("Coordinator tools", "For adjustments and end-of-day tasks"),
		widget.NewLabel("Manual Entry records a duration without changing a member's current status."),
	)))
	return container.NewBorder(
		container.NewVBox(sectionHeading("Time clock", "A simple, reliable way to record member time."), widget.NewSeparator()),
		toolsCard,
		nil, nil,
		container.NewGridWithColumns(2, scanCard, statusCard),
	)
}

func buildAppHeader(closeButton *widget.Button) fyne.CanvasObject {
	background := canvas.NewRectangle(navyColor)
	markBackground := canvas.NewRectangle(primaryColor)
	markBackground.SetMinSize(fyne.NewSize(42, 42))
	mark := canvas.NewText("OC", color.White)
	mark.TextStyle = fyne.TextStyle{Bold: true}
	mark.TextSize = 18
	mark.Alignment = fyne.TextAlignCenter
	logo := container.NewMax(markBackground, container.NewCenter(mark))
	title := canvas.NewText("OpenChronoLog", color.White)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 20
	subtitle := canvas.NewText("MEMBER TIME CLOCK", color.NRGBA{R: 208, G: 213, B: 221, A: 255})
	subtitle.TextSize = 11
	date := canvas.NewText(time.Now().Format("Monday, January 2, 2006"), color.NRGBA{R: 208, G: 213, B: 221, A: 255})
	brand := container.NewHBox(logo, container.NewVBox(title, subtitle))
	content := container.NewBorder(nil, nil, brand, container.NewHBox(date, closeButton), nil)
	return container.NewMax(background, container.NewPadded(content))
}

// handleCloseRequest centralizes the quit confirmation flow for both the
// window close intercept and the explicit Close button.
func handleCloseRequest() {
	in := countClockedIn()
	if in > 0 {
		showConfirm(
			"Members Clocked In",
			fmt.Sprintf("%d member(s) are still clocked in. Continue quitting?", in),
			func(ok bool) {
				if !ok {
					return
				}
				showConfirm("Quit", "Do you want to quit?", func(ok2 bool) {
					if ok2 {
						mainWin.Close()
					}
				})
			},
		)
		return
	}
	showConfirm("Quit", "Do you want to quit?", func(ok bool) {
		if ok {
			mainWin.Close()
		}
	})
}
