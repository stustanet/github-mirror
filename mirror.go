package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

type config struct {
	GithubToken string `json:"github_token"`
	GitlabURL   string `json:"gitlab_url"`
	OrgName     string `json:"org_name"`
	UserAgent   string `json:"user_agent"`
}

func (c *config) parseFile(filepath string) (err error) {
	var f *os.File
	if f, err = os.Open(filepath); err != nil {
		return
	}
	err = json.NewDecoder(f).Decode(&c)
	f.Close()
	return
}

var cfg config

type githubRepos []struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Path        string    `json:"path"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type gitlabRepos []struct {
	Name              string    `json:"name"`
	NameWithNamespace string    `json:"name_with_namespace"`
	Description       string    `json:"description"`
	Path              string    `json:"path"`
	CreatedAt         time.Time `json:"created_at"`
	LastActivityAt    time.Time `json:"last_activity_at"`
}

type createRepo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Homepage    string `json:"homepage"`
	HasIssues   bool   `json:"has_issues"`
	HasProjects bool   `json:"has_projects"`
	HasWiki     bool   `json:"has_wiki"`
}

type updateRepo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func setGithubHeaders(r *http.Request) {
	r.Header.Set("Accept", "application/vnd.github.v3+json")
	r.Header.Set("Authorization", "token "+cfg.GithubToken)
	r.Header.Set("User-Agent", cfg.UserAgent)
}

func getGithub(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/"+path, nil)
	if err != nil {
		return nil, err
	}
	setGithubHeaders(req)
	return http.DefaultClient.Do(req)
}

func sendGithub(method, path string, jsonData interface{}) (*http.Response, error) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(jsonData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, "https://api.github.com/"+path, &buf)
	if err != nil {
		return nil, err
	}
	setGithubHeaders(req)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	return http.DefaultClient.Do(req)
}

func getGithubRepos() (repos githubRepos, err error) {
	page := 1
	for {
		var newRepos githubRepos
		resp, err := getGithub("orgs/" + cfg.OrgName +
			"/repos?type=public&per_page=100&page=" + strconv.Itoa(page))
		if err != nil {
			return nil, err
		}
		if err = json.NewDecoder(resp.Body).Decode(&newRepos); err != nil {
			return nil, err
		}
		if len(newRepos) < 100 {
			if page == 1 {
				return newRepos, nil
			} else {
				return append(repos, newRepos...), nil
			}
		} else {
			repos = append(repos, newRepos...)
			page++
		}
	}
}

func getGitlabRepos() (repos gitlabRepos, err error) {
	page := 1
	for {
		var newRepos gitlabRepos
		resp, err := http.Get(cfg.GitlabURL + "api/v4/groups/" + cfg.OrgName +
			"/projects?visibility=public&simple=1&per_page=100&page=" + strconv.Itoa(page))
		if err != nil {
			return nil, err
		}
		if err = json.NewDecoder(resp.Body).Decode(&newRepos); err != nil {
			return nil, err
		}
		if len(newRepos) < 100 {
			if page == 1 {
				return newRepos, nil
			} else {
				return append(repos, newRepos...), nil
			}
		} else {
			repos = append(repos, newRepos...)
			page++
		}
	}
	return
}

func createGithubRepo(name, description, path string) error {
	cr := createRepo{
		Name:        name,
		Description: description,
		Homepage:    cfg.GitlabURL + cfg.OrgName + "/" + name,
		HasIssues:   true,
		HasProjects: false,
		HasWiki:     false,
	}
	resp, err := sendGithub("POST", "orgs/"+cfg.OrgName+"/repos", cr)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != 201 {
		return fmt.Errorf("unexpected response: %d %s", resp.StatusCode, resp.Status)
	}
	return pushGithubRepo(name, path)
}

func updateGithubRepo(name, description, path string) error {
	ur := updateRepo{
		Name:        name,
		Description: description,
	}
	resp, err := sendGithub("PATCH", "repos/"+cfg.OrgName+"/"+name, ur)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected response: %s", resp.Status)
	}
	return pushGithubRepo(name, path)
}

func pushGithubRepo(name, path string) error {
	cmd := exec.Command("git", "push", "--mirror", "https://"+cfg.GithubToken+"@github.com/"+cfg.OrgName+"/"+name+".git")
	cmd.Dir = "/var/opt/gitlab/git-data/repositories/" + cfg.OrgName + "/" + path + ".git/"
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", stdoutStderr)
	return nil
}

func main() {
	err := cfg.parseFile("/etc/github-mirror.json")
	if err != nil {
		log.Fatal(err)
	}

	glr, err := getGitlabRepos()
	if err != nil {
		log.Fatal(err)
	}
	sort.Slice(glr, func(i, j int) bool { return glr[i].Name < glr[j].Name })

	ghr, err := getGithubRepos()
	if err != nil {
		log.Fatal(err)
	}
	sort.Slice(ghr, func(i, j int) bool { return ghr[i].Name < ghr[j].Name })

	i := 0
	for _, repo := range glr {
		// skip repos which have no activity yet or are just shared with the group, not owned by it
		if repo.LastActivityAt == repo.CreatedAt ||
			!strings.HasPrefix(repo.NameWithNamespace, cfg.OrgName) {
			continue
		}
		// skip repos which are on Github but not on Gitlab
		for i < len(ghr) && ghr[i].Name < repo.Name {
			i++
		}

		if i < len(ghr) && ghr[i].Name == repo.Name {
			fmt.Println("exists:", repo.Name)
			err := updateGithubRepo(repo.Name, repo.Description, repo.Path)
			if err != nil {
				log.Println(err)
			}
			i++
		} else {
			fmt.Println("not on GH:", repo.Name)
			err := createGithubRepo(repo.Name, repo.Description, repo.Path)
			if err != nil {
				log.Println(err)
			}
		}
	}
}
