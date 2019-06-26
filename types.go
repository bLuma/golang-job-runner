package main

import (
	"time"

	"github.com/satori/go.uuid"
)

type ConfigurationID uuid.UUID

func (cid ConfigurationID) MarshalText() (text []byte, err error) {
	return []byte(uuid.UUID(cid).String()), nil
}

func (cid *ConfigurationID) UnmarshalText(text []byte) error {
	uuid, err := uuid.FromString(string(text))
	if err != nil {
		return err
	}

	*cid = ConfigurationID(uuid)
	return nil
}

func (cid ConfigurationID) String() string {
	return uuid.UUID(cid).String()
}

func GetNewID() ConfigurationID {
	return ConfigurationID(uuid.Must(uuid.NewV4()))
}

type Configuration struct {
	ID      ConfigurationID
	Params  []string
	Runtime int
}

type LeasedConfiguration struct {
	Configuration
	Host      string
	StartTime time.Time
	Expiry    time.Time
	MemUsed   []int64
	CPUUsed   []int64
}

type FinishedConfiguration struct {
	Configuration
	Host      string
	StartTime time.Time
	EndTime   time.Time
	MemUsed   []int64
	CPUUsed   []int64
	Status    string
}
