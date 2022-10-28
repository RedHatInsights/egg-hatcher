package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/julienschmidt/httprouter"
	"github.com/otiai10/copy"
)

func getBranches(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	var err error

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	remote_name := p.ByName("forkname")
	if remote_name == "" {
		remote_name = "origin"
	}
	remote, err := repo.Remote(remote_name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	refs, err := remote.List(&git.ListOptions{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	branches := make([]map[string]string, 0)
	for _, ref := range refs {
		if ref.Name().IsBranch() {
			branch := map[string]string{
				"name":       ref.Name().Short(),
				"fullBranch": remote.Config().Name + "/" + ref.Name().Short(),
			}
			branches = append(branches, branch)
		}
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}
	sort.Slice(branches, func(i, j int) bool {
		return branches[i]["name"] < branches[j]["name"]
	})

	data, err := json.Marshal(&branches)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}
	w.Write(data)
}

func getCacheForks(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	lock.RLock()
	defer lock.RUnlock()
	data, err := json.Marshal(&forksCache)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}
	w.Write(data)
}

func getTags(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var err error

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	remote, err := repo.Remote("origin")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	refs, err := remote.List(&git.ListOptions{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	tags := make([]map[string]string, 0)
	for _, ref := range refs {
		if ref.Name().IsTag() {
			tag := map[string]string{
				"fullTag": ref.Name().Short(),
				"name":    strings.TrimPrefix(strings.TrimPrefix(ref.Name().Short(), "insights-core-"), "falafel-"),
			}
			tags = append(tags, tag)
		}
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}
	sort.Slice(tags, func(i, j int) bool {
		return tags[i]["name"] < tags[j]["name"]
	})

	data, err := json.Marshal(&tags)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}
	w.Write(data)
}

func getBranchesfromFork(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	var err error

	name := p.ByName("forkname")
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "missing required parameter: name")
		return
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}
	// Check if remote exists
	remotes, err := repo.Remotes()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	remote_exists := false
	for _, remote := range remotes {
		if remote.Config().Name == name {
			//change the remote
			remote_exists = true
		}
	}

	//create a new remote
	if !remote_exists {
		url := "https://github.com/" + name + "/insights-core"
		config := &config.RemoteConfig{
			Name: name,
			URLs: []string{url},
		}

		_, err = repo.CreateRemote(config)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%v", err)
			return
		}

		err = repo.Fetch(&git.FetchOptions{RemoteName: name})
		if err != nil {
			if err != git.NoErrAlreadyUpToDate {
				fmt.Fprintf(w, "%v", err)
			}
		}
	}

	getBranches(w, r, p)
}

func getBranch(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	var err error

	name := p.ByName("name")
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "missing required parameter: name")
		return
	}

	dir, err := ioutil.TempDir("", "egg-hatcher-")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	err = copy.Copy(repoPath, dir)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}
	defer os.RemoveAll(dir)

	repo, err := git.PlainOpen(dir)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	wt, err := repo.Worktree()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	actual_remote := p.ByName("forkname")
	if actual_remote == "" {
		actual_remote = "origin"
	}

	branch := plumbing.NewRemoteReferenceName(actual_remote, name)
	err = wt.Checkout(&git.CheckoutOptions{
		Branch: branch,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	cmd := exec.Command("./build_client_egg.sh")
	cmd.Dir = dir
	err = cmd.Run()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	f, err := os.Open(filepath.Join(dir, "insights.zip"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	head, err := repo.Head()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"insights-core-%v-%v.egg\"", strings.TrimPrefix(branch.Short(), actual_remote+"/"), head.Hash()))
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(data)), 10))
	w.Write(data)
}

func getTag(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	var err error

	name := p.ByName("name")
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "missing required parameter: name")
		return
	}

	dir, err := ioutil.TempDir("", "egg-hatcher-")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	err = copy.Copy(repoPath, dir)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}
	defer os.RemoveAll(dir)

	repo, err := git.PlainOpen(dir)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	wt, err := repo.Worktree()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	branch := plumbing.NewTagReferenceName(name)
	err = wt.Checkout(&git.CheckoutOptions{
		Branch: branch,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	cmd := exec.Command("./build_client_egg.sh")
	cmd.Dir = dir
	err = cmd.Run()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	f, err := os.Open(filepath.Join(dir, "insights.zip"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}
	head, err := repo.Head()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"insights-core-%v-%v.egg\"", strings.TrimPrefix(branch.Short(), "origin/"), head.Hash()))
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(data)), 10))
	w.Write(data)
}
