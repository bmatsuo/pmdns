package main

type ConfigLoader interface {
	LoadConfigLocation(location string) ([]byte, error)
}

func LoadConfigBytes(location string) {
}
