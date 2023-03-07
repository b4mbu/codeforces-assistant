package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
	"golang.design/x/clipboard"
)

const problemPrefixPattern = "/contest/%s/problem/"

type Problem struct {
	Number  string // or letter
	Samples []Sample
}

type Sample struct {
	Input  string
	Output string
}

type Verdict struct {
	OK                   bool
	TestNumber           int
	Input                string
	Output               string
	Answer               string
	LinesCorrectnessMask []bool
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

func testProgram(sourceFile string) (*Verdict, error) {
	res, err := os.Create("tmp-output.out")
	if err != nil {
		return nil, err
	}
	res.Close()
	defer os.Remove("tmp-output.out")

	files, err := ioutil.ReadDir("./")
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		test := 0
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".in") {
			test++
			cmd := exec.Command("bash", "-c", fmt.Sprintf("g++ %s && ./a.out < %s > tmp-output.out && rm ./a.out", sourceFile, file.Name()))
			cmd.Output()
			userAns, err := ioutil.ReadFile("tmp-output.out")
			if err != nil {
				return nil, err
			}
			rightAns, err := ioutil.ReadFile(strings.TrimSuffix(file.Name(), ".in") + ".out")
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
				return &Verdict{
					OK:                   false,
					TestNumber:           test,
					Input:                input,
					Output:               output,
					Answer:               answer,
					LinesCorrectnessMask: stringsMatchingMask(output, answer),
				}, nil
			}
		}
	}
	return &Verdict{OK: true}, nil
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

func printVerdict(verdict *Verdict) {
	if verdict.OK {
		color.Green("OK")
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
		verdict, err := testProgram(sourceFile)
		if err != nil {
			color.Red(err.Error())
			os.Exit(1)
		}
		printVerdict(verdict)
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
