package config

import "github.com/spf13/viper"

// Ssl on "enable" is true, run the server on https
type Ssl struct {
  Enable bool
  Crt    string
  Key    string
}

// Telegram on "enable" is true - send message logs for user in Telegram
type Telegram struct {
  Enable      bool
  AccessToken string
  UserID      int64
}

// Repository this data from org/rep
type Repository struct {
  Name   string
  Branch string
  Path   string
}

// Repositories this data from organization
type Repositories struct {
  Main      Repository
  Docs      Repository
  Websocket Repository
}

// Config for the application
type Config struct {
  Secret        string // Github hook
  Addr          string // Run app
  Destroy       string // Remove all data
  WebsocketPort string // Kill/Start server
  Ssl           Ssl
  Telegram      Telegram
  Repositories  Repositories
}

const (
  // RedisChannel - name of a channel for publishing the message to Redis
  RedisChannel = "autodeploy"

  // RepAutodeploy - name the repository on github.com
  RepAutodeploy = "autodeploy"
)

// Data - collected from the config file/.env/default
var Data *Config

// LoadConfig collects data and writes to variable
func LoadConfig() {
  viper.SetConfigName(RepAutodeploy)
  viper.SetEnvPrefix(RepAutodeploy)

  viper.SetDefault("secret", "")
  viper.SetDefault("addr", "0.0.0.0:4000")
  viper.SetDefault("websocketPort", "3000")
  viper.SetDefault("destroy", "@every 16h")
  viper.SetDefault("telegram.enable", false)
  viper.SetDefault("ssl.enable", false)

  viper.AddConfigPath(".")
  viper.AddConfigPath("$HOME/." + RepAutodeploy)

  _ = viper.ReadInConfig()
  viper.AutomaticEnv()

  Data = &Config{
    Secret:        viper.GetString("secret"),
    Addr:          viper.GetString("addr"),
    WebsocketPort: viper.GetString("websocketPort"),
    Destroy:       viper.GetString("destroy"),
    Ssl: Ssl{
      Enable: viper.GetBool("ssl.enable"),
      Crt:    viper.GetString("ssl.crt"),
      Key:    viper.GetString("ssl.key"),
    },
    Telegram: Telegram{
      Enable:      viper.GetBool("telegram.enable"),
      AccessToken: viper.GetString("telegram.access_token"),
      UserID:      viper.GetInt64("telegram.user_id"),
    },
    Repositories: Repositories{
      Main: Repository{
        Name:   viper.GetString("repositories.main.name"),
        Branch: viper.GetString("repositories.main.branch"),
        Path:   viper.GetString("repositories.main.path"),
      },
      Docs: Repository{
        Name:   viper.GetString("repositories.docs.name"),
        Branch: viper.GetString("repositories.docs.branch"),
        Path:   viper.GetString("repositories.docs.path"),
      },
      Websocket: Repository{
        Name:   viper.GetString("repositories.websocket.name"),
        Branch: viper.GetString("repositories.websocket.branch"),
        Path:   viper.GetString("repositories.websocket.path"),
      },
    },
  }
}
