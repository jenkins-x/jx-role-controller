package loghelpers

import (
	"github.com/jenkins-x/jx-logging/pkg/log"
)

// InitLogrus initialises logging nicely
func InitLogrus() {
	// lets force jx to initialise
	log.Logger()
}
