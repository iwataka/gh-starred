// TODO: interactive shell with input completion

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/c-bata/go-prompt"
	"github.com/cli/go-gh"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
)

const (
	COMMAND_NAME = "gh-starred"
)

type repository struct {
	Name     string   `json:"name"`
	FullName string   `json:"full_name"`
	Topics   []string `json:"topics"`
	HtmlURL  string   `json:"html_url"`
}

func main() {
	app := &cli.App{
		Name:  COMMAND_NAME,
		Usage: "make operations about your starred repositories",
		Commands: []*cli.Command{
			{
				Name:  "repos",
				Usage: "list your starred repositories",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:    "topics",
						Usage:   "topics to filter repositories",
						Aliases: []string{"t"},
					},
				},
				Action: repos,
			},
			{
				Name:   "topics",
				Usage:  "list topics in your starred repositories",
				Action: topics,
			},
			{
				Name:   "shell",
				Usage:  "activate interactive shell mode",
				Action: shell,
			},
		},
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "batch-size",
				Usage: "batch size to retrieve your starred repository",
				Value: 5,
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func repos(ctx *cli.Context) error {
	starredRepos, err := getRepos(ctx.Int("batch-size"))
	if err != nil {
		return err
	}

	topics := ctx.StringSlice("topics")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "URL"})
	for _, repo := range starredRepos {
		shouldPrint := false
		if len(topics) == 0 {
			shouldPrint = true
		} else {
			for _, t1 := range repo.Topics {
				for _, t2 := range topics {
					if t1 == t2 {
						shouldPrint = true
						break
					}
				}
				if shouldPrint {
					break
				}
			}
		}
		if shouldPrint {
			table.Append([]string{repo.Name, repo.HtmlURL})
		}
	}

	table.Render()
	return nil
}

func topics(ctx *cli.Context) error {
	starredRepos, err := getRepos(ctx.Int("batch-size"))
	if err != nil {
		return err
	}

	topicExists := map[string]bool{}
	for _, repo := range starredRepos {
		for _, topic := range repo.Topics {
			topicExists[topic] = true
		}
	}

	for topic := range topicExists {
		fmt.Println(topic)
	}

	return nil
}

func shell(ctx *cli.Context) error {
	executer := AppExecuter{ctx.App}
	completer := AppCompleter{ctx.App}
	p := prompt.New(executer.execute, completer.complete, prompt.OptionPrefix(fmt.Sprintf("%s> ", COMMAND_NAME)))
	p.Run()
	return nil
}

type AppExecuter struct {
	app *cli.App
}

func (e *AppExecuter) execute(in string) {
	args := []string{COMMAND_NAME}
	args = append(args, strings.Fields(in)...)
	e.app.Run(args) //nolint:errcheck
}

type AppCompleter struct {
	app *cli.App
}

func (c *AppCompleter) complete(in prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{}
}

func getRepos(batchSize int) ([]repository, error) {
	perPage := 100

	starredRepos := []repository{}

	for i := 1; ; i += batchSize {
		var repos []repository
		var err error
		if batchSize == 1 {
			repos, err = getReposPerPage(i, perPage)
		} else {
			repos, err = getReposPerPageBatch(i, perPage, batchSize)
		}
		if err != nil {
			return nil, err
		}
		starredRepos = append(starredRepos, repos...)
		if len(repos) < perPage*batchSize {
			break
		}
	}

	return starredRepos, nil
}

func getReposPerPageBatch(page, perPage, batchSize int) ([]repository, error) {
	var wg sync.WaitGroup
	result := []repository{}
	for i := 0; i < batchSize; i++ {
		wg.Add(1)
		go func(page, perPage int) {
			defer wg.Done()
			repos, err := getReposPerPage(page, perPage)
			if err != nil {
				log.Fatalln(err)
			}
			result = append(result, repos...)
		}(page+i, perPage)
	}
	wg.Wait()
	return result, nil
}

func getReposPerPage(page, perPage int) ([]repository, error) {
	args := []string{
		"api",
		fmt.Sprintf("user/starred?page=%d&per_page=%d", page, perPage),
	}
	stdOut, _, err := gh.Exec(args...)
	if err != nil {
		return nil, err
	}
	var repos []repository
	err = json.Unmarshal(stdOut.Bytes(), &repos)
	if err != nil {
		return nil, err
	}

	return repos, nil
}
