package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"text/template"
)

var (
	regexpTemplate    = regexp.MustCompile(`{{.+}}`)
	regexpLastLine    = regexp.MustCompile(`},$`)
	regexpFirstSpaces = regexp.MustCompile(`^(\s+)`)
)

func main() {
	app := cli.NewApp()
	app.Name = "gsed"
	app.Usage = "strem editor written in Go"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "target",
			Value: "piyopiyo",
			Usage: "target file",
		},
		cli.StringFlag{
			Name:  "template",
			Value: "hogehoge",
			Usage: "template file path",
		},
		cli.StringFlag{
			Name:  "values",
			Value: "character",
			Usage: "insert character to template",
		},
	}
	app.Action = gsed

	app.Run(os.Args)
}

func gsed(c *cli.Context) error {
	// open target file
	target, err := os.Open(c.String("target"))
	if err != nil {
		fmt.Println(err)
	}
	defer target.Close()

	// open template file
	templateFile, err := os.Open(c.String("template"))
	if err != nil {
		fmt.Println(err)
	}
	defer templateFile.Close()

	// open values file
	values, err := ioutil.ReadFile(c.String("values"))
	if err != nil {
		fmt.Println(err)
	}

	// extract values
	var valuesMap map[string]string
	if err := yaml.Unmarshal(values, &valuesMap); err != nil {
		fmt.Println(err)
	}
	parseValuesMap := make(map[string]string)
	for key, val := range valuesMap {
		parseValuesMap[strings.Title(key)] = val
	}

	// parse template
	tpl := template.Must(template.New("template").ParseFiles(c.String("template")))
	var tplBuffer bytes.Buffer
	if err := tpl.Execute(&tplBuffer, parseValuesMap); err != nil {
		fmt.Println(err)
	}

	// create regexp for template
	templateRegexpStringArray, err := createRegexpStringArrayFromTemplate(templateFile)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(templateRegexpStringArray)

	// regexp compile for regTemplate
	templateRegexpArray := compileRegexpStringArray(templateRegexpStringArray)

	// insert values to lastline in target file
	outputBuffer, err := insertValuesLastlineInTargetFile(templateRegexpArray, tplBuffer)
	if err != nil {
		fmt.Println(err)
	}

	// insert template to target file
	output, err := os.Create(`/tmp/gsed/testdata/output.txt`)
	if err != nil {
		fmt.Println(err)
	}
	defer output.Close()

	outputBuffer.WriteTo(output)
	return nil
}

func createRegexpStringArrayFromTemplate(file *os.File) ([]string, error) {
	var regexpStringArray []string
	sc := bufio.NewScanner(file)
	for sc.Scan() {
		if err := sc.Err(); err != nil {
			return nil, err
		}
		rs := regexpTemplate.ReplaceAllString(sc.Text(), ".+")
		regexpStringArray = append(regexpStringArray, rs)
	}
	return regexpStringArray, nil
}

func compileRegexpStringArray(regexpStringArray []string) []*regexp.Regexp {
	var regexpArray []*regexp.Regexp
	for _, rs := range regexpStringArray {
		regexpArray = append(regexpArray, regexp.MustCompile(rs))
	}
	return regexpArray
}

func addLF(input string) string {
	return input + "\n"
}

func insertValuesLastlineInTargetFile() (bytes.Buffer, error) {
	sc := bufio.NewScanner(target)
	flags := [2]bool{false, false} // first is befor, second is now
	length := len(templateRegexpArray)
	var bufs []string // copy all target contents
	var lastLines = make([]string, length)
	var lastLineNumber int
	for ln := 0; sc.Scan(); ln++ {
		if err := sc.Err(); err != nil {
			fmt.Println(err)
			break
		}

		for i, rt := range templateRegexpArray {
			if !rt.MatchString(sc.Text()) {
				flags[0] = flags[1]
				flags[1] = false
				break
			}
			lastLines[i] = sc.Text()
			flags[0] = flags[1]
			flags[1] = true
			if i < length-1 {
				bufs = append(bufs, sc.Text())
				sc.Scan()
				ln++
				continue
			}
			lastLineNumber = ln
		}

		if flags == [2]bool{true, false} {
			// find last line
			tmp := bufs[lastLineNumber]
			bufs[lastLineNumber] = tmp + ","

			firstSpaces := []string{}
			// insert after last line
			for _, lastLine := range lastLines {
				firstSpaces = append(firstSpaces, regexpFirstSpaces.FindStringSubmatch(lastLine)[1])
			}
			scanner := bufio.NewScanner(&tplBuffer)
			for i := 0; scanner.Scan(); i++ {
				bufs = append(bufs, firstSpaces[i]+scanner.Text())
			}
		}
		bufs = append(bufs, sc.Text())
	}

	var outputBuffer bytes.Buffer
	for _, buf := range bufs {
		outputBuffer.WriteString(addLF(buf))
	}
}
