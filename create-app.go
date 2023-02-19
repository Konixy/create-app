package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/cli/oauth/device"
	"github.com/joho/godotenv"

	"github.com/AlecAivazis/survey/v2"
	"github.com/google/go-github/v50/github"
	"github.com/pterm/pterm"
	"golang.org/x/oauth2"

	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	gitHttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

var qs = []*survey.Question{
	{
		Name:   "name",
		Prompt: &survey.Input{Message: "What's the name of your app?"},
		Validate: func(val interface{}) error {
			regex := regexp.MustCompile(`^[a-zA-Z1-9-_]+$`)
			if str, ok := val.(string); !ok || !regex.MatchString(str) {
				return errors.New("the name don't match the regex")
			}
			return nil
		},
	},
	{
		Name:   "directory",
		Prompt: &survey.Input{Message: "Where do you want to deploy your app? " + pterm.Gray("(default: \".\")")},
		Validate: func(val interface{}) error {
			regex := regexp.MustCompile(`((?:[^/]*/)*)(.*)`)
			str, ok := val.(string)
			if !ok {
				return errors.New("invalid response")
			} else if str == "" {
				return nil
			} else if !regex.MatchString(str) {
				return errors.New("the path don't match the regex")
			}
			return nil
		},
	},
	{
		Name:   "framework",
		Prompt: &survey.Select{Message: "What framework do you want to use?", Options: []string{"React", "NextJs", "NodeJs", "DiscordJs"}},
	},
}

func main() {
	// the answers will be written to this struct
	answers := struct {
		Name      string
		Directory string
		Framework string
	}{}

	// perform the questions
	err := survey.Ask(qs, &answers)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if answers.Directory == "" {
		answers.Directory = "."
	}

	if answers.Framework == "NextJs" {
		techs := []string{}
		err := survey.AskOne(&survey.MultiSelect{Message: "What techs do you want to use?", Options: []string{"TailwindCSS", "SASS", "Prettier", "ESLint", "FontAwesome"}}, &techs)

		if err != nil {
			fmt.Println(err.Error())
			return
		}

		fmt.Printf("Techs: %s", strings.Join(techs, ", "))
	}

	fmt.Println()
	fmt.Printf("Building %s in \"%s\"...", answers.Name, answers.Directory)
	fmt.Println()

	repo := false
	survey.AskOne(&survey.Confirm{Message: "Would you like to create a github repo?"}, &repo)

	if repo {
		godotenv.Load(".env")
		githubToken := os.Getenv("GITHUB_TOKEN")
		if githubToken == "" {
			githubToken = GithubOAuth()
		}

		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: githubToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		client := github.NewClient(tc)

		user, _, err := client.Users.Get(ctx, "")
		if err != nil {
			githubToken = GithubOAuth()

			ts := oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: githubToken},
			)
			tc := oauth2.NewClient(ctx, ts)
			client = github.NewClient(tc)
		}

		fmt.Printf("Connected as %s", *user.Login)

		repoName := answers.Name
		repoNameRes := struct {
			Name string
		}{Name: repoName}
		survey.Ask([]*survey.Question{{Name: "name", Prompt: &survey.Input{Message: "Enter the name of the github repo (default \"" + repoName + "\"):"}, Validate: func(val interface{}) error {
			regex := regexp.MustCompile(`^[a-zA-Z1-9-_]+$`)
			str, ok := val.(string)
			if !ok {
				return errors.New("invalid response")
			} else if str == "" {
				return nil
			} else if !regex.MatchString(str) {
				return errors.New("the name don't match the regex")
			}
			return nil
		}}}, repoNameRes)
		if repoNameRes.Name == "" {
			repoNameRes.Name = repoName
		}

		isPrivate := "private"
		survey.AskOne(&survey.Select{Message: "Do you want you repo public or private?", Default: isPrivate, Options: []string{"private", "public"}}, &isPrivate)

		isPrivateBool := new(bool)
		*isPrivateBool = isPrivate == "private"

		repo, _, err := client.Repositories.Create(ctx, "", &github.Repository{
			Name:    &repoNameRes.Name,
			Private: isPrivateBool,
		})
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Repository created: %s\n", *repo.HTMLURL)
		fmt.Printf("Pushing changes...")

		r, _ := git.Init(memory.NewStorage(), memfs.New())

		_, err = r.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{*repo.HTMLURL},
		})

		if err != nil {
			panic(err)
		}

		list, err := r.Remotes()
		if err != nil {
			panic(err)
		}

		for _, r := range list {
			fmt.Println(r)
		}

		// wt, err := r.Worktree()
		if err != nil {
			fmt.Println("Error getting worktree:", err)
			os.Exit(1)
		}

		initCmd := exec.Command("git", "init")
		initCmd.Dir = answers.Directory
		initCmdOutput, err := initCmd.CombinedOutput()
		if err != nil {
			fmt.Println("Error adding files to Git index:", err)
			fmt.Println(string(initCmdOutput))
			return
		}

		remoteCmd := exec.Command("git", "remote", "add", "origin", *repo.HTMLURL)
		remoteCmd.Dir = answers.Directory
		remoteCmdOutput, err := remoteCmd.CombinedOutput()
		if err != nil {
			fmt.Println("Error adding files to Git index:", err)
			fmt.Println(string(remoteCmdOutput))
			return
		}

		addCmd := exec.Command("git", "add", "-A")
		addCmd.Dir = answers.Directory
		addCmdOutput, err := addCmd.CombinedOutput()
		if err != nil {
			fmt.Println("Error adding files to Git index:", err)
			fmt.Println(string(addCmdOutput))
			return
		}

		commitCmd := exec.Command("git", "commit", "-m", "Initial commit")
		commitCmd.Dir = answers.Directory
		commitCmdOutput, err := commitCmd.CombinedOutput()
		if err != nil {
			fmt.Println("Error commiting files to Git index:", err)
			fmt.Println(string(commitCmdOutput))
			return
		}

		fmt.Println("Files added to Git index.")

		// commitMsg := "Initial commit"
		// wt.AddGlob(answers.Directory)
		// commit, err := wt.Commit(commitMsg, &git.CommitOptions{
		// 	Author: &object.Signature{
		// 		Name:  "Create-App",
		// 		Email: "konixy.p@gmail.com",
		// 	},
		// })
		// if err != nil {
		// 	fmt.Println("Error committing changes:", err)
		// 	os.Exit(1)
		// }

		// Push the changes to the remote repository
		err = r.Push(&git.PushOptions{
			RemoteName: "origin",
			Auth: &gitHttp.BasicAuth{
				Username: "Create-App",
				Password: githubToken,
			},
		})
		if err != nil {
			fmt.Println("Error pushing changes:", err)
			os.Exit(1)
		}

		// fmt.Printf("Commit id: %s", commit.String())
		fmt.Println()
		fmt.Println("Changes pushed to remote repository.")
	}
}

func GithubOAuth() string {
	clientID := os.Getenv("OAUTH_CLIENT_ID")
	scopes := []string{"repo", "read:org"}
	httpClient := http.DefaultClient

	code, err := device.RequestCode(httpClient, "https://github.com/login/device/code", clientID, scopes)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Copy code: %s\n", code.UserCode)
	fmt.Printf("then open: %s\n", code.VerificationURI)

	accessToken, err := device.Wait(context.TODO(), httpClient, "https://github.com/login/oauth/access_token", device.WaitOptions{
		ClientID:   clientID,
		DeviceCode: code,
	})
	if err != nil {
		panic(err)
	}

	os.Setenv("GITHUB_TOKEN", accessToken.Token)
	godotenv.Write(map[string]string{"OAUTH_CLIENT_ID": clientID, "GITHUB_TOKEN": accessToken.Token}, ".env")
	return accessToken.Token
}
