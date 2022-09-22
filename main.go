package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/v47/github"
	"github.com/julienschmidt/httprouter"
)

const repoURL = "https://github.com/RedHatInsights/insights-core"

var repoPath = ""

var forks_cache = make([]map[string]string, 0)

func getGithubForks() {
	var err error
	forks_cache = nil
	client := github.NewClient(nil)

	ctx := context.Background()
	opt := &github.RepositoryListForksOptions{}
	repos, _, err := client.Repositories.ListForks(ctx, "RedHatInsights", "insights-core", opt)

	if err != nil {
		log.Fatal(err)
	}

	forks := make([]map[string]string, 0)
	fork := map[string]string{
		"fullName": "RedHatInsights/insights-core",
		"name":     "RedHatInsights",
	}
	forks = append(forks, fork)
	for _, repos := range repos {
		fork := map[string]string{
			"fullName": repos.GetFullName(),
			"name":     *repos.GetOwner().Login,
		}
		forks = append(forks, fork)

	}
	forks_cache = forks
}

func main() {
	var err error

	repoPath, err = ioutil.TempDir("", "egg-hatcher-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(repoPath)

	log.Println("git clone " + repoURL + " " + repoPath)
	repo, err := git.PlainClone(repoPath, false, &git.CloneOptions{
		URL: repoURL,
	})
	if err != nil {
		log.Fatal(err)
	}

	err = repo.Fetch(&git.FetchOptions{})
	if err != nil {
		if err != git.NoErrAlreadyUpToDate {
			log.Fatal(err)
		}
	}

	getGithubForks()

	go func() {
		var err error
		for {
			time.Sleep(5 * time.Minute)

			var repo *git.Repository
			repo, err = git.PlainOpen(repoPath)
			if err != nil {
				break
			}

			log.Println("git fetch")
			err := repo.Fetch(&git.FetchOptions{})
			if err != nil {
				if err == git.NoErrAlreadyUpToDate {
					continue
				}
				log.Printf("Error: %v", err)
				continue
			}

			getGithubForks()

		}
		if err != nil {
			log.Fatal(err)
		}
	}()

	r := httprouter.New()

	r.GET("/fork", getCacheForks)
	r.GET("/fork/:forkname/branch", getBranchesfromFork)
	r.GET("/fork/:forkname/branch/:name", getBranch)
	r.GET("/tag", getTags)
	r.GET("/tag/:name", getTag)
	r.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		http.ServeFile(w, r, "./index.html")
	})

	log.Println("egg-hatcher now accepting connections on port 3000 ...")
	log.Fatal(http.ListenAndServe("localhost:3000", r))
}
