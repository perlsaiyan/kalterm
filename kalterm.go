package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	timeseries "github.com/codesuki/go-time-series"
	"github.com/mitchellh/mapstructure"
	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/tcell"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgets/donut"
	"github.com/mum4k/termdash/widgets/linechart"
	"github.com/mum4k/termdash/widgets/text"
)

type kalExperience struct {
	Current int `json:"cumulative"`
}

type kalGold struct {
	Current int `json:"current"`
}

type kalLocation struct {
	Path     string `json:"path"`
	Length   int    `json:"length"`
	Position int    `json:"position"`
}

type kalMessage struct {
	Text string
}

// playType indicates how to play a donut.
type playType int

var loc kalLocation
var logMessage kalMessage
var xpSeries *timeseries.TimeSeries
var goldSeries *timeseries.TimeSeries
var xpprevious int
var goldprevious int
var xpByMinute []float64
var goldByMinute []float64
var xpByHour []float64
var goldByHour []float64
var xpgain int
var goldgain int

const (
	playTypePercent playType = iota
	playTypeAbsolute
)

func parseJSON(keys map[string]interface{}, blob interface{}, c chan string) {
	for i := range keys {
		switch i {
		case "experience":
			var xp kalExperience
			mapstructure.Decode(keys[i], &xp)

			if xpprevious > 0 {
				xpgain = xp.Current - xpprevious
				xpprevious = xp.Current
				if xpgain > 0 {
					xpSeries.Increase(xpgain)
				}
			} else {
				xpprevious = xp.Current
			}
		case "gold":
			var gold kalGold
			mapstructure.Decode(keys[i], &gold)

			if goldprevious > 0 {
				goldgain = gold.Current - goldprevious
				goldprevious = gold.Current
				if goldgain > 0 {
					goldSeries.Increase(goldgain)
				}
			} else {
				goldprevious = gold.Current
			}
		case "location":
			mapstructure.Decode(keys[i], &loc)
			loc.Position = loc.Position - 1
			//log.Printf("Path: %v / %d on %s", loc.Position, loc.Length, loc.Path)

		case "message":
			mapstructure.Decode(keys[i], &logMessage)
			c <- logMessage.Text
		}
	}
}

func playPathDonut(ctx context.Context, d *donut.Donut, start, step int, delay time.Duration, pt playType) {
	progress := 0

	ticker := time.NewTicker(delay)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			switch pt {
			case playTypePercent:
				if err := d.Percent(progress); err != nil {
					panic(err)
				}
			case playTypeAbsolute:
				if loc.Length == 0 {
					loc.Length = 1
				}

				if err := d.Absolute(loc.Position, loc.Length); err != nil {
					panic(err)
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

func playXPChart(ctx context.Context, lc *linechart.LineChart, delay time.Duration) {

	ticker := time.NewTicker(delay)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if x, err := xpSeries.Recent(time.Minute); err == nil {
				if x > 0 {
					extrapolate := 60 * x / 1000000
					xpByMinute = append(xpByMinute, extrapolate)
				}
			}

			if x, err := xpSeries.Recent(time.Hour); err == nil {
				if x > 0 {
					xpByHour = append(xpByHour, x/float64(1000000.00))
				}
			} else {
				xpByHour = append(xpByHour, 0)
			}

			if err := lc.Series("extrap", xpByMinute,
				linechart.SeriesCellOpts(cell.FgColor(cell.ColorNumber(33))),
				linechart.SeriesXLabels(map[int]string{
					0: "start",
				}),
			); err != nil {
				panic(err)
			}

			if err := lc.Series("hour", xpByHour,
				linechart.SeriesCellOpts(cell.FgColor(cell.ColorWhite))); err != nil {
				panic(err)
			}

		case <-ctx.Done():
			return
		}
	}
}

func playGoldChart(ctx context.Context, lc *linechart.LineChart, delay time.Duration) {

	ticker := time.NewTicker(delay)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if x, err := goldSeries.Recent(time.Minute); err == nil {
				if x > 0 {
					extrapolate := 60 * x / 1000000
					goldByMinute = append(goldByMinute, extrapolate)
				}
			}

			if x, err := goldSeries.Recent(time.Hour); err == nil {
				if x > 0 {
					goldByHour = append(xpByHour, x/float64(1000000.00))
				}
			} else {
				goldByHour = append(goldByHour, 0)
			}

			if err := lc.Series("extrap", goldByMinute,
				linechart.SeriesCellOpts(cell.FgColor(cell.ColorNumber(33))),
				linechart.SeriesXLabels(map[int]string{
					0: "start",
				}),
			); err != nil {
				panic(err)
			}

			if err := lc.Series("hour", goldByHour,
				linechart.SeriesCellOpts(cell.FgColor(cell.ColorWhite))); err != nil {
				panic(err)
			}

		case <-ctx.Done():
			return
		}
	}
}

func writeLines(ctx context.Context, t *text.Text, c chan string) {

	for {

		msg := <-c
		t.Write(fmt.Sprintf("%s\n", msg))
	}
}

func tintinReader(conn net.Conn, c chan string) {
	log.Printf("Connecting to tintin instance")
	reader := bufio.NewReader(conn)
	var JSON interface{}
	for {
		message, _ := reader.ReadString('\n')
		json.Unmarshal([]byte(message), &JSON)
		if JSON != nil {
			list := JSON.(map[string]interface{})
			parseJSON(list, JSON, c)
		} else {
			//log.Print("Got message: ", message)
		}
	}
}

func main() {

	connHost := flag.String("host", "127.0.0.1", "host/ip of tintin instance")
	connPort := flag.String("port", "9595", "port")

	flag.Parse()

	conn, err := net.Dial("tcp", *connHost+":"+*connPort)
	if err != nil {
		fmt.Println("Error connecting:", err.Error())
		os.Exit(1)
	}

	t, err := tcell.New()
	if err != nil {
		panic(err)
	}
	defer t.Close()

	ctx, cancel := context.WithCancel(context.Background())

	pathDonut, err := donut.New(
		donut.CellOpts(cell.FgColor(cell.ColorGreen)),
		donut.Label("Path", cell.FgColor(cell.ColorGreen)),
	)

	xpSeries, err = timeseries.NewTimeSeries()
	goldSeries, err = timeseries.NewTimeSeries()

	xpLC, err := linechart.New(
		linechart.AxesCellOpts(cell.FgColor(cell.ColorRed)),
		linechart.YLabelCellOpts(cell.FgColor(cell.ColorGreen)),
		linechart.XLabelCellOpts(cell.FgColor(cell.ColorCyan)),
		/* linechart.YAxisCustomScale(float64(0), float64(60)), */
	)

	goldLC, err := linechart.New(
		linechart.AxesCellOpts(cell.FgColor(cell.ColorRed)),
		linechart.YLabelCellOpts(cell.FgColor(cell.ColorGreen)),
		linechart.XLabelCellOpts(cell.FgColor(cell.ColorCyan)),
		/*linechart.YAxisCustomScale(float64(0), float64(20)),*/
	)

	textblock, err := text.New(text.RollContent(), text.WrapAtWords())
	msgChannel := make(chan string)
	go tintinReader(conn, msgChannel)
	go playPathDonut(ctx, pathDonut, 0, 1, 250*time.Millisecond, playTypeAbsolute)
	go playXPChart(ctx, xpLC, time.Minute)
	go playGoldChart(ctx, goldLC, time.Minute)
	go writeLines(ctx, textblock, msgChannel)

	c, err := container.New(
		t,
		container.Border(linestyle.Light),
		container.BorderTitle("PRESS Q TO QUIT"),
		container.SplitVertical(
			container.Left(
				container.SplitHorizontal(
					container.Top(container.PlaceWidget(textblock), container.Border(linestyle.Light), container.BorderTitle("Log")),
					container.Bottom(container.PlaceWidget(pathDonut), container.Border(linestyle.Light), container.BorderTitle("Path")),
				),
			),

			container.Right(
				container.SplitHorizontal(
					container.Top(container.PlaceWidget(xpLC), container.BorderTitle("XP"), container.Border(linestyle.Light)),
					container.Bottom(container.PlaceWidget(goldLC), container.BorderTitle("Gold"), container.Border(linestyle.Light)),
				),
			),
		),
	)
	if err != nil {
		panic(err)
	}

	quitter := func(k *terminalapi.Keyboard) {
		if k.Key == 'q' || k.Key == 'Q' {
			cancel()
		}
	}
	if err := termdash.Run(ctx, t, c, termdash.KeyboardSubscriber(quitter), termdash.RedrawInterval(1*time.Second)); err != nil {
		panic(err)

	}
}
