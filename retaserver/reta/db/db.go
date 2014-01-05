package db

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"appengine"
	"appengine/datastore"
)

func SubmitActivity(c appengine.Context, player string, version string, data string) error {
	activity := Activity{
		Player:     player,
		Version:    version,
		ServerTime: time.Now(),
		Data:       data,
	}

	//TODO: Limit activity insertion (ex: 100)
	_, err := datastore.Put(c, datastore.NewIncompleteKey(c, "Activity", nil), &activity)
	if err != nil {
		return err
	}

	return SubmitEvent(c, activity.Player, activity.Version, activity.Data)
}

func SubmitEvent(c appengine.Context, player string, version string, data string) error {
	b := []byte(data)

	var f interface{}
	err := json.Unmarshal(b, &f)
	if err != nil {
		return err
	}

	var duration time.Duration = 0
	var tev TimedEvent

	var ev Event
	ev.Player = player
	ev.Version = version

	//Parse json to event object
	m := f.(map[string]interface{})
	for k, v := range m {
		if k == "Name" {
			//Just get the name
			action := v.(string)
			ev.Action = action
		} else if k == "Time" {
			//Convert to time
			layout := "01/02/2006 03:04:05"
			clienttime := v.(string)
			ev.Date, _ = time.Parse(layout, clienttime)
		} else if k == "Parameters" {
			//Convert to parameters
			pars := v.([]interface{})
			parameters := make([]Parameter, len(pars))
			for i, par := range pars {
				var pinterface interface{}
				p := []byte(par.(string))

				err = json.Unmarshal(p, &pinterface)
				if err != nil {
					return err
				}

				//There's only one parameter per map
				pmap := pinterface.(map[string]interface{})
				for kpar, vpar := range pmap {
					param := Parameter{
						Key:   kpar,
						Value: vpar.(string),
					}
					//See above comment
					parameters[i] = param
				}
			}
			ev.Parameters = parameters
		} else if k == "Duration" {
			//Convert to duration
			durr := v.(string)
			duration, _ = time.ParseDuration(durr)
		}
	}

	//Save to appropriate datastore
	if duration == 0 {
		c.Debugf("Event: %v\n", ev)
		_, err := datastore.Put(c, datastore.NewIncompleteKey(c, "Event", nil), &ev)
		if err != nil {
			return err
		}
	} else {
		tev.Info = ev
		tev.Duration = duration

		c.Debugf("Timed Event: %v\n", tev)
		_, err := datastore.Put(c, datastore.NewIncompleteKey(c, "Timed Event", nil), &tev)
		if err != nil {
			return err
		}
	}

	return nil
}

func ListActivities(w http.ResponseWriter, c appengine.Context, limit int) {
	q := datastore.NewQuery("Activity").Order("-ServerTime").Limit(limit)
	for t := q.Run(c); ; {
		var act Activity

		_, err := t.Next(&act)
		if err == datastore.Done {
			break
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Activity: %v\n\n", act)
	}

	return
}
