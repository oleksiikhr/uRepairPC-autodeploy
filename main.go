package main

import (
	"fmt"
	"github.com/spf13/viper"
	"gopkg.in/go-playground/webhooks.v5/github"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var hook *github.Webhook

func main() {
	// Init Viper - Get data from the file
	err := initViper()
	if err != nil {
		panic(err)
	}

	// Init Github - Set secret
	hook, err = github.New(github.Options.Secret(viper.GetString("secret")))
	if err != nil {
		panic(err)
	}

	// Route
	http.HandleFunc("/", githubEventHandler)

	// Run server
	fmt.Println("Run:", viper.GetString("addr"))
	if err := http.ListenAndServe(viper.GetString("addr"), nil); err != nil {
		panic(err)
	}
}

// Find config file and set default value
func initViper() error {
	ex, err := os.Executable()
	if err != nil {
		return err
	}

	viper.SetConfigName("config")
	viper.SetDefault("addr", "0.0.0.0:4000")
	viper.SetDefault("dir", filepath.Dir(ex))
	viper.SetDefault("secret", "")
	viper.SetDefault("websocketPort", "3000")

	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.urepairpc")

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	return viper.ReadInConfig() // Find and read the config file
}

// Main logic to handle events from Github
func githubEventHandler(w http.ResponseWriter, r *http.Request) {
	payload, err := hook.Parse(r, github.PingEvent, github.PullRequestEvent)
	if err != nil {
		if err == github.ErrHMACVerificationFailed || err == github.ErrEventNotFound || err == github.ErrInvalidHTTPMethod ||
			err == github.ErrParsingPayload || err == github.ErrMissingHubSignatureHeader || err == github.ErrEventNotSpecifiedToParse {
			fmt.Println("[Event Handler]", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	switch payload.(type) {

	case github.PingPayload:
		w.Write([]byte("pong"))
		return

	case github.PullRequestPayload:
		pullRequest := payload.(github.PullRequestPayload)
		if pullRequest.Action == "closed" && pullRequest.PullRequest.Merged {
			go pullRequestMerged(&pullRequest)
			w.Write([]byte("merged"))
			return
		}
		break
	}

	w.Write([]byte("ok"))
}

func pullRequestMerged(pullRequest *github.PullRequestPayload) {
	var cmd *exec.Cmd
	if _, err := os.Stat(viper.GetString("dir") + "/" + pullRequest.Repository.Name); os.IsNotExist(err) {
		// Repository not exists? - Clone
		fmt.Println("[" + pullRequest.Repository.Name + "] Clone..")
		cmd = exec.Command("git", "clone", pullRequest.Repository.CloneURL)
		cmd.Dir = viper.GetString("dir")
	} else {
		// Repository exists? - Pull from origin master with force flag
		fmt.Println("[" + pullRequest.Repository.Name + "] Pull..")
		cmd = exec.Command("git", "pull", "origin", "master", "-f")
		cmd.Dir = viper.GetString("dir") + "/" + pullRequest.Repository.Name
	}

	// Execute command
	if err := cmd.Run(); err != nil {
		fmt.Println("[Clone/Pull Repository]", err)
		return
	}

	switch pullRequest.Repository.Name {
	case "web":
		handleWebRep("web")
		break
	case "server":
		handleServerRep("server")
		break
	case "websocket":
		handleWebsocketRep("server")
		break
	default:
		fmt.Println("[Handle Repository] Not Supported:", pullRequest.Repository.Name)
	}
}

// uRepairPC - Web
func handleWebRep(name string) {
	runCmd(name, "npm", "ci")
	runCmd(name, "npm", "run", "build")
}

// uRepairPC - Websocket
func handleWebsocketRep(name string) {
	runCmd(name, "fuser", "-k", viper.GetString("websocketPort")+"/tcp")
	runCmd(name, "npm", "ci")
	runCmd(name, "npm", "run", "build")
	runCmd(name, "npm", "run", "prod")
}

// uRepairPC - Server
func handleServerRep(name string) {
	runCmd(name, "composer", "install", "--optimize-autoloader", "--no-dev")
	runCmd(name, "php", "artisan", "cache:clear")
	runCmd(name, "php", "artisan", "config:clear")
	runCmd(name, "php", "artisan", "migrate:refresh", "--force")
	runCmd(name, "php", "artisan", "db:seed", "--force")
	runCmd(name, "php", "artisan", "config:cache")
}

// Helper function for console command
func runCmd(repositoryName string, commands ...string) bool {
	fmt.Println(strings.Join(commands, " "))
	cmd := exec.Command(commands[0], commands[1:]...)
	cmd.Dir = viper.GetString("dir") + "/" + repositoryName
	if err := cmd.Run(); err != nil {
		fmt.Println("["+repositoryName+"]", err)
		return false
	}

	return true
}
