package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type Redirect struct {
	Shortname string
	Url       string
	Requests  int32
}

type Settings struct {
	Redirects []*Redirect
	Filename  string
}

func main() {
	port := flag.String("http_port", "8080", "Port number to listen")
	configFile := flag.String("config", "redirects.json", "Port number to listen")
	flag.Parse()
	log.Printf("Starting golinks...")
	r := NewRedirector(*configFile)
	r.ReadConfig()

	http.HandleFunc("/add/", r.AddLink)
	http.HandleFunc("/list/", r.GetLinks)
	http.HandleFunc("/del/", r.DelLink)
	http.HandleFunc("/", r.Redirect)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatal("Unable to listen %s", err)
	}
	for {
	}
}

func NewRedirector(file string) *Settings {
	return &Settings{
		Filename: file,
	}
}

func (m *Settings) ReadConfig() {
	log.Printf("reading configuration...")
	jsonBlob, err := ioutil.ReadFile(m.Filename)
	if err != nil {
		log.Printf("Unable to read configuration %s", err)
		return
	}
	if err := json.Unmarshal(jsonBlob, &m.Redirects); err != nil {
		log.Printf("error unmarshalling %s", err)
	}
}

func (m *Settings) SaveToDisk() {
	b, err := json.Marshal(m.Redirects)
	if err != nil {
		log.Printf("error marshalling %s", err)
	}

	if err := ioutil.WriteFile(m.Filename, b, 0644); err != nil {
		log.Fatalf("unable to open file %s", err)
	}
	log.Printf("saving to disk.")
}

func (m *Settings) GetLinks(w http.ResponseWriter, r *http.Request) {
	var s string
	for _, v := range m.Redirects {
		s += v.Shortname + "->" + v.Url + "<BR>"
	}
	SendHtml(w, s)
}

func (m *Settings) Redirect(w http.ResponseWriter, r *http.Request) {
	var url string
	req := strings.Split(r.URL.Path[1:], "/")
	args := strings.Join(req[1:], "/")
	sh := strings.Trim(req[0], " ")
	for _, v := range m.Redirects {
		if v.Shortname == sh {
			url = v.Url
			h := w.Header()
			h.Set("Cache-Control", "private, no-cache")
			http.Redirect(w, r, url+"/"+args, 302)
			break
		}
	}
	SendHtml(w, "Shortname "+sh+" not found!")

}

func (m *Settings) DelLink(w http.ResponseWriter, r *http.Request) {
	req := strings.Trim(r.URL.Path[5:], " ")
	for i, v := range m.Redirects {
		if v.Shortname == req {
			m.Redirects = append(m.Redirects[:i], m.Redirects[i+1:]...)
			m.SaveToDisk()
			break
		}
	}
	http.Redirect(w, r, "/list", 302)
}

func (m *Settings) AddLink(w http.ResponseWriter, r *http.Request) {
	req := strings.Split(r.URL.Path[5:], "|")

	// Sanitize input.
	for i, _ := range req {
		req[i] = strings.Trim(req[i], " ")
	}

	if len(req) >= 2 {
		// Ensure proper formatting for redirect url.
		var validUrl = regexp.MustCompile(`^[a-z, ,A-Z,0-9,-,_,/,:,\.]+$`)
		if !validUrl.MatchString(req[1]) {
			SendHtml(w, "Redirect url should be fully qualified URL http://something")
			return
		}
		// Verify if shortname already exists.
		for _, v := range m.Redirects {
			if v.Shortname == req[0] {
				SendHtml(w, "Shortname already points to "+v.Url)
				return
			}
		}
		// Add shortname, redirect to list.
		m.Redirects = append(m.Redirects, &Redirect{
			Shortname: req[0],
			Url:       "http://" + req[1],
			Requests:  0,
		})
		SendHtml(w, "Setting up redirect for  "+req[0]+" -> "+req[1])
		m.SaveToDisk()
		return
	}
	fmt.Fprintf(w, "Incorrect add format use add/<shortname>|url")
}

func SendHtml(w http.ResponseWriter, text string) {
	fmt.Fprintf(w, `<html>
				<head>
				<title>Redirects Setup</title>
				</head>
				<body>`)
	fmt.Fprintf(w, text)
	fmt.Fprintf(w, `</body>
				</html>`)
}
