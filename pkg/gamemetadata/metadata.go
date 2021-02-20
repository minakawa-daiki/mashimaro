package gamemetadata

import (
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/pkg/errors"
)

type Metadata struct {
	GameID  string `yaml:"gameId" firestore:"gameId"`
	Command string `yaml:"command" firestore:"command"`
}

func Marshal(md *Metadata) ([]byte, error) {
	return yaml.Marshal(md)
}

func Unmarshal(data []byte, md *Metadata) error {
	return yaml.Unmarshal(data, md)
}

func (md *Metadata) ParseCommand() (cmd string, args []string, err error) {
	cmds := strings.Split(md.Command, " ")
	if len(cmds) < 1 {
		err = errors.New("short command")
		return
	}
	cmd = cmds[0]
	if len(cmds) > 1 {
		args = cmds[1:]
	}
	return
}
