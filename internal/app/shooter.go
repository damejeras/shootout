package app

type ShooterConfig struct {
	RedisAddr   string `config:"REDIS_ADDR"`
	ArbiterAddr string `config:"ARBITER_ADDR"`
	Name        string `config:"SHOOTER_NAME"`
	Health      int    `config:"SHOOTER_HEALTH"`
	Damage      int    `config:"SHOOTER_DAMAGE"`
}
