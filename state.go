package main

import (
	"encoding/json"
	"github.com/phemmer/sawmill"
	"io/ioutil"
)

func GetDomains(cfg *Config) *Domains {
	filename := cfg.DBFilename
	if len(cfg.DBFilename) > 0 {
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			sawmill.Warning("GetDomains", sawmill.Fields{
				"filename": filename,
				"err": err.Error(),
			})

			return NewDomains(&cfg.DBFilename)
		}

		domains := make(map[DomainKey]*Domain)
		err = json.Unmarshal(data, &domains)
		if err != nil {
			sawmill.Warning("GetDomains", sawmill.Fields{
				"filename": filename,
				"err": err.Error(),
			})

			return NewDomains(&cfg.DBFilename)
		}

		return &Domains{
			domains: domains,
			filename: &cfg.DBFilename,
		}
	}

	return NewDomains(&cfg.DBFilename)
}