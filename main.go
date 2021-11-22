package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ttacon/chalk"
)

const (
	URLTemplate  = "https://wow.zamimg.com/images/wow/icons/large/%s.jpg"
	FileTemplate = "%s.jpg"
)

func main() {
	fmt.Print(chalk.Cyan, "-+-+-+-+- WoW Icon Downloader -+-+-+-+-\n\n", chalk.Reset)

	slugs := os.Args[1:]

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

	var numErrs int
	for _, ID := range slugs {
		fmt.Printf("Downloading %s... ", ID)
		if err := download(ID); err != nil {
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

func download(ID string) error {
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
	_, err = io.Copy(out, resp.Body) // TODO: should probably verify bytes written
	return err
}
