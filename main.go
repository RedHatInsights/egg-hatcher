package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/v47/github"
	"github.com/julienschmidt/httprouter"
)

const repoURL = "https://github.com/RedHatInsights/insights-core"

var repoPath string
var forksCache = make([]map[string]string, 0)
var forkCacheTimestamp = time.Now()
var lock sync.RWMutex

func getGithubForks() error {
	var err error
	client := github.NewClient(nil)

	ctx := context.Background()
	opt := &github.RepositoryListForksOptions{}
	repos, _, err := client.Repositories.ListForks(ctx, "RedHatInsights", "insights-core", opt)

	if err != nil {
		return err
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
	lock.Lock()
	defer lock.Unlock()
	forksCache = forks
	forkCacheTimestamp = time.Now()
	return err
}

func main() {
	var err error

	repoPath, err = os.MkdirTemp("", "egg-hatcher-")
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

	if err := getGithubForks(); err != nil {
		log.Fatalf("error creating github forks cache: %v", err)
	} else {
		log.Println("forked repo cache created")
	}

	go func() {
		var err error
		for {
			time.Sleep(5 * time.Minute)

			timeNow := time.Now()
			timeElapse := timeNow.Sub(forkCacheTimestamp).Hours()
			if timeElapse > 1 {
				log.Println("refreshing fork cache")
				if err := getGithubForks(); err != nil {
					log.Fatalf("error creating github forks cache: %v", err)
					continue
				}
			}

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
	r.GET("/branch", getBranches)
	r.GET("/branch/:name", getBranch)
	r.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		http.ServeFile(w, r, "./index.html")
	})

	log.Println("egg-hatcher now accepting connections on port 3000 ...")
	log.Fatal(http.ListenAndServe("localhost:3000", r))
}
