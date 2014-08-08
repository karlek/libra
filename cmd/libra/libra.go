package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"os"
	"runtime"
	"time"

	"azul3d.org/audio.v1"
	_ "azul3d.org/audio/wav.v1"
	"code.google.com/p/draw2d/draw2d"
	"github.com/mewkiz/pkg/errutil"
	sdlaudio "github.com/mewmew/sdl/audio"
	"github.com/mewmew/sdl/win"
	"github.com/mewmew/we"
)

const (
	// TODO(karlek): make width and height into flags.
	width, height = 640, 480

	fps = 60
)

func play(filename string) {
	// Load the sound file.
	s, err := sdlaudio.Open(filename)
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
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	br := bufio.NewReader(f)

	dec, magic, err := audio.NewDecoder(br)
	if err != nil {
		return err
	}
	fmt.Println("magic:", magic)
	conf := dec.Config()
	fmt.Println(conf)

	// Play the music.
	go play(filename)

	// buf holds 1/60 second of interleaved audio samples.
	nsamples := conf.Channels * conf.SampleRate
	buf := make(audio.PCM32Samples, nsamples/fps)
	// Update and event loop.
	for {
		err := update(buf, dec)
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

func update(buf audio.PCM32Samples, dec audio.Decoder) error {
	n, err := dec.Read(buf)
	if err != nil {
		return err
	}
	fmt.Println(len(buf), n)
	// when 0 frames are read, it means we've reached EOF.
	if n == 0 {
		return errutil.NewNoPos("EOF")
	}

	// The x scale is relative to the number of frames.
	xscale := int(len(buf)) / width

	// The y scale is relative to the range of values from the frames.
	yscale := Range(buf) / height * 5

	i := image.NewRGBA(image.Rect(0, 0, width, height))
	black := color.RGBA{0, 0, 0, 255}
	draw.Draw(i, i.Bounds(), &image.Uniform{black}, image.ZP, draw.Src)

	oldx, oldy := 0, height/2
	for x := 0; x < width; x += 4 {
		// loudness from the frame scaled down to the image.
		loudness := int(buf[x*xscale]) / yscale

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

func Range(frames audio.PCM32Samples) int {
	var max, min audio.PCM32 = 0, 0
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
	defer sdlaudio.Quit()

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
