package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

func getLogPath() string {
	path, ok := os.LookupEnv("BULLETLOG_FILE")
	if !ok {
		path = ".BULLETLOG"
	}

	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		file, err := os.Create(path)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
	}
	return path
}

const dateFormat = "20060102"

func getDate() (time.Time, error) {
	date, ok := os.LookupEnv("BULLETLOG_DATE")
	if ok {
		return time.Parse(dateFormat, date)
	}
	return time.Now().Truncate(24 * time.Hour), nil
}

func getDateFromHeader(line string) (*time.Time, error) {
	if !strings.HasPrefix(line, "##") {
		return nil, errors.New("The prefix must be ##")
	}
	f := strings.Fields(line)
	if len(f) != 2 {
		return nil, errors.New("Invalid header notion")
	}
	dateStr := f[1]
	t, err := time.Parse(dateFormat, dateStr)
	t = t.Truncate(24 * time.Hour)
	return &t, err
}

func addNote(c *cli.Context) error {
	return addBullet(c, "*")
}

func addTask(c *cli.Context) error {
	return addBullet(c, "-")
}

func addBullet(c *cli.Context, mark string) error {
	note := c.Args().First()

	entry := fmt.Sprintf("%s %s", mark, note)

	path := getLogPath()
	date, err := getDate()
	if err != nil {
		log.Fatal(err)
	}
	dateStr := date.Format(dateFormat)

	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}

	tmpfile, err := ioutil.TempFile("", ".BULLETLOG.*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if fileInfo.Size() == 0 {
		fmt.Fprintf(tmpfile, "## %s\n\n%s\n\n", dateStr, entry)
	} else {
		reader := bufio.NewReader(file)

		firstLine := true
		appended := false
		for {
			line, err := reader.ReadString('\n')

			if firstLine {
				latest, err := getDateFromHeader(line)
				if err != nil {
					log.Fatal(err)
				}
				if date.After(*latest) {
					// New section
					fmt.Fprintf(tmpfile, "## %s\n\n%s\n", dateStr, entry)
					appended = true
				}
				firstLine = false
			} else if !appended {
				t, err := getDateFromHeader(line)
				if err == nil {
					// Add an entry
					if date.After(*t) {
						fmt.Fprintf(tmpfile, "%s\n\n", entry)
					}
					appended = true
				}
			}

			fmt.Fprintf(tmpfile, line)
			if err != nil {
				break
			}
		}
		if !appended {
			fmt.Fprintf(tmpfile, "%s\n\n", entry)
		}

		if err != nil && err != io.EOF {
			log.Fatal(err)
		}

	}
	os.Rename(tmpfile.Name(), path)

	return nil
}

func listNotes(c *cli.Context) error {
	mark := "* "

	path := getLogPath()
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		if strings.HasPrefix(line, mark) {
			println(strings.TrimSuffix(line, "\n"))
		}
	}
	return nil
}

func listTasks(c *cli.Context) error {
	mark := "- "

	path := getLogPath()
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	lineNumber := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		if strings.HasPrefix(line, mark) {
			task := strings.TrimLeft(line, mark)
			fmt.Printf("%d: %s\n", lineNumber, strings.TrimSuffix(task, "\n"))
			lineNumber += 1
		}
	}

	return nil
}

func completeTask(c *cli.Context) error {
	taskNumber, err := strconv.Atoi(c.Args().First())
	if err != nil {
		return err
	}

	mark := "- "

	path := getLogPath()
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	tmpfile, err := ioutil.TempFile("", ".BULLETLOG.*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	lineNumber := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		if strings.HasPrefix(line, mark) {
			if taskNumber == lineNumber {
				task := strings.TrimLeft(line, mark)
				line = fmt.Sprintf("x %s", task)
			}
			lineNumber += 1
		}

		fmt.Fprintf(tmpfile, line)
	}
	os.Rename(tmpfile.Name(), path)

	return nil
}

func main() {
	app := &cli.App{
		Name:  "blt",
		Usage: "Take a log quickly like bullets.",
		Commands: []*cli.Command{
			{
				Name:    "add",
				Aliases: []string{"a", "note"},
				Usage:   "Add a note",
				Action:  addNote,
			},
			{
				Name:    "task",
				Aliases: []string{"t"},
				Usage:   "Add a task",
				Action:  addTask,
			},
			{
				Name:    "notes",
				Aliases: []string{"ls"},
				Usage:   "List notes",
				Action:  listNotes,
			},
			{
				Name:    "tasks",
				Aliases: []string{"ts"},
				Usage:   "List tasks",
				Action:  listTasks,
			},
			{
				Name:    "complete",
				Aliases: []string{"comp"},
				Usage:   "Complete task",
				Action:  completeTask,
			},
		},
	}
	app.Run(os.Args)
}
