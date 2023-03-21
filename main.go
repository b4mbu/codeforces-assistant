package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/cheggaaa/pb/v3"
	"github.com/fatih/color"
	"golang.design/x/clipboard"
)

const (
	problemPrefixPattern = "/contest/%s/problem/"
	configName           = "acf-config.json"
)

var (
	config = Config{
		Compiler: "g++",
		Standart: "c++17",
	}
)

type Config struct {
	Compiler string `json:"compiler"`
	Standart string `json:"standart"`
}

type Verdict struct {
	/* Solution verdict for local test */
	OK                   bool // OK or WA
	TestNumber           int
	Input                string
	Output               string
	Answer               string
	AverageExecutingTime []time.Duration
	LinesCorrectnessMask []bool
}

type Problem struct {
	Number  string // or letter
	Samples []Sample
}

type Sample struct {
	Input  string
	Output string
}

func loadConfig() {
	dir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	rawConfig, err := os.ReadFile(dir + "/" + configName)
	if err != nil {
		return
	}
	json.Unmarshal(rawConfig, &config)
}

func getProblemsPath(contestNumber string) ([]string, error) {
	url := fmt.Sprintf("https://codeforces.com/contest/%s", contestNumber)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New("invalid contest number")
	}
	var problemPaths []string

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	was := make(map[string]bool)

	filter := func(i int, s *goquery.Selection) bool {
		href, ok := s.Attr("href")
		if !ok {
			return false
		}
		if strings.HasPrefix(href, fmt.Sprintf(problemPrefixPattern, contestNumber)) && !was[href] {
			was[href] = true
			return true
		}
		return false
	}

	doc.Find("a").FilterFunction(filter).Each(func(i int, s *goquery.Selection) {
		path, _ := s.Attr("href")
		problemPaths = append(problemPaths, path)
	})
	return problemPaths, nil
}

func getProblem(contestNumber, problemNumber string) (*Problem, error) {
	url := fmt.Sprintf("https://codeforces.com/contest/%s/problem/%s", contestNumber, problemNumber)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New("invalid contest number or problem number")
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	samples := make([]Sample, doc.Find(".sample-test").Find(".input").Length())

	doc.Find(".sample-test").Find(".input").Each(func(i int, s *goquery.Selection) {
		pre := s.Find("pre")
		if pre.Find("div").Length() > 0 {
			pre.Find("div").Each(func(j int, sDiv *goquery.Selection) {
				samples[i].Input += sDiv.Text() + "\n"
			})
		} else {
			samples[i].Input = pre.Text()
		}
	})

	doc.Find(".sample-test").Find(".output").Each(func(i int, s *goquery.Selection) {
		pre := s.Find("pre")
		if pre.Find("div").Length() > 0 {
			pre.Find("div").Each(func(j int, sDiv *goquery.Selection) {
				samples[i].Output += sDiv.Text() + "\n"
			})
		} else {
			samples[i].Output = pre.Text()
		}
	})

	return &Problem{
		Number:  problemNumber,
		Samples: samples,
	}, nil
}

func loadContest(contestNumber string) error {
	problemsPaths, err := getProblemsPath(contestNumber)
	if err != nil {
		return err
	}

	for _, path := range problemsPaths {
		number := strings.TrimPrefix(path, fmt.Sprintf(problemPrefixPattern, contestNumber))
		problem, err := getProblem(contestNumber, number)
		if err != nil {
			return err
		}
		if err = createIOFiles(problem); err != nil {
			return err
		}
	}

	return nil
}

func createIOFiles(problem *Problem) error {
	if err := os.Mkdir(problem.Number, os.ModePerm); err != nil {
		return err
	}

	for i, sample := range problem.Samples {
		fin, err := os.Create(fmt.Sprintf("./%s/%d.in", problem.Number, i+1))
		if err != nil {
			return err
		}
		defer fin.Close()

		fout, err := os.Create(fmt.Sprintf("./%s/%d.out", problem.Number, i+1))
		if err != nil {
			return err
		}
		defer fout.Close()

		fin.Write([]byte(sample.Input))
		fout.Write([]byte(sample.Output))
	}

	return nil
}

func testSolution(sourceFile string, bench int) (*Verdict, error) {
	outfile := "tmp-output.out"
	res, err := os.Create(outfile)
	if err != nil {
		return nil, err
	}
	res.Close()
	defer os.Remove(outfile)

	files, err := ioutil.ReadDir("./")
	if err != nil {
		return nil, err
	}

	inputFilesCount := 0
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".in") {
			inputFilesCount++
		}
	}

	averageExecutingTime := make([]int64, inputFilesCount)
	execute := func(filename, infile, outfile string) (time.Duration, error) {
		compile := func(filename, infile, outfile string) (string, string, string, error) {
			cmd := exec.Command("bash", "-c", fmt.Sprintf("%s --std=%s %s", config.Compiler, config.Standart, filename))
			if _, err := cmd.Output(); err != nil {
				return "", "", "", errors.New("error while compiling")
			}
			return "./a.out", infile, outfile, nil
		}

		run := func(filename, infile, outfile string, err error) (string, time.Duration, error) {
			if err != nil {
				return "", 0, err
			}

			startTime := time.Now()
			cmd := exec.Command("bash", "-c", fmt.Sprintf("%s < %s > %s", filename, infile, outfile))
			if _, err := cmd.Output(); err != nil {
				return "", 0, errors.New("error while running")
			}
			finishTime := time.Now()
			return filename, finishTime.Sub(startTime), nil
		}

		remove := func(filename string, executingTime time.Duration, err error) (time.Duration, error) {
			if err != nil {
				return 0, err
			}

			cmd := exec.Command("bash", "-c", fmt.Sprintf("rm %s", filename))
			if _, err := cmd.Output(); err != nil {
				return 0, errors.New("error while removing")
			}
			return executingTime, nil
		}

		return remove(run(compile(filename, infile, outfile)))
	}

	bar := pb.StartNew(bench * inputFilesCount)

	for i := 0; i < bench; i++ {
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".in") {
				test, err := strconv.Atoi(strings.TrimSuffix(file.Name(), ".in"))

				if err != nil {
					return nil, err
				}
				executingTime, err := execute(sourceFile, file.Name(), outfile)
				averageExecutingTime[test-1] += executingTime.Nanoseconds()
				if err != nil {
					return nil, err
				}

				userAns, err := ioutil.ReadFile(outfile)
				if err != nil {
					return nil, err
				}

				rightAns, err := ioutil.ReadFile(strings.Replace(file.Name(), ".in", ".out", 1))
				if err != nil {
					return nil, err
				}

				inputBytes, err := ioutil.ReadFile(file.Name())
				if err != nil {
					return nil, err
				}

				var (
					input  = strings.Trim(string(inputBytes), " \n\t")
					output = strings.Trim(string(userAns), " \n\t")
					answer = strings.Trim(string(rightAns), " \n\t")
				)

				if output != answer {
					bar.Finish()
					return &Verdict{
						OK:                   false,
						TestNumber:           test,
						Input:                input,
						Output:               output,
						Answer:               answer,
						LinesCorrectnessMask: stringsMatchingMask(output, answer),
					}, nil
				}
				bar.Increment()
			}
		}
	}
	bar.Finish()

	resTime := make([]time.Duration, len(averageExecutingTime))
	for i, t := range averageExecutingTime {
		resTime[i] = time.Nanosecond * time.Duration(t/int64(bench))
	}
	return &Verdict{OK: true, AverageExecutingTime: resTime}, nil
}

func stringsMatchingMask(a, b string) []bool {
	var (
		aLines    = strings.Split(a, "\n")
		bLines    = strings.Split(b, "\n")
		minLength int
		maxLength int
	)

	if len(aLines) < len(bLines) {
		minLength = len(aLines)
		maxLength = len(bLines)
	} else {
		minLength = len(bLines)
		maxLength = len(bLines)
	}

	res := make([]bool, maxLength)

	for i := 0; i < minLength; i++ {
		res[i] = aLines[i] == bLines[i]
	}
	return res
}

func printVerdict(verdict *Verdict, bench bool) {
	if verdict.OK {
		color.Green("OK")
		if bench {
			fmt.Println("Average Executing Time:")
			for i, t := range verdict.AverageExecutingTime {
				fmt.Printf("Test #%d: %s\n", i+1, t.String())
			}
		}
	} else {
		color.Red(fmt.Sprintf("Wrong answer at test #%d\n", verdict.TestNumber))

		fmt.Printf("Input:\n%s\n", verdict.Input)
		fmt.Println()

		fmt.Println("Output:")
		outputLines := strings.Split(verdict.Output, "\n")
		for i, line := range outputLines {
			if verdict.LinesCorrectnessMask[i] {
				color.Green(line)
			} else {
				color.Red(line)
			}
		}
		fmt.Println()

		fmt.Printf("Answer:\n%s\n", verdict.Answer)
	}
}

func writeFileToClipboard(filename string) error {
	text, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return writeToClipboard(text)
}

func writeToClipboard(s []byte) error {
	if err := clipboard.Init(); err != nil {
		return err
	}

	clipboard.Write(clipboard.FmtText, s)
	return nil
}

func init() {
	loadConfig()
}

func main() {
	color.Output = os.Stdout
	if len(os.Args) < 2 {
		color.Red("Not enough arguments")
		os.Exit(1)
	}
	command := os.Args[1]
	switch command {
	case "contest":
		if len(os.Args) < 3 {
			color.Red("No contest number was provided")
			os.Exit(1)
		}
		contestNumber := os.Args[2]
		if err := loadContest(contestNumber); err != nil {
			color.Red(err.Error())
			os.Exit(1)
		}
		color.Green(fmt.Sprintf("Contest %s was loaded.\nGood luck!", contestNumber))
	case "test":
		if len(os.Args) < 3 {
			color.Red("Source file must be provided")
			os.Exit(1)
		}
		sourceFile := os.Args[2]

		benchCount := 1
		if len(os.Args) == 5 {
			if os.Args[3] == "-b" {
				n, err := strconv.Atoi(os.Args[4])
				if err == nil {
					benchCount = n
				}
			}
		}

		verdict, err := testSolution(sourceFile, benchCount)
		if err != nil {
			color.Red(err.Error())
			os.Exit(1)
		}
		printVerdict(verdict, benchCount > 0)
		if !verdict.OK {
			os.Exit(1)
		}
	case "copy":
		if len(os.Args) < 3 {
			color.Red("Source file must be provided")
			os.Exit(1)
		}
		sourceFile := os.Args[2]
		if err := writeFileToClipboard(sourceFile); err != nil {
			color.Red(err.Error())
			os.Exit(1)
		}
		color.Green(fmt.Sprintf("File %s was copied to clipboard", sourceFile))
	}
}
