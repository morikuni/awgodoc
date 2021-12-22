package awgodoc

import (
	"context"
	"fmt"
	"strings"

	"github.com/gocolly/colly"
	pkggodevclient "github.com/guseggert/pkggodev-client"
	"github.com/morikuni/failure"
)

type Client struct {
	client client
}

type client interface {
	Search(req pkggodevclient.SearchRequest) (*pkggodevclient.SearchResults, error)
}

func NewClient() *Client {
	return &Client{
		pkggodevclient.New(),
	}
}

type Package struct {
	Name              string
	ImportPath        string
	IsStandardLibrary bool
	URL               string
}

type Kind string

const (
	KindType     Kind = "type"
	KindConstant Kind = "constant"
	KindFunction Kind = "function"
	KindVariable Kind = "variable"
	KindMethod   Kind = "method"
)

type Symbol struct {
	Name              string
	Kind              Kind
	ImportPath        string
	IsStandardLibrary bool
	Code              string
	URL               string
}

func (cli *Client) getCollector() *colly.Collector {
	// TODO: set better UA.
	return colly.NewCollector(
		colly.UserAgent("github.com/morikuni/awgodoc"),
	)
}

func (cli *Client) SearchPackages(ctx context.Context, query string) ([]*Package, error) {
	query = strings.Replace(query, " ", "+", -1)

	c := cli.getCollector()

	var pkgs []*Package
	c.OnHTML(".SearchSnippet-headerContainer", func(e *colly.HTMLElement) {
		name := e.ChildText("a")
		importPath := e.ChildText(".SearchSnippet-header-path")
		isStandardLibrary := e.ChildText("span.go-Chip")

		name = name[:strings.LastIndex(name, importPath)]
		name = strings.TrimSpace(name)
		importPath = strings.TrimSpace(strings.Trim(importPath, "()"))

		pkgs = append(pkgs, &Package{
			Name:              name,
			ImportPath:        importPath,
			IsStandardLibrary: isStandardLibrary != "",
			URL:               fmt.Sprintf("https://pkg.go.dev/%s", importPath),
		})
	})

	err := c.Visit(fmt.Sprintf("https://pkg.go.dev/search?limit=10&q=%s&m=package", query))
	if err != nil {
		return nil, failure.Wrap(err)
	}

	return pkgs, nil
}

func (cli *Client) SearchSymbols(ctx context.Context, query string) ([]*Symbol, error) {
	query = strings.Replace(query, " ", "+", -1)

	c := cli.getCollector()

	var pkgs []*Symbol
	c.OnHTML(".SearchSnippet", func(e *colly.HTMLElement) {
		name := e.ChildText("a[data-test-id='snippet-title']")
		kind := e.ChildText("span.SearchSnippet-symbolKind")
		importPath := e.ChildText(".SearchSnippet-headerContainer a:not([data-test-id])")
		isStandardLibrary := e.ChildText("span.go-Chip")
		code := e.ChildText("pre.SearchSnippet-symbolCode")

		name = name[strings.Index(name, kind)+len(kind):]
		name = strings.TrimSpace(name)

		pkgs = append(pkgs, &Symbol{
			Name:              name,
			Kind:              Kind(kind),
			ImportPath:        importPath,
			IsStandardLibrary: isStandardLibrary != "",
			Code:              code,
			URL:               fmt.Sprintf("https://pkg.go.dev/%s#%s", importPath, name),
		})
	})

	err := c.Visit(fmt.Sprintf("https://pkg.go.dev/search?limit=10&q=%s&m=symbol", query))
	if err != nil {
		return nil, failure.Wrap(err)
	}

	return pkgs, nil
}
