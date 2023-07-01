package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

var (
	assets       []string
	excludePath  string
	files        FilePair
	packageName  string
	templateFile FileContent
)

type Link struct {
	Name string
	Href string
}

type TemplateData struct {
	Title    string
	Heading  string
	Content  string
	NavLinks []Link
	Package  string
	Css      []string
}

func main() {
	parseFlags()

	t, err := template.New("page").Parse(templateFile.String())

	if err != nil {
		log.Fatal(err)
	}

	var templates = make(map[string]TemplateData)
	var nav []Link

	for source, target := range files {
		content, err := ioutil.ReadFile(source)

		if err != nil {
			log.Fatal(fmt.Sprintf("Error processing file %s", source), err)
		}

		doc := parseMarkdown(content)
		html := renderHTML(doc)
		heading := getFirstHeading(doc)
		basename := filepath.Base(target)

		if basename == "index.html" {
			heading = "README"
		}

		title := fmt.Sprintf("bzlparty - %s", heading)

		if packageName != "" {
			title = fmt.Sprintf("bzlparty - %s - %s", packageName, heading)
		}

		var data = TemplateData{
			Title:   title,
			Content: string(html),
			Heading: heading,
		}

		templates[target] = data
		if packageName != "" && heading != "README" {
			nav = append(nav, Link{Href: strings.TrimPrefix(target, excludePath), Name: heading})
		}
	}

	sort.Slice(nav, func(i, j int) bool {
		return nav[i].Name < nav[j].Name
	})

	for target, data := range templates {
		data.NavLinks = nav
		data.Css = assets
		data.Package = packageName

		file, err := os.Create(target)

		if err = t.Execute(file, data); err != nil {
			log.Fatal(fmt.Sprintf("Could not write content to file %s", target))
		}
	}
}

func parseFlags() {
	flag.Func("assets", "Asset files", func(s string) error {
		assets = strings.Split(s, ",")

		if len(assets) == 0 {
			return errors.New("No assets given")
		}

		return nil
	})
	flag.Var(&files, "files", "Source files")
	flag.Var(&templateFile, "template", "Template")
	flag.StringVar(&excludePath, "exclude-path", "", "exclude from path")
	flag.StringVar(&packageName, "package", "", "package name")
	flag.Parse()
}

type FilePair map[string]string

func (v *FilePair) String() string {
	var r []string

	for source, target := range *v {
		r = append(r, fmt.Sprintf("%s:%s", source, target))
	}

	return strings.Join(r, ",")
}

func (v *FilePair) Set(s string) error {
	r := make(map[string]string)

	for _, f := range strings.Split(s, ",") {
		source, target, err := splitFiles(f)

		if err != nil {
			return err
		}

		r[source] = target
	}

	if len(r) == 0 {
		return errors.New("No files given")
	}

	*v = r

	return nil
}

type FileContent []byte

func (v *FileContent) Set(path string) (err error) {
	*v, err = ioutil.ReadFile(path)

	return
}

func (v *FileContent) String() string {
	return string(*v)
}

func splitFiles(s string) (string, string, error) {
	x := strings.Split(s, ":")

	if len(x) < 2 {
		return "", "", errors.New("File pair uncomplete")
	}

	return x[0], x[1], nil
}

func getFirstHeading(doc ast.Node) (heading string) {
	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		if h, ok := node.(*ast.Heading); ok && entering {
			if h.Level == 1 {
				heading = string(h.Children[0].AsLeaf().Literal)
				return ast.Terminate
			}
		}

		return ast.GoToNext
	})

	return
}

func parseMarkdown(md []byte) ast.Node {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)
	return p.Parse(md)
}

func renderHTML(doc ast.Node) []byte {
	htmlFlags := html.CommonFlags
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}
