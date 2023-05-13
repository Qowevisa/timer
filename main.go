package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/fatih/color"
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
)

const TempFile = "/tmp/timers.json"

func loadTimers() {
	mutex.Lock()
	defer mutex.Unlock()

	data, err := ioutil.ReadFile(TempFile)
	if err != nil {
		return
	}

	json.Unmarshal(data, &timers)
	for id, timer := range timers {
		if timer.Name == "123" {
		}
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
					if hours == 0 && mins == 0 && secs == 0 {
						if timeString == "" {
							color.Red("You didn't set up any of the needed arguments. Stupid")
							return nil
						} else {
							// TODO tokenize this shit
						}
					}
					end := time.Now().Add(time.Duration(secs) * time.Second)
					end = end.Add(time.Duration(mins) * time.Minute)
					end = end.Add(time.Duration(hours) * time.Hour)
					loadTimers()
					mutex.Lock()
					timers[counter] = &Timer{
						Name: name,
						End:  end,
					}
					fmt.Printf("Started timer %d : %s -> %dh\n", counter, name, hours)
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
						fmt.Printf("You have %d timers\n", len(timers))
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
							color.Green("Finished\n")
						} else {
							_mins := int(math.Floor(timerLeft.Minutes()))
							switch {
							case 0 <= _mins && _mins <= 15:
								color.Red("%02d:%02d:%02d.%04d left", hours, mins, secs, mils)
							case 16 <= _mins && _mins <= 30:
								color.Yellow("%02d:%02d:%02d.%04d left", hours, mins, secs, mils)
							default:
								color.Blue("%02d:%02d:%02d.%04d left", hours, mins, secs, mils)
							}
						}
					}
					fmt.Printf("\n")
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
						color.Green("Deleted timer %d\n", id)
					} else {
						color.Red("No timer with id %d\n", id)
					}
					fmt.Printf("\n")
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
