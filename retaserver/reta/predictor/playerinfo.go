package predictor

import (
	"time"
)

type PlayerInfo struct {
	Name             string
	TutorialMomentum float64
	LevelMomentum    float64
	GameplayConsumed int
	SocialActivities int
	Progression      float64
	Level            int
	Day1Retention    bool
}

func GetPlayerInformation(begin time.Time, end time.Time, infos *[]PlayerInfo) error {
	return nil
}
