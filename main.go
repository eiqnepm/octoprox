package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"
)

var (
	haURL   *url.URL
	client  = &http.Client{Timeout: 10 * time.Second}
	token   = os.Getenv("TOKEN")
	sensors = strings.Split(os.Getenv("SENSORS"), ",")
)

type state struct {
	State      string
	Attributes struct {
		UnitOfMeasurement string `json:"unit_of_measurement"`
		FriendlyName      string `json:"friendly_name"`
	}
}

func getStates() []state {
	var (
		wg     sync.WaitGroup
		m      sync.Mutex
		states []state
	)

	for _, sensor := range sensors {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, err := http.NewRequest("GET", haURL.JoinPath("api", "states", sensor).String(), nil)
			if err != nil {
				log.Println(err)
				return
			}

			req.Header.Add("Authorization", "Bearer "+token)
			resp, err := client.Do(req)
			if err != nil {
				log.Println(err)
				return
			}

			defer func() {
				if err := resp.Body.Close(); err != nil {
					log.Println(err)
				}
			}()

			if resp.StatusCode != 200 {
				log.Println(resp.Status)
				return
			}

			b, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Println(err)
				return
			}

			var body state
			if err := json.Unmarshal(b, &body); err != nil {
				log.Println(err)
				return
			}

			func() {
				m.Lock()
				defer m.Unlock()
				states = append(states, body)
			}()
		}()
	}

	wg.Wait()
	return states
}

func main() {
	var err error
	haURL, err = url.Parse(os.Getenv("URL"))
	if err != nil {
		log.Fatal(err)
	}

	index, err := template.ParseFiles("index.html")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		index.Execute(w, nil)
	})

	states, err := template.ParseFiles("states.html")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/states", func(w http.ResponseWriter, req *http.Request) {
		s := getStates()
		sort.Slice(s, func(a, b int) bool {
			return s[a].Attributes.FriendlyName < s[b].Attributes.FriendlyName
		})

		states.Execute(w, struct {
			States []state
		}{s})
	})

	http.ListenAndServe(":3000", nil)
}
