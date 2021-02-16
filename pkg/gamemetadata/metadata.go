package gamemetadata

type Metadata struct {
	GameID  string `yaml:"gameId" firestore:"gameId"`
	Command string `yaml:"command" firestore:"command"`
}
