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

type noteStore struct {
	Username string
	Notes    []note
}

// main is the entry point of the Go program.
//
// It reads the contents of a file called "input.txt" in the same directory.
// Then, it searches for URLs in the content using a regular expression.
// It retrieves the host and username from each URL and makes API calls to get the user ID, notes count, and user notes.
// Finally, it saves the note URLs to a file called "output.txt".
func main() {
	// read from a file called input in the same directory
	body, err := os.ReadFile("input.txt")
	if err != nil {
		log.Fatal(err)
	}
	example := string(body)

	regex := regexp.MustCompile(`(?m)[<]?(https?:\/\/[^\s<>]+)[>]?\b`)
	result := regex.FindAllStringSubmatch(example, -1)
	noteCache := []noteStore{}
	noteURLs := []string{}
	for _, element := range result {
		log.Print("------------------------------------------------")
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

		var notes []note
		// if noteCache is empty get notes
		if len(noteCache) == 0 {
			log.Print("Getting notes...")
			notes, err = getUserNotes(id, notesCount)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			for _, note := range noteCache {
				if note.Username == id {
					log.Print("Found in cache")
					notes = note.Notes
					break
				} else {
					log.Print("Not in cache, getting notes...")
					notes, err = getUserNotes(id, notesCount)
					if err != nil {
						log.Fatal(err)
					}
					break
				}
			}
		}
		noteCache = append(noteCache, noteStore{Username: id, Notes: notes})

		var match bool
		for _, note := range notes {
			// check if note url matches example and print id and url
			if strings.Contains(note.URL, element[0]) {
				id = note.ID
				noteURL := "https://blahaj.zone/notes/" + id
				saveURL := noteURL + " = " + element[0]
				noteURLs = append(noteURLs, saveURL)
				match = true
				log.Printf("noteID: %s, url: %s", id, noteURL)
			}
		}
		if !match {
			saveURL := "No match for found"
			noteURLs = append(noteURLs, saveURL)
			log.Printf("No match for %s", element[0])
		}
	}
	err = os.WriteFile("output.txt", []byte(strings.Join(noteURLs, "\n")), 0644)
	if err != nil {
		log.Fatal(err)
	}
}

// getUserID retrieves the user ID and notes count for a given username and host.
//
// Parameters:
// - username: The username of the user.
// - host: The host of the user.
//
// Returns:
// - id: The user ID.
// - notesCount: The count of notes for the user.
// - error: The error that occurred during the API call.
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

// getUserNotes retrieves the notes of a user based on the provided user ID and the count of notes.
//
// Parameters:
// - userid: a string representing the ID of the user.
// - notesCount: an int64 representing the count of notes to retrieve.
//
// Returns:
// - []note: an array of notes.
// - error: an error if there was a problem retrieving the notes.
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

// postAPI sends a POST request to the specified path with the provided data and returns the response body as a string.
//
// Parameters:
// - path: the URL path to send the request to.
// - data: the data to include in the request body.
//
// Returns:
// - string: the response body as a string.
// - error: any error that occurred during the request.
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
