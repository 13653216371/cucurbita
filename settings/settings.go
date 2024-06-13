package settings

import "os"

var Storage = "/var/lib/cucurbita/"
var Address = ":80"

func init() {
	if value := os.Getenv("CUCURBITA_STORAGE"); len(value) != 0 {
		Storage = value
	}
	if value := os.Getenv("CUCURBITA_ADDRESS"); len(value) != 0 {
		Address = value
	}
}
