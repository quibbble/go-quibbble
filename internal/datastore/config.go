package datastore

type DatastoreConfig struct {
	Redis RedisConfig
}

type RedisConfig struct {
	Enabled  bool
	Host     string
	Password string
}
