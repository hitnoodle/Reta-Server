package predictor

import (
	"time"

	"appengine"

	"reta/db"
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

func GetPlayerInformation(c appengine.Context, begin time.Time, end time.Time, infos *[]PlayerInfo) error {
	//Get events
	var eventsData []db.Event
	err := db.GetAllEvents(c, begin, end, &eventsData)
	if err != nil {
		return err
	}

	//Get timed events
	var timedeventsData []db.TimedEvent
	err = db.GetAllTimedEvents(c, begin, end, &timedeventsData)
	if err != nil {
		return err
	}

	//Create result array
	var playerinfos []PlayerInfo
	playerlen := len(playerinfos)

	length := len(eventsData)
	for i := 0; i < length; i++ {
		//Check if name already exist
		exist := false
		for j := 0; j < playerlen; j++ {
			if playerinfos[j].Name == eventsData[i].Player {
				exist = true
				break
			}
		}

		//Insert name for first pass
		if !exist {
			//Check whether first day + 1 is end at most
			tomorrow := eventsData[i].Date.AddDate(0, 0, 1)
			duration := end.Sub(tomorrow)

			//Add if still in region
			if duration.Hours() >= 0 {
				info := PlayerInfo{Name: eventsData[i].Player}
				playerinfos = append(playerinfos, info)
				playerlen++
			}
		}
	}

	//Second pass!
	playerlen = len(playerinfos)
	for i := 0; i < playerlen; i++ {
		length := len(eventsData)
		for j := 0; j < length; j++ {
			if eventsData[j].Player == playerinfos[i].Name {

			}
		}

		length = len(timedeventsData)
		for j := 0; j < length; j++ {
			if timedeventsData[j].Player == playerinfos[i].Name {

			}
		}
	}

	return nil
}
