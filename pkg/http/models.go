package http

type ServerConfig struct {
	Port string
}

type RouterConfig struct {
	TimeoutSec         int
	RequestPerSecLimit int
	DisableCors        bool
	AllowedOrigins     []string
	AllowedMethods     []string
	AllowedHeaders     []string
}
