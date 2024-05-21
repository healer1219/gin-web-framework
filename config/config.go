package config

type Config struct {
	App          App                    `mapstructure:"app" json:"app" yaml:"app"`
	Log          Log                    `mapstructure:"mlog" json:"mlog" yaml:"mlog"`
	Database     Database               `mapstructure:"database" json:"database" yaml:"database"`
	DatabaseMap  map[string]Database    `mapstructure:"dbs" json:"dbs" yaml:"dbs"`
	Redis        Redis                  `mapstructure:"redis" json:"redis" yaml:"redis"`
	CustomConfig map[string]interface{} `mapstructure:"custom" json:"custom" yaml:"custom"`
	Cloud        Registry               `mapstructure:"cloud" json:"cloud" yaml:"cloud"`
}
