package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli/v2"
)

type Timer struct {
	Name string
	End  time.Time
}

var (
	timers  = make(map[int]*Timer)
	counter = -1
	mutex   = &sync.Mutex{}
	colors  = Colors{}
	esc     = Escape{}
)

const TempFile = "/tmp/timers.json"

// Escape strings
type Escape struct{}

func (e Escape) ClearLine() string {
	return "\033[2K"
}

func (e Escape) MoveHorCur() string {
	return "\033[G"
}

// Colors
type Colors struct{}

func (c Colors) Reset() string {
	return "\033[38;5;7m"
}

func (c Colors) Red() string {
	return "\033[38;5;1m"
}

func (c Colors) Green() string {
	return "\033[38;5;2m"
}

func (c Colors) Yellow() string {
	return "\033[38;5;3m"
}

func (c Colors) Blue() string {
	return "\033[38;5;4m"
}

func loadTimers() {
	mutex.Lock()
	defer mutex.Unlock()

	data, err := ioutil.ReadFile(TempFile)
	if err != nil {
		counter = 0
		return
	}

	json.Unmarshal(data, &timers)
	for id := range timers {
		if id > counter {
			counter = id
		}
	}
	counter++
}

func saveTimers() {
	mutex.Lock()
	defer mutex.Unlock()

	data, err := json.Marshal(timers)
	if err != nil {
		fmt.Println("Error saving timers:", err)
		return
	}

	ioutil.WriteFile(TempFile, data, 0644)
}

func parseTimeString(end *time.Time, timeString string) error {
	parts := strings.Split(timeString, ":")
	var hours, mins, secs int
	var err error
	switch len(parts) {
	case 1:
		hours, err = strconv.Atoi(parts[0])
		if err != nil {
			return err
		}
		break
	case 2:
		hours, err = strconv.Atoi(parts[0])
		if err != nil {
			return err
		}

		mins, err = strconv.Atoi(parts[1])
		if err != nil {
			return err
		}
		break
	case 3:
		hours, err = strconv.Atoi(parts[0])
		if err != nil {
			return err
		}

		mins, err = strconv.Atoi(parts[1])
		if err != nil {
			return err
		}

		secs, err = strconv.Atoi(parts[2])
		if err != nil {
			return err
		}
		break
	}
	now := time.Now()
	_, offset := now.Zone()
	_timeString := fmt.Sprintf(
		"%d-%02d-%02dT%02d:%02d:%02d+%02d:00",
		now.Year(),
		now.Month(),
		now.Day(),
		hours,
		mins,
		secs,
		offset/3600)

	// Parse time string
	ret, err := time.Parse(time.RFC3339, _timeString)
	if err != nil {
		return err
	}

	if ret.Unix() < now.Unix() {
		ret = ret.Add(24 * time.Hour)
	}

	*end = ret
	return nil
}

func formatDuration(d time.Duration) string {
	total := float64(d)
	var unit string
	switch {
	case d >= time.Hour:
		total = total / float64(time.Hour)
		unit = "h"
	case d >= time.Minute:
		total = total / float64(time.Minute)
		unit = "m"
	default:
		total = total / float64(time.Second)
		unit = "s"
	}
	return fmt.Sprintf("%.2f%s", total, unit)
}

func main() {
	app := &cli.App{
		UseShortOptionHandling: true,
		Commands: []*cli.Command{
			{
				Name:    "add",
				Aliases: []string{"a"},
				Usage:   "add new timer",
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "hours", Aliases: []string{"H"}},
					&cli.IntFlag{Name: "minutes", Aliases: []string{"M"}},
					&cli.IntFlag{Name: "seconds", Aliases: []string{"S"}},
					&cli.StringFlag{Name: "time", Aliases: []string{"T"}},
					&cli.StringFlag{Name: "name", Aliases: []string{"n"}},
				},
				Action: func(cCtx *cli.Context) error {
					name := cCtx.String("name")
					hours := cCtx.Int("hours")
					mins := cCtx.Int("minutes")
					secs := cCtx.Int("seconds")
					timeString := cCtx.String("time")
					var end time.Time
					if hours == 0 && mins == 0 && secs == 0 {
						if timeString == "" {
							fmt.Printf(
								"%sYou didn't set up any of the needed arguments. Stupid%s\n",
								colors.Red(),
								colors.Reset(),
							)
							return nil
						} else {
							// TODO tokenize this shit
							err := parseTimeString(&end, timeString)
							if err != nil {
								fmt.Printf("%sERROR: %s%s\n", colors.Red(), err, colors.Reset())
								return nil
							}
						}
					} else {
						now := time.Now()
						end = now.Add(time.Duration(hours) * time.Hour)
						end = end.Add(time.Duration(mins) * time.Minute)
						end = end.Add(time.Duration(secs) * time.Second)
					}
					loadTimers()
					mutex.Lock()
					timers[counter] = &Timer{
						Name: name,
						End:  end,
					}
					fmt.Printf(
						"%sStarted timer %d%s : %s%s%s\n",
						colors.Green(),
						counter,
						colors.Reset(),
						colors.Blue(),
						name,
						colors.Reset())
					counter++
					mutex.Unlock()
					saveTimers()

					return nil
				},
			},
			{
				Name:    "list",
				Aliases: []string{"l", "ls"},
				Usage:   "list all timers",
				Action: func(cCtx *cli.Context) error {
					loadTimers()
					mutex.Lock()
					if len(timers) > 0 {
						fmt.Printf(
							"You have %s%d%s timers\n",
							colors.Blue(),
							len(timers),
							colors.Reset())
					} else {
						fmt.Printf("You don't have any timers. Set up with add command\n")
						return nil
					}
					for id, timer := range timers {
						timerLeft := timer.End.Sub(time.Now())
						hours := int(math.Floor(timerLeft.Hours()))
						mins := int(math.Floor(timerLeft.Minutes())) % 60
						secs := int(math.Floor(timerLeft.Seconds())) % 60
						mils := timerLeft.Microseconds() % 1000
						// printing stuff
						fmt.Printf("%d : %s -> ", id, timer.Name)
						if mils <= 0 {
							fmt.Printf("%sFinished%s\n", colors.Green(), colors.Reset())
						} else {
							_mins := int(math.Floor(timerLeft.Minutes()))
							switch {
							case 0 <= _mins && _mins <= 15:
								fmt.Printf("%s%02d:%02d:%02d.%04d%s left\n", colors.Red(), hours, mins, secs, mils, colors.Reset())
							case 16 <= _mins && _mins <= 30:
								fmt.Printf("%s%02d:%02d:%02d.%04d%s left\n", colors.Yellow(), hours, mins, secs, mils, colors.Reset())
							default:
								fmt.Printf("%s%02d:%02d:%02d.%04d%s left\n", colors.Blue(), hours, mins, secs, mils, colors.Reset())
							}
						}
					}
					mutex.Unlock()
					return nil
				},
			},
			{
				Name:    "delete",
				Aliases: []string{"d"},
				Usage:   "delete a timer",
				Action: func(c *cli.Context) error {
					id, err := strconv.Atoi(c.Args().First())
					if err != nil {
						fmt.Println("Please provide a valid id")
						return err
					}
					loadTimers()
					mutex.Lock()
					if _, exists := timers[id]; exists {
						delete(timers, id)
						fmt.Printf("%sDeleted timer %d%s\n", colors.Green(), id, colors.Reset())
					} else {
						fmt.Printf("%sNo timer with id %d%s\n", colors.Red(), id, colors.Reset())
					}
					mutex.Unlock()
					saveTimers()

					return nil
				},
			},
			{
				Name:    "clear",
				Aliases: []string{"c"},
				Usage:   "clear all timers",
				Action: func(c *cli.Context) error {
					saveTimers()
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
