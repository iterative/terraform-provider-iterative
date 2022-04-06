package main

import (
    "bytes"
	"time"
	"net/http"
    "net/url"
	"encoding/json"
    "errors"
    "fmt"
    "reflect"

)

// Plausible event, as per https://plausible.io/docs/events-api
type Event struct {
	Domain      string `json:"domain"`
	Name        string `json:"name"`
	URL         string `json:"url"`
    Referrer    string `json:"referrer,omitempty"`
    ScreenWidth int    `json:"screen_width,omitempty"`
    Properties  any    `json:"props,omitempty"`
}

type Tags struct{
	Source string `utm:"source"`
	Medium string `utm:"medium"`
	Campaign string `utm:"campaign"`
	Term string `utm:"term"`
	Content string `utm:"content"`
}

func AddTags(originalUrl string, tags Tags) (string, error) {
	u, err := url.Parse(originalUrl)
	if err != nil {
		return "", err
	}
	q := u.Query()


    for _, field := range reflect.VisibleFields(reflect.TypeOf(tags)) {
		value := reflect.ValueOf(tags).FieldByIndex(field.Index)
            fmt.Println(string(field.Tag.Get("utm"))) //name_field
			fmt.Println(string(field.Name)) //name_field
			fmt.Println(value) //name_field
    }


	q.Set("utm_source", "golang")
	q.Set("utm_medium", "golang")
	q.Set("utm_campaign", "golang")
	q.Set("utm_term", "golang")
	q.Set("utm_content", "something")
	u.RawQuery = q.Encode()
	return u.String(), nil
}


type any interface{}
func NewEvent() *Event{
		

    e, err := GetEnvironment()

    if err != nil {
        panic(err)
    }


	return &Event{
		Name:     "test", //"pageview",
        Domain: "testingtestingtesting",
		URL:     u.String(),
        Referrer: "none.uk",
        Properties: e,
	}
}

func SendEvent(event *Event) error {
	body, err := json.Marshal(event)
    if err != nil {
        return err
    }
fmt.Println(string(body))
    req, err := http.NewRequest("POST", "https://plausible.io/api/event", bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("User-Agent", "TPI")
    req.Header.Set("X-Forwarded-For", "127.0.0.1")
    
    client := &http.Client{Timeout: time.Second * 30}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
	defer resp.Body.Close()
	if resp.StatusCode != 202 {
		return errors.New("server returned: " + resp.Status)
	}
    return nil
}

