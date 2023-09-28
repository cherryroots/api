package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

type note struct {
	ID  string
	URL string
}

func main() {
	// read from a file called input in the same directory
	body, err := os.ReadFile("input.txt")
	if err != nil {
		log.Fatal(err)
	}
	example := string(body)

	regex := regexp.MustCompile(`(?m)[<]?(https?:\/\/[^\s<>]+)[>]?\b`)
	result := regex.FindAllStringSubmatch(example, -1)
	for _, element := range result {
		log.Printf("URL: %s", element[0])
		u, err := url.Parse(element[0])
		if err != nil {
			log.Fatal(err)
		}
		host := u.Host
		username := strings.Split(u.Path, "/")[1]
		username, _ = strings.CutPrefix(username, "@")
		log.Printf("host: %s, username: %s", host, username)
		id, notesCount, err := getUserID(username, host)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("userID: %s, notesCount: %d", id, notesCount)
		notes, err := getUserNotes(id, notesCount)
		if err != nil {
			log.Fatal(err)
		}
		for _, note := range notes {
			// check if note url matches example and print id and url
			if strings.Contains(note.URL, element[0]) {
				id = note.ID
				noteurl := "https://blahaj.zone/notes/" + id
				log.Printf("noteID: %s, url: %s", id, noteurl)
			}
		}
	}
}

func getUserID(username string, host string) (string, int64, error) {
	url := "https://blahaj.zone/api/users/show"
	json := []byte(`{"username": "` + username + `", "host": "` + host + `"}`)
	body, err := postAPI(url, string(json))
	if err != nil {
		return "", 0, err
	}

	id := gjson.Get(string(body), "id").String()
	notesCount := gjson.Get(string(body), "notesCount").Int()

	return id, notesCount, nil
}

func getUserNotes(userid string, notesCount int64) ([]note, error) {
	// array of notes
	var noteList = []note{}

	totalPasses := math.Ceil(float64(notesCount) / 100.0)
	var passes int64

	for i := 0; i < int(totalPasses); i++ {
		time.Sleep(1 * time.Second)
		url := "https://blahaj.zone/api/users/notes"
		var json = []byte{}
		if int(totalPasses) == 1 {
			json = []byte(`{"userId": "` + userid + `", "limit": ` + fmt.Sprint(notesCount) + `}`)
			passes += notesCount
		}
		if int(totalPasses) > 1 && i == 0 {
			json = []byte(`{"userId": "` + userid + `", "limit": ` + fmt.Sprint(100) + `}`)
			passes += 100
		}
		if int(totalPasses) > 1 && i > 0 {
			// get last id in notesList
			lastID := noteList[len(noteList)-1].ID
			json = []byte(`{"userId": "` + userid + `", "limit": ` + fmt.Sprint(notesCount-passes) + `", "sinceId": "` + lastID + `"}`)
			passes += notesCount - passes
		}
		body, err := postAPI(url, string(json))
		if err != nil {
			return noteList, err
		}

		jsonArray := gjson.Parse(string(body)).Array()
		for _, json := range jsonArray {
			noteList = append(noteList, note{
				ID:  json.Get("id").String(),
				URL: json.Get("url").String(),
			})
		}
	}

	return noteList, nil
}

func postAPI(path string, data string) (string, error) {
	req, err := http.NewRequest("POST", path, strings.NewReader(string(data)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
