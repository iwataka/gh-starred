// TODO: interactive shell with input completion

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/olekukonko/tablewriter"
	"github.com/cli/go-gh"
	"github.com/urfave/cli/v2"
)

type repository struct {
	Name     string   `json:"name"`
	FullName string   `json:"full_name"`
	Topics   []string `json:"topics"`
	HtmlURL  string   `json:"html_url"`
}

func main() {
	app := &cli.App{
		Name:  "gh-starred",
		Usage: "make operations about your starred repositories",
		Commands: []*cli.Command{
			{
				Name:  "repos",
				Usage: "list your starred repositories",
				Flags: []cli.Flag{
					&cli.StringFlag{
						// TODO: accept multiple topics
						Name:  "topic",
						Usage: "topic to filter repositories",
					},
				},
				Action: repos,
			},
			{
				Name:   "topics",
				Usage:  "list topics in your starred repositories",
				Action: topics,
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

	topic := ctx.String("topic")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "URL"})
	for _, repo := range starredRepos {
		shouldPrint := false
		if topic == "" {
			shouldPrint = true
		} else {
			for _, t := range repo.Topics {
				if t == topic {
					shouldPrint = true
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
