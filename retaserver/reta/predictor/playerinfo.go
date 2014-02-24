package predictor

import (
	"strconv"
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
	//c.Debugf("Total Player: %v\n", playerlen)
	for i := 0; i < playerlen; i++ {
		//Prepare data
		social := 0
		gameplay := 0
		progression := 0.0

		var first, last time.Time
		assigned := false

		length := len(eventsData)
		for j := 0; j < length; j++ {
			if eventsData[j].Player == playerinfos[i].Name {
				if eventsData[j].Action == "Game Feature Consumed" { //Save gameplay feature when consumed
					gameplay++
				} else if eventsData[j].Action == "Social Feature Consumed" { //Save social feature when consumed
					social++
				} else if eventsData[j].Action == "Game Progression" { //Save gameplay progression
					//Increase progression
					paramlen := len(eventsData[j].Parameters)
					for k := 0; k < paramlen; k++ {
						if eventsData[j].Parameters[k].Key == "Increase" {
							progress, _ := strconv.ParseFloat(eventsData[j].Parameters[k].Value, 64)
							progression += progress
						}
					}
				}

				date := eventsData[j].Date
				if !assigned {
					first = date
					last = date
					assigned = true
				} else {
					//Min days as the first
					if first.Sub(date).Hours() >= 0 {
						first = date
					}

					//Max days as the last
					if last.Sub(date).Hours() <= 0 {
						last = date
					}
				}
			}
		}

		//Save data
		playerinfos[i].GameplayConsumed = gameplay
		playerinfos[i].SocialActivities = social
		playerinfos[i].Progression = progression
		playerinfos[i].Level = int(progression) / 5

		//Is retented?
		tomorrow := first.AddDate(0, 0, 1)
		duration := last.Sub(tomorrow)
		if duration.Hours() >= 0 {
			playerinfos[i].Day1Retention = true
		}

		//Prepare data for Level Momentum
		level := 1
		levelduration := 0.0

		length = len(timedeventsData)
		for j := 0; j < length; j++ {
			//Get player timed event
			if timedeventsData[j].Info.Player == playerinfos[i].Name {
				//Save Tutorial Momentum in Minutes
				if timedeventsData[j].Info.Action == "Tutorial Duration" {
					playerinfos[i].TutorialMomentum = timedeventsData[j].Duration.Minutes()
				} else if timedeventsData[j].Info.Action == "Level Duration" {
					level++
					levelduration += timedeventsData[i].Duration.Minutes()
				}
			}
		}

		//Save Level Momentum in Minutes
		playerinfos[i].LevelMomentum = levelduration / float64(level)
		//playerinfos[i].Level = level

		//c.Debugf("Player:\n%+v\n", playerinfos[i])
	}

	*infos = playerinfos

	return nil
}
