package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ttacon/chalk"
)

const (
	URLTemplate       = "https://wow.zamimg.com/images/wow/icons/large/%s.jpg"
	FileTemplate      = "%s.jpg"
	DefaultOverlayURL = "https://wow.zamimg.com/images/Icon/large/border/default.png"
)

type ImagePipeline func(r io.Reader, w io.Writer) error

func main() {
	fmt.Print(chalk.Cyan, "-+-+-+-+- WoW Icon Downloader -+-+-+-+-\n\n", chalk.Reset)

	var overlayPath string
	var noOverlay bool
	flag.StringVar(&overlayPath, "overlay", DefaultOverlayURL, "Path to a png image to overlay on top of icons.")
	flag.BoolVar(&noOverlay, "no-overlay", false, "If set, nothing will be overlayed on top of icons.")
	flag.Parse()

	// TODO: Implement help/usage flag. Currently it not very useful.

	slugs := flag.Args()
	for len(slugs) == 0 {
		fmt.Println("Paste the IDs of the icons you want to download, separated by spaces:")
		fmt.Println("For example: classicon_paladin inv_ore_oxxein")
		fmt.Println("You can also enter 'q' to quit.")
		var in string
		var err error
		reader := bufio.NewReader(os.Stdin)
		for in == "" {
			fmt.Print("> ")
			if in, err = reader.ReadString('\n'); err != nil {
				fmt.Printf("Failed to read input: %s\n", err.Error())
				time.Sleep(1 * time.Second)
				os.Exit(1)
				return
			}
			in = strings.TrimSpace(in)
		}
		if in == "q" {
			os.Exit(0)
			return
		}
		slugs = strings.Split(strings.TrimSpace(in), " ")
	}

	var transformer ImagePipeline
	if !noOverlay {
		var err error
		transformer, err = buildImageOverlayPipeline(overlayPath)
		if err != nil {
			fmt.Println(chalk.Red, err.Error(), chalk.Reset)
			os.Exit(1)
		}
	}

	var numErrs int
	for _, ID := range slugs {
		fmt.Printf("Downloading %s... ", ID)
		if err := download(ID, transformer); err != nil {
			fmt.Println(chalk.Red, chalk.Bold, "[ERROR]", chalk.Reset, err.Error())
			numErrs++
		} else {
			fmt.Println(chalk.Green, chalk.Bold, "[SUCCESS]", chalk.Reset)
		}
	}

	if numErrs == 0 {
		fmt.Println(chalk.Green, "All downloads completed successfully.", chalk.Reset)
	} else if numErrs < len(slugs) {
		fmt.Printf(chalk.Yellow.Color("\nCompleted with %d errors.\n"), numErrs)
	} else {
		fmt.Println(chalk.Red, "All downloads failed.", chalk.Reset)
	}
}

// download fetches the icon for the given ID and applies an optional
// transformation on the data before writing it to a local file.
func download(ID string, pipeline ImagePipeline) error {
	resp, err := http.Get(fmt.Sprintf(URLTemplate, ID))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}
	defer resp.Body.Close()
	out, err := os.Create(fmt.Sprintf(FileTemplate, ID))
	if err != nil {
		return err
	}
	defer out.Close()
	if pipeline == nil {
		_, err = io.Copy(out, resp.Body) // TODO: should probably verify bytes written
	} else {
		err = pipeline(resp.Body, out)
	}
	return err
}

func isValidUrl(toTest string) bool {
	_, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}

	u, err := url.Parse(toTest)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}

// fetchOverlay tries to retrieve and decode a png image from the given path.
// If the path is a remote URL, then it is fetched, otherwise it is assumed that
// the path is a local file.
func fetchOverlay(path string) (image.Image, error) {
	if isValidUrl(path) {
		resp, err := http.Get(path)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch overlay image from '%s' | %w", path, err)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch overlay image from '%s' | Unexpected response status: %s", path, resp.Status)
		}
		defer resp.Body.Close()
		img, err := png.Decode(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to decode overlay image | %w", err)
		}
		return img, nil
	} else {
		data, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open local overlay image at '%s' | %w", path, err)
		}
		defer data.Close()
		img, err := png.Decode(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode overlay image | %w", err)
		}
		return img, nil
	}
}

func overlayImage(base, overlay image.Image, baseOffset, overlayOffset *image.Point) *image.RGBA {
	if baseOffset == nil {
		baseOffset = &image.Point{X: -7, Y: -7}
	}
	if overlayOffset == nil {
		overlayOffset = &image.Point{}
	}
	bounds := base.Bounds().Union(overlay.Bounds())
	img := image.NewRGBA(bounds)
	draw.Draw(img, bounds, base, *baseOffset, draw.Src)
	draw.Draw(img, bounds, overlay, *overlayOffset, draw.Over)
	return img
}

func buildImageOverlayPipeline(overlayPath string) (ImagePipeline, error) {
	overlay, err := fetchOverlay(overlayPath)
	if err != nil {
		return nil, err
	}
	return func(r io.Reader, w io.Writer) error {
		base, err := jpeg.Decode(r)
		if err != nil {
			return err
		}
		img := overlayImage(base, overlay, nil, nil) // TODO: add flags to configure offset
		return jpeg.Encode(w, img, &jpeg.Options{Quality: 100})
	}, nil
}
