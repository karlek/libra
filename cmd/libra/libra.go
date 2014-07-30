package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"os"
	"runtime"
	"time"

	"code.google.com/p/draw2d/draw2d"
	"github.com/mewkiz/pkg/errutil"
	"github.com/mewmew/sdl/audio"
	"github.com/mewmew/sdl/win"
	"github.com/mewmew/we"
	snd "github.com/mkb218/gosndfile/sndfile"
)

const (
	// TODO(karlek): make width and height into flags.
	width, height = 640, 480

	fps = 60
)

func play(filename string) {
	// Load the sound file.
	s, err := audio.Open(filename)
	if err != nil {
		log.Fatalln(err)
	}

	// Play the sound file.
	snd, err := s.Play()
	if err != nil {
		log.Fatalln(err)
	}

	// Wait until the sound has reached the end.
	<-snd.End
}

func oscilloscope(filename string) (err error) {
	// Open the window.
	err = win.Open(width, height, win.Resizeable)
	if err != nil {
		return errutil.Err(err)
	}
	defer win.Close()

	// Info about the sound, number of samples, sample rate etc.
	info := snd.Info{}

	// snd.Read is the read flag.
	f, err := snd.Open(filename, snd.Read, &info)
	if err != nil {
		return errutil.Err(err)
	}
	defer f.Close()

	// Play the music.
	go play(filename)

	// Update and event loop.
	for {
		err := update((2*info.Samplerate)/fps, f)
		if err != nil {
			return errutil.Err(err)
		}

		// Display window updates on screen.
		err = win.Update()
		if err != nil {
			return errutil.Err(err)
		}

		// Poll events until the event queue is empty.
		for e := win.PollEvent(); e != nil; e = win.PollEvent() {
			switch e.(type) {
			case we.Close:
				os.Exit(0)
			}
		}
		time.Sleep(time.Second / fps)
	}
	return nil
}

func update(samplerate int32, f *snd.File) error {
	// frames holds 1 second of frames, the number of frames depends on the
	// sample rate.
	frames := make([]int32, samplerate)

	read, err := f.ReadFrames(frames)
	if err != nil {
		return errutil.Err(err)
	}
	fmt.Println(len(frames), read)
	// when 0 frames are read, it means we've reached EOF.
	if read == 0 {
		return errutil.NewNoPos("EOF")
	}

	// The x scale is relative to the number of frames.
	xscale := int(len(frames)) / width

	// The y scale is relative to the range of values from the frames.
	yscale := Range(frames) / height * 5

	i := image.NewRGBA(image.Rect(0, 0, width, height))
	black := color.RGBA{0, 0, 0, 255}
	draw.Draw(i, i.Bounds(), &image.Uniform{black}, image.ZP, draw.Src)

	oldx, oldy := 0, height/2
	for x := 0; x < width; x += 4 {
		// loudness from the frame scaled down to the image.
		loudness := int(frames[x*xscale]) / yscale

		// To center the oscilloscope on the y axis we add height/2.
		y := loudness + height/2

		// draw a line between (x, y) and (oldx, oldy)
		line(i, x, y, oldx, oldy)

		// Update old x/y.
		oldx, oldy = x, y
	}
	img, err := win.ReadImage(i)
	if err != nil {
		return errutil.Err(err)
	}
	return win.Draw(image.ZP, img)
}

func line(i *image.RGBA, x, y, x0, y0 int) {
	gc := draw2d.NewGraphicContext(i)
	gc.SetStrokeColor(color.RGBA{0xff, 0xff, 0xff, 0xff})
	gc.MoveTo(float64(x), float64(y))
	gc.LineTo(float64(x0), float64(y0))
	gc.Stroke()
}

func Range(frames []int32) int {
	var max, min int32 = 0, 0
	for _, frame := range frames {
		if frame > max {
			max = frame
		} else if frame < min {
			min = frame
		}
	}
	return int(max - min)
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [FILE],,,\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	// The audio library is automatically initialized when imported. Quit the
	// audio library on return.
	defer audio.Quit()

	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
	}
	for _, path := range flag.Args() {
		err := oscilloscope(path)
		if err != nil {
			log.Fatalln(err)
		}
	}
}
