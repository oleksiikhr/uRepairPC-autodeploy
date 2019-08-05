package main

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"

	"github.com/go-redis/redis"
	"github.com/robfig/cron/v3"
	"github.com/uRepairPC/autodeploy/pkg/config"
	"github.com/uRepairPC/autodeploy/pkg/logger"
	"github.com/uRepairPC/autodeploy/pkg/telegram"
	"gopkg.in/go-playground/webhooks.v5/github"
)

var (
	redisClient *redis.Client
	hook        *github.Webhook
)

func main() {
	var err error

	// Init Viper - Get data from the file and env
	config.LoadConfig()

	// Init Github
	hook, err = github.New(github.Options.Secret(config.Data.Secret))
	if err != nil {
		logger.Panic(err)
	}

	// Init Telegram
	if config.Data.Telegram.Enable {
		if err = telegram.NewTelegram(config.Data.Telegram.AccessToken); err != nil {
			logger.Panic(err)
		}
	}

	// Run Cron
	if err = runCron(); err != nil {
		logger.Panic(err)
	}

	// Redis
	redisClient = redis.NewClient(&redis.Options{})

	// Route
	http.HandleFunc("/", githubEventHandler)

	// Run server
	if err := runServer(); err != nil {
		logger.Panic(err)
	}
}

func runServer() error {
	logger.Info("Run server: " + config.Data.Addr)

	if config.Data.Ssl.Enable {
		return http.ListenAndServeTLS(config.Data.Addr, config.Data.Ssl.Crt, config.Data.Ssl.Key, nil)
	}

	return http.ListenAndServe(config.Data.Addr, nil)
}

func runCron() error {
	c := cron.New()

	// Clear all data every xx hours (DB, other)
	if config.Data.Destroy != "" {
		_, err := c.AddFunc(config.Data.Destroy, func() {
			logger.Info("Destroy server by cron")
			handleMainRep()
		})
		if err != nil {
			return err
		}
	}

	c.Start()
	return nil
}

func githubEventHandler(w http.ResponseWriter, r *http.Request) {
	payload, err := hook.Parse(r, github.PingEvent, github.PullRequestEvent)
	if err != nil {
		if err == github.ErrHMACVerificationFailed || err == github.ErrEventNotFound || err == github.ErrInvalidHTTPMethod ||
			err == github.ErrParsingPayload || err == github.ErrMissingHubSignatureHeader || err == github.ErrEventNotSpecifiedToParse {
			logger.Warning(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	output := "ok"

	switch payload.(type) {
	case github.PingPayload:
		output = "pong"
		break

	case github.PullRequestPayload:
		pullRequestPayload := payload.(github.PullRequestPayload)
		if pullRequestPayload.Action == "closed" && pullRequestPayload.PullRequest.Merged {
			output = "accepted pullRequest"
			go pullRequestEvent(&pullRequestPayload)
		}
		break

	case github.PushPayload:
		pushPayload := payload.(github.PushPayload)
		if !pushPayload.Deleted {
			output = "accepted push"
			go pushEvent(&pushPayload)
		}
		break
	}

	w.Write([]byte(output))
}

func pullRequestEvent(pullRequestPayload *github.PullRequestPayload) {
	branch := pullRequestPayload.PullRequest.Base.Ref

	switch pullRequestPayload.Repository.Name {
	case config.Data.Repositories.Main.Name:
		if config.Data.Repositories.Main.Branch == branch {
			handleMainRep()
		}
		break
	case config.Data.Repositories.Docs.Name:
		if config.Data.Repositories.Docs.Branch == branch {
			handleDocsRep()
		}
		break
	default:
		logger.Warning(pullRequestPayload.Repository.Name + " pullRequestEvent not supported")
	}
}

func pushEvent(pushPayload *github.PushPayload) {
	switch pushPayload.Repository.Name {
	case config.Data.Repositories.Main.Name:
		if pushPayload.Ref == "refs/heads/"+config.Data.Repositories.Main.Branch {
			handleMainRep()
		}
		break
	case config.Data.Repositories.Docs.Name:
		if pushPayload.Ref == "refs/heads/"+config.Data.Repositories.Docs.Branch {
			handleDocsRep()
		}
		break
	default:
		logger.Warning(pushPayload.Repository.Name + " pushEvent not supported")
	}
}

// uRepairPC/uRepairPC
func handleMainRep() {
	rep := &config.Data.Repositories.Main
	ws := &config.Data.Repositories.Websocket

	redisPublishStatus(rep, true)

	// Update files from Github
	cmd(rep, "git", "pull", "origin", rep.Branch, "-f")

	// Remove all user files
	cmd(rep, "rm", "-rf", rep.Path+"/storage/app/requests/")
	cmd(rep, "rm", "-rf", rep.Path+"/storage/app/equipments/")
	cmd(rep, "rm", "-rf", rep.Path+"/storage/app/users/")
	cmd(rep, "rm", "-rf", rep.Path+"/storage/app/public/global/")
	cmd(rep, "rm", "-f", rep.Path+"/storage/app/manifest.json")
	cmd(rep, "rm", "-f", rep.Path+"/storage/app/settings.json")

	// Install dependencies. Refresh DB
	cmd(rep, "composer", "install", "--optimize-autoloader")
	cmd(rep, "php", "artisan", "cache:clear")
	cmd(rep, "php", "artisan", "config:clear")
	cmd(rep, "php", "artisan", "migrate:fresh", "--force")
	cmd(rep, "php", "artisan", "db:seed", "--force")
	cmd(rep, "php", "artisan", "config:cache")

	// Stop/Update/Start Websocket server
	cmd(ws, "pm2", "delete", "app")
	cmd(ws, "npm", "ci", "--production")
	cmd(ws, "npm", "run", "start")

	redisPublishStatus(rep, false)
	logger.Info(rep.Name + ": Complete")
}

// uRepairPC/docs
func handleDocsRep() {
	rep := &config.Data.Repositories.Docs
	cmd(rep, "git", "pull", "origin", rep.Branch, "-f")
	cmd(rep, "npm", "ci")
	cmd(rep, "npm", "run", "build:docs")
	logger.Info(rep.Name + ": Complete")
}

// Execute command in project folder
func cmd(rep *config.Repository, commands ...string) bool {
	logger.Info(rep.Name + ": " + strings.Join(commands, " "))
	cmd := exec.Command(commands[0], commands[1:]...)
	cmd.Dir = rep.Path
	if err := cmd.Run(); err != nil {
		logger.Error(err)
		return false
	}

	return true
}

// Publish status to Redis (Websocket accept)
func redisPublishStatus(rep *config.Repository, process bool) {
	data, _ := json.Marshal(map[string]interface{}{
		"event": "status",
		"data": map[string]interface{}{
			"process": process,
		},
	})

	redisClient.Publish(config.RedisChannel+"."+rep.Name, data)
}
