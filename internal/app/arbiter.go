package app

type ArbiterConfig struct {
	Port      string `config:"PORT"`
	RedisAddr string `config:"REDIS_ADDR"`
}
