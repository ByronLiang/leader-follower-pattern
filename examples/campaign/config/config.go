package config

import (
	"os"
)

var (
	KeyPrefix       = ""
	EtcdHost        = ""
	BindAddress     = ""
	RegisterAddress = ""
)

func InitConfig() {
	if os.Getenv("KeyPrefix") != "" {
		KeyPrefix = os.Getenv("KeyPrefix")
	}
	if os.Getenv("EtcdHost") != "" {
		EtcdHost = os.Getenv("EtcdHost")
	}
	if os.Getenv("BindAddress") != "" {
		BindAddress = os.Getenv("BindAddress")
	}
	if os.Getenv("RegisterAddress") != "" {
		RegisterAddress = os.Getenv("RegisterAddress")
	}
}
