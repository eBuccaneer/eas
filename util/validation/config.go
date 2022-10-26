package validation

import (
	"ethattacksim/util/file"
	"log"
	"strings"
)

func ValidateConfig(config *file.Config) {
	// add config validation here
	var err []string = make([]string, 0, 2)
	if config.Seed() < 0 {
		err = append(err, "Please use a positive integer as seed")
	}
	if config.OutPath() == "" {
		err = append(err, "OutPath should be set")
	}
	if strings.HasSuffix(config.OutPath(), "/") {
		err = append(err, "OutPath should not end with '/'")
	}

	if len(err) > 0 {
		var errMessage string = "There are configuration errors:\n"
		for _, err := range err {
			errMessage += err + "\n"
		}
		log.Panic(errMessage)
	}
}
