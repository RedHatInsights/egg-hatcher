package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/julienschmidt/httprouter"
	"gopkg.in/src-d/go-git.v4"
)

const repoURL = "https://github.com/RedHatInsights/insights-core"

var repoPath = ""

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

		}
		if err != nil {
			log.Fatal(err)
		}
	}()

	r := httprouter.New()

	r.GET("/branch", getBranches)
	r.GET("/branch/:name", getBranch)
	r.GET("/tag", getTags)
	r.GET("/tag/:name", getTag)
	r.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		http.ServeFile(w, r, "./index.html")
	})

	log.Println("egg-hatcher now accepting connections on port 3000 ...")
	log.Fatal(http.ListenAndServe("localhost:3000", r))
}
