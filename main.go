package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-redis/redis"
	"github.com/robfig/cron"
	"github.com/spf13/viper"
	"gopkg.in/go-playground/webhooks.v5/github"
)

const (
	RedisChannel = "autodeploy"
	RepWebsocket = "websocket"
	RepServer    = "server"
	RepWeb       = "web"
	RepDocs      = "docs"
)

var (
	redisClient *redis.Client
	hook        *github.Webhook
)

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

	// Cron
	c := cron.New()
	c.AddFunc(viper.GetString("refresh"), func() {
		// Clear all data every xx hours (DB, other)
		fmt.Println("[CRON] refresh server")
		handleServerRep()
	})
	c.Start()

	// Redis
	redisClient = redis.NewClient(&redis.Options{})

	// Route
	http.HandleFunc("/", githubEventHandler)

	// Run server
	fmt.Println("Run:", viper.GetString("addr"))
	if viper.GetBool("ssl") {
		err = http.ListenAndServeTLS(viper.GetString("addr"), viper.GetString("sslCrt"), viper.GetString("sslKey"), nil)
	} else {
		err = http.ListenAndServe(viper.GetString("addr"), nil)
	}

	if err != nil {
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
	viper.SetDefault("ssl", false)
	viper.SetDefault("refresh", "@every 16h")

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
		if pullRequest.Action == "closed" && pullRequest.PullRequest.Merged &&
			pullRequest.PullRequest.Base.Ref == pullRequest.Repository.DefaultBranch {
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
		log(pullRequest.Repository.Name, "git clone")
		cmd = exec.Command("git", "clone", pullRequest.Repository.CloneURL)
		cmd.Dir = viper.GetString("dir")
	} else {
		// Repository exists? - Pull from origin master with force flag
		log(pullRequest.Repository.Name, "pull origin master -f")
		cmd = exec.Command("git", "pull", "origin", "master", "-f")
		cmd.Dir = viper.GetString("dir") + "/" + pullRequest.Repository.Name
	}

	// Execute command
	if err := cmd.Run(); err != nil {
		log(pullRequest.Repository.Name, err)
		return
	}

	switch pullRequest.Repository.Name {
	case RepWeb:
		handleWebRep()
		break
	case RepServer:
		handleServerRep()
		break
	case RepWebsocket:
		handleWebsocketRep()
		break
	case RepDocs:
		handleDocsRep()
		break
	default:
		fmt.Println("[Handle Repository] Not Supported:", pullRequest.Repository.Name)
	}
}

// uRepairPC - Web
func handleWebRep() {
	redisPublishStatus(RepWeb, true)
	runCmd(RepWeb, "npm", "ci")
	runCmd(RepWeb, "npm", "run", "build")
	redisPublishStatus(RepWeb, false)
	log(RepWeb, "Complete")
}

// uRepairPC - Websocket
func handleWebsocketRep() {
	redisPublishStatus(RepWebsocket, true)
	runCmd(RepWebsocket, "fuser", "-k", viper.GetString("websocketPort")+"/tcp")
	runCmd(RepWebsocket, "npm", "ci")
	runCmd(RepWebsocket, "npm", "run", "build")
	runCmd(RepWebsocket, "npm", "run", "prod")
	// redis false on reconnect to the websocket
}

// uRepairPC - Server
func handleServerRep() {
	redisPublishStatus(RepServer, true)
	runCmd(RepServer, "composer", "install", "--optimize-autoloader")
	runCmd(RepServer, "php", "artisan", "cache:clear")
	runCmd(RepServer, "php", "artisan", "config:clear")
	runCmd(RepServer, "php", "artisan", "migrate:refresh", "--force")
	runCmd(RepServer, "php", "artisan", "db:seed", "--force")
	runCmd(RepServer, "php", "artisan", "config:cache")
	redisPublishStatus(RepServer, false)
	log(RepServer, "Complete")
}

// uRepairPC - Docs
func handleDocsRep() {
	runCmd(RepDocs, "npm", "ci")
	runCmd(RepDocs, "npm", "run", "build:docs")
	log(RepDocs, "Complete")
}

// Helper function for console command
// Run only in folder in the project
func runCmd(repositoryName string, commands ...string) bool {
	log(repositoryName, strings.Join(commands, " "))
	cmd := exec.Command(commands[0], commands[1:]...)
	cmd.Dir = viper.GetString("dir") + "/" + repositoryName
	if err := cmd.Run(); err != nil {
		log(repositoryName, err)
		return false
	}

	return true
}

func redisPublishStatus(repositoryName string, process bool) {
	data, _ := json.Marshal(map[string]interface{}{
		"event": "autodeploy.status",
		"data": map[string]interface{}{
			"name":    repositoryName,
			"process": process,
		},
	})

	redisClient.Publish(RedisChannel+"."+repositoryName, data)
}

func log(repositoryName string, text interface{}) {
	fmt.Println("["+repositoryName+"]", text)
}
