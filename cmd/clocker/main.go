/*
 * Copyright (c) Benjamin Zores
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	ClockFrequency = 1 * time.Second
	ConfigFile     = ".clocker"
)

var trackers = []*Tracker{}

type Tracker struct {
	Label   string        `yaml:"label"`
	Elapsed time.Duration `yaml:"elapsed"`
	Active  bool          `yaml:"-"`
	Timer   chan struct{} `yaml:"-"`

	// UI References
	PlayButton *widget.Button `yaml:"-"`

	// Data Bindings
	LabelStr   binding.String `yaml:"-"`
	ElapsedStr binding.String `yaml:"-"`
}

func (t *Tracker) Start() {

	go func() {
		for {
			select {
			default:
			case <-t.Timer: // stop call
				return
			}
			t.Elapsed += ClockFrequency
			_ = t.ElapsedStr.Set(shortDur(t.Elapsed))
			time.Sleep(ClockFrequency)
		}
	}()

	t.Active = true
	t.PlayButton.SetIcon(theme.MediaPauseIcon())
}

func (t *Tracker) Stop() {
	t.Timer <- struct{}{}
	t.Active = false
	t.PlayButton.SetIcon(theme.MediaPlayIcon())
}

func NewTracker(label string, duration time.Duration) {
	t := &Tracker{
		Label:      label,
		Elapsed:    duration,
		Active:     false,
		Timer:      make(chan struct{}, 1),
		LabelStr:   binding.NewString(),
		ElapsedStr: binding.NewString(),
	}

	_ = t.LabelStr.Set(t.Label)
	_ = t.ElapsedStr.Set(shortDur(t.Elapsed))
	trackers = append(trackers, t)
}

func DeleteTracker(t *Tracker) {
	for idx, id := range trackers {
		if id == t {
			trackers = append(trackers[:idx], trackers[idx+1:]...)
			break
		}
	}
}

func ResetTrackers() {
	for _, t := range trackers {
		t.Stop()
		t.Elapsed = 0
		_ = t.ElapsedStr.Set("0s")
	}
}

func shortDur(d time.Duration) string {
	s := d.String()
	if strings.HasSuffix(s, "m0s") {
		s = s[:len(s)-2]
	}
	if strings.HasSuffix(s, "h0m") {
		s = s[:len(s)-2]
	}
	return s
}

func addTrackerDialog(w fyne.Window) {
	tracker := widget.NewEntry()
	items := []*widget.FormItem{
		widget.NewFormItem("", tracker),
		widget.NewFormItem("", widget.NewLabel("")),
	}

	dialog.ShowForm("New Tracker", "Add", "Cancel", items, func(b bool) {
		if !b {
			return
		}
		log.Println("Adding new clock", tracker.Text)
		NewTracker(tracker.Text, 0)
		update(w)
	}, w)
}

func editTrackerDialog(w fyne.Window, t *Tracker) {
	tracker := widget.NewEntry()
	tracker.SetText(t.Label)
	items := []*widget.FormItem{
		widget.NewFormItem("", tracker),
		widget.NewFormItem("", widget.NewLabel("")),
	}

	dialog.ShowForm("Edit Tracker", "Update", "Cancel", items, func(b bool) {
		if !b {
			return
		}
		t.Label = tracker.Text
		_ = t.LabelStr.Set(tracker.Text)
		log.Println("Updating new clock", tracker.Text)
	}, w)
}

func deleteTrackerDialog(w fyne.Window, t *Tracker) {
	text := fmt.Sprintf("Are you sure you want to delete tracker %s ?", t.Label)
	dialog.ShowConfirm("Delete Tracker ?", text, func(b bool) {
		if !b {
			return
		}
		DeleteTracker(t)
		update(w)
	}, w)
}

func resetTrackersDialog(w fyne.Window) {
	dialog.ShowConfirm("Reset timers ?", "Are you sure you want to reset all counters ?", func(b bool) {
		if !b {
			return
		}
		ResetTrackers()
		update(w)
	}, w)
}

func makeMenu(w fyne.Window) fyne.CanvasObject {
	return container.NewGridWithColumns(2,
		widget.NewButtonWithIcon("", theme.ListIcon(), func() {
			addTrackerDialog(w)
		}),
		widget.NewButtonWithIcon("", theme.HistoryIcon(), func() {
			resetTrackersDialog(w)
		}),
	)
}

func makeTrackerList(w fyne.Window) fyne.CanvasObject {
	trackerList := []fyne.CanvasObject{}
	for _, t := range trackers {
		playButton := widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {})
		playButton.OnTapped = func() {
			if t.Active {
				t.Stop()
			} else {
				t.Start()
			}
		}
		t.PlayButton = playButton

		label := widget.NewLabel("")
		label.Bind(t.LabelStr)

		elapsed := widget.NewLabel("")
		elapsed.Bind(t.ElapsedStr)

		editButton := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
			editTrackerDialog(w, t)
		})

		trashButton := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
			deleteTrackerDialog(w, t)
		})

		settingsBox := container.NewHBox(elapsed, editButton, trashButton)

		c := container.NewBorder(nil, nil, playButton, settingsBox, label)
		trackerList = append(trackerList, c)
	}

	return container.NewVBox(trackerList...)
}

func update(w fyne.Window) {
	menu := makeMenu(w)
	trackers := makeTrackerList(w)
	panel := container.NewBorder(nil, menu, nil, nil, trackers)
	w.SetContent(panel)
	saveConfig()
}

func readConfig() {
	var trackers []Tracker

	home, _ := os.UserHomeDir()
	confFile := fmt.Sprintf("%s/%s", home, ConfigFile)

	cfg, err := os.Open(confFile)
	if err != nil {
		return
	}
	defer cfg.Close()

	contents, _ := io.ReadAll(cfg)
	err = yaml.Unmarshal(contents, &trackers)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, t := range trackers {
		NewTracker(t.Label, t.Elapsed)
	}
}

func saveConfig() {
	home, _ := os.UserHomeDir()
	confFile := fmt.Sprintf("%s/%s", home, ConfigFile)

	content, _ := yaml.Marshal(trackers)
	_ = os.WriteFile(confFile, content, 0600)
}

func main() {
	a := app.New()
	w := a.NewWindow("Clocker")
	readConfig()
	update(w)
	w.Resize(fyne.NewSize(400, 800))
	w.SetOnClosed(func() {
		saveConfig()
	})
	w.ShowAndRun()
}
