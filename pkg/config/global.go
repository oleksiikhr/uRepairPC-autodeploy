package config

import "github.com/spf13/viper"

type Ssl struct {
	Enable bool
	Crt    string
	Key    string
}

type Telegram struct {
	Enable      bool
	AccessToken string
	UserId      int64
}

type Repository struct {
	Name   string
	Branch string
	Path   string
}

type Repositories struct {
	Main Repository
	Docs Repository
}

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
	RedisChannel  = "autodeploy"
	RepAutodeploy = "autodeploy"
)

var Data *Config

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
			UserId:      viper.GetInt64("telegram.user_id"),
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
		},
	}
}
