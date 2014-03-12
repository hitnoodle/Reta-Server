package db

import (
	"encoding/json"
	"time"

	"appengine"
	"appengine/datastore"
)

type Parameter struct {
	Key   string
	Value string
}

type Event struct {
	Player     string
	Version    string
	Action     string
	Date       time.Time
	Parameters []Parameter
}

type TimedEvent struct {
	Info     Event
	Duration time.Duration
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
			layout := "01/02/2006 15:04:05"
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

func GetAllEvents(c appengine.Context, begin time.Time, end time.Time, events *[]Event) error {
	q := datastore.NewQuery("Event").Filter("Date >=", begin).Filter("Date <=", end).Order("Date")

	var eventsData []Event

	_, err := q.GetAll(c, &eventsData)
	if err != nil {
		return err
	}

	/*
		done := false
		var cursor *datastore.Cursor
		for !done {
			q = q.Limit(1000)
			if cursor != nil {
				q = q.Start(*cursor)
			}

			t := q.Run(c)
			output := 0
			for true {
				var ev Event
				_, err := t.Next(&ev)
				if err == datastore.Done {
					break
				}

				if err != nil {
					return err
				}

				eventsData = append(eventsData, ev)
				output++
			}

			if output == 0 {
				done = true
			} else {
				newCursor, err := t.Cursor()
				if err != nil {
					return err
				}
				cursor = &newCursor
			}
		}
	*/

	*events = eventsData

	return nil
}

func GetAllTimedEvents(c appengine.Context, begin time.Time, end time.Time, timedevents *[]TimedEvent) error {
	q := datastore.NewQuery("Timed Event").Filter("Info.Date >=", begin).Filter("Info.Date <=", end).Order("Info.Date")

	var timedeventsData []TimedEvent

	_, err := q.GetAll(c, &timedeventsData)
	if err != nil {
		return err
	}

	/*
		done := false
		var cursor *datastore.Cursor
		for !done {
			q = q.Limit(1000)
			if cursor != nil {
				q = q.Start(*cursor)
			}

			t := q.Run(c)
			output := 0
			for true {
				var tev TimedEvent
				_, err := t.Next(&tev)
				if err == datastore.Done {
					break
				}

				if err != nil {
					return err
				}

				timedeventsData = append(timedeventsData, tev)
				output++
			}

			if output == 0 {
				done = true
			} else {
				newCursor, err := t.Cursor()
				if err != nil {
					return err
				}
				cursor = &newCursor
			}
		}
	*/

	*timedevents = timedeventsData

	return nil
}
