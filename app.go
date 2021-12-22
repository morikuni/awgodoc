package awgodoc

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
	"unicode"

	aw "github.com/deanishe/awgo"
	"github.com/morikuni/failure"
	"golang.org/x/sync/errgroup"
)

type App struct {
	wf     *aw.Workflow
	client *Client
}

func NewApp() *App {
	return &App{
		aw.New(),
		NewClient(),
	}
}

func (app *App) Run() {
	app.wf.Run(func() {
		err := app.run()
		if err != nil {
			log.Println(err)
			return
		}
	})
}

func (app *App) run() error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	args := app.wf.Args()
	query := strings.Join(args, " ")
	log.Println(query)

	eg, ctx := errgroup.WithContext(ctx)

	var (
		syms []*Symbol
		pkgs []*Package
	)
	eg.Go(func() error {
		var err error
		syms, err = app.client.SearchSymbols(ctx, query)
		if err != nil {
			return failure.Wrap(err)
		}
		return nil
	})

	eg.Go(func() error {
		var err error
		pkgs, err = app.client.SearchPackages(ctx, query)
		if err != nil {
			return failure.Wrap(err)
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return failure.Wrap(err)
	}

	standardText := func(isStandardLibrary bool) string {
		if isStandardLibrary {
			return " [✅ Standard Library]"
		}
		return ""
	}

	type Item struct {
		Name              string
		ImportPath        string
		IsStandardLibrary bool
		IsSymbol          bool

		Title    string
		Subtitle string
		URL      string
	}
	items := make([]*Item, 0, len(pkgs)+len(syms))
	for _, pkg := range pkgs {
		items = append(items, &Item{
			Name:              pkg.Name,
			ImportPath:        pkg.ImportPath,
			IsStandardLibrary: pkg.IsStandardLibrary,

			Title: fmt.Sprintf("%s (%s)%s", pkg.Name, pkg.ImportPath, standardText(pkg.IsStandardLibrary)),
			URL:   pkg.URL,
		})
	}
	for _, sym := range syms {
		items = append(items, &Item{
			Name:              sym.Name,
			ImportPath:        sym.ImportPath,
			IsStandardLibrary: sym.IsStandardLibrary,
			IsSymbol:          true,

			Title:    fmt.Sprintf("%s (%s)%s", sym.Name, sym.ImportPath, standardText(sym.IsStandardLibrary)),
			Subtitle: sym.Code,
			URL:      sym.URL,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		ii, ij := items[i], items[j]

		if ii.ImportPath != ij.ImportPath {
			target := strings.ToLower(strings.Join(args, "/"))
			iPath := strings.ToLower(ii.ImportPath)
			jPath := strings.ToLower(ij.ImportPath)
			if strings.Contains(iPath, target) {
				return true
			}
			if strings.Contains(jPath, target) {
				return false
			}
		}

		containsUpper := func(s string) bool {
			for _, r := range s {
				if unicode.IsUpper(r) {
					return true
				}
			}
			return false
		}
		if containsUpper(query) {
			// Is the query contains upper case, assume that the user is searching a symbol.
			if !ij.IsSymbol {
				return true
			}
			return ii.IsSymbol
		}

		if ii.Name != ij.Name {
			iL := strings.ToLower(ii.Name)
			jL := strings.ToLower(ij.Name)
			for i := len(args) - 1; i >= 0; i-- {
				arg := args[i]
				argL := strings.ToLower(arg)
				if ii.Name == arg {
					return true
				}
				if ij.Name == arg {
					return false
				}
				if iL == argL {
					return true
				}
				if jL == argL {
					return false
				}
			}
		}

		if ii.IsStandardLibrary && !ij.IsStandardLibrary {
			return true
		}
		if !ii.IsStandardLibrary && ij.IsStandardLibrary {
			return false
		}

		return true
	})

	for _, item := range items {
		i := app.wf.NewItem(item.Title).
			Valid(true).
			Arg(item.URL)
		if item.Subtitle != "" {
			i.Subtitle(item.Subtitle)
		}
	}

	app.wf.SendFeedback()
	return nil
}
