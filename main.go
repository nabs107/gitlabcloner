package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"

	"github.com/fatih/color"
)

type Config struct {
	GitLabURL   string `json:"gitlab_url"`
	GroupID     string `json:"group_id"`
	AccessToken string `json:"access_token"`
}

type Group struct {
	ID        int       `json:"id"`
	Projects  []Project `json:"projects"`
	Subgroups []Group   `json:"subgroups"`
	// Include other fields if needed
}

type Project struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Namespace         Namespace
	HTTPURLToRepo     string `json:"http_url_to_repo"`
	SubprojectsCount  int    `json:"subprojects_count"`
	NameWithNamespace string `json:"name_with_namespace"`
}

type Namespace struct {
	FullPath string `json:"full_path"`
}

func main() {
	configPath := path.Join(os.Getenv("HOME"), "config.json")
	config := Config{}

	if fileExists(configPath) {
		configData, err := ioutil.ReadFile(configPath)
		if err != nil {
			log.Fatal("Error reading config file:", err)
		}
		err = json.Unmarshal(configData, &config)
		if err != nil {
			log.Fatal("Error parsing config file:", err)
		}
	}

	if config.GitLabURL == "" || config.GroupID == "" || config.AccessToken == "" {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Enter GitLab URL (e.g., https://gitlab.com): ")
		gitlabURL, _ := reader.ReadString('\n')
		config.GitLabURL = strings.TrimSpace(gitlabURL)

		fmt.Print("Enter the group ID: ")
		groupID, _ := reader.ReadString('\n')
		config.GroupID = strings.TrimSpace(groupID)

		fmt.Print("Enter the access token: ")
		accessToken, _ := reader.ReadString('\n')
		config.AccessToken = strings.TrimSpace(accessToken)

		configData, err := json.Marshal(config)
		if err != nil {
			log.Fatal("Error serializing config:", err)
		}

		err = ioutil.WriteFile(configPath, configData, 0644)
		if err != nil {
			log.Fatal("Error writing config file:", err)
		}

		fetchProjects(config)
	} else {
		fetchProjects(config)
	}
}

func fetchProjects(config Config) {
	fullURL := fmt.Sprintf("%sapi/v4/groups/%s/projects?include_subgroups=true&private_token=%s&per_page=100", config.GitLabURL, config.GroupID, config.AccessToken)

	resp, err := http.Get(fullURL)
	if err != nil {
		log.Fatal("Error fetching projects:", err)
	}
	defer resp.Body.Close()

	var projects []Project
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		log.Fatal("Error parsing JSON response:", err)
	}

	// Sort the projects by name in ascending order
	sort.Slice(projects, func(i, j int) bool {
		return strings.ToLower(projects[i].Name) < strings.ToLower(projects[j].Name)
	})

	fmt.Println("Available Projects:")
	for _, project := range projects {
		var projectType string

		if strings.Contains(strings.ToLower(project.NameWithNamespace), "android") {
			projectType = "android"
		} else if strings.Contains(strings.ToLower(project.NameWithNamespace), "ios") {
			projectType = "ios"
		} else {
			projectType = project.NameWithNamespace
		}
		fmt.Printf("%s %s %s\n", project.Name, color.YellowString(fmt.Sprintf("%d", project.ID)), projectType)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter the project ID to clone: ")
	projectID, _ := reader.ReadString('\n')
	projectID = strings.TrimSpace(projectID)

	var selectedProject Project
	for _, project := range projects {
		if fmt.Sprint(project.ID) == projectID {
			selectedProject = project
			break
		}
	}

	if selectedProject.ID == 0 {
		log.Fatal("Invalid project selected")
	}

	cloneURL := selectedProject.HTTPURLToRepo
	fmt.Println("Cloning project...")
	fmt.Println("Cloning", selectedProject.Name)
	cmd := exec.Command("git", "clone", cloneURL)
	err = cmd.Run()
	if err != nil {
		log.Fatal("Error cloning project:", err)
	}
	fmt.Println("Project cloned successfully.")
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}
