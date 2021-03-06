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

	"github.com/julienschmidt/httprouter"
	"github.com/otiai10/copy"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

func getBranches(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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

	branch := plumbing.NewRemoteReferenceName("origin", name)
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
