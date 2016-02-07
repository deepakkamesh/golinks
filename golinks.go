package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type redirect struct {
	Shortname string
	Url       string
	Requests  int32
}

type settings struct {
	redirects []*redirect
	filename  string
}

func main() {
	port := flag.String("http_port", "8080", "Port number to listen")
	configFile := flag.String("config", "redirects.json", "Configuration filename")
	flag.Parse()
	log.Printf("Starting golinks...")
	r := newRedirector(*configFile)
	r.readConfig()

	http.HandleFunc("/add/", r.addLink)
	http.HandleFunc("/list/", r.getLinks)
	http.HandleFunc("/del/", r.delLink)
	http.HandleFunc("/", r.redirect)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatal("Unable to listen %s", err)
	}
}

func newRedirector(file string) *settings {
	return &settings{
		filename: file,
	}
}

// readConfig attempts to marshal a saved config file. If none exists
// a new one is created.
func (m *settings) readConfig() {
	log.Printf("Reading configuration...")
	jsonBlob, err := ioutil.ReadFile(m.filename)
	if err != nil {
		log.Printf("No config file found. Using new config file.")
		return
	}
	if err := json.Unmarshal(jsonBlob, &m.redirects); err != nil {
		log.Printf("Error unmarshalling %s", err)
	}
}

//saveToDisk unmarshals the struct to a json config file.
func (m *settings) saveToDisk() error {
	b, err := json.Marshal(m.redirects)
	if err != nil {
		log.Printf("Error marshalling %s", err)
		return fmt.Errorf("error marshalling %s", err)
	}

	if err := ioutil.WriteFile(m.filename, b, 0644); err != nil {
		return fmt.Errorf("unable to open file %s", err)
	}
	log.Printf("saving to disk.")
	return nil
}

// getLinks displays all the current redirects.
func (m *settings) getLinks(w http.ResponseWriter, r *http.Request) {
	var s string
	for _, v := range m.redirects {
		s += fmt.Sprintf("Shortname: %s -> Url: %s  Count %d <BR>", v.Shortname, v.Url, v.Requests)
	}
	sendHtml(w, s)
}

// redirect redirects the shortname to the url.
func (m *settings) redirect(w http.ResponseWriter, r *http.Request) {
	req := strings.Split(r.URL.Path[1:], "/")
	sh := strings.Trim(req[0], " ")
	for _, v := range m.redirects {
		if v.Shortname == sh {
			//substitute shortname with real url
			req[0] = v.Url
			h := w.Header()
			h.Set("Cache-Control", "private, no-cache")
			http.Redirect(w, r, strings.Join(req, "/"), 302)
			break
		}
	}
	sendHtml(w, "Shortname "+sh+" not found!")
}

func (m *settings) delLink(w http.ResponseWriter, r *http.Request) {
	req := strings.Trim(r.URL.Path[5:], " ")
	for i, v := range m.redirects {
		if v.Shortname == req {
			m.redirects = append(m.redirects[:i], m.redirects[i+1:]...)
			if err := m.saveToDisk(); err != nil {
				http.Error(w, fmt.Sprintf("Internal error deleting redirect."), http.StatusInternalServerError)
			}
			break
		}
	}
	http.Redirect(w, r, "/list", 302)
}

func (m *settings) addLink(w http.ResponseWriter, r *http.Request) {

	// Validate if we recieved a good request
	var validReq = regexp.MustCompile(`^(http|https|ftp)/[a-z,A-Z,0-9]+/[a-z, ,A-Z,0-9,=,-,_,/,:,\.]+$`)
	if !validReq.MatchString(r.URL.Path[5:]) {
		sendHtml(w, html.EscapeString("Request should be of form /add/<protocol eg. http, https, ftp>/<shortname>/<redirect>"))
		return
	}

	req := strings.Split(r.URL.Path[5:], "/")

	// Sanitize input.
	for i, _ := range req {
		req[i] = strings.Trim(req[i], " ")
	}

	shortname := req[1]
	url := req[0] + "://" + strings.Join(req[2:], "/")

	// Verify if shortname already exists.
	for _, v := range m.redirects {
		if v.Shortname == shortname {
			sendHtml(w, "Shortname already points to "+v.Url)
			return
		}
	}
	// Add shortname, redirect to list.
	m.redirects = append(m.redirects, &redirect{
		Shortname: shortname,
		Url:       url,
		Requests:  0,
	})

	if err := m.saveToDisk(); err != nil {
		http.Error(w, fmt.Sprintf("Internal error saving redirect."), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/list", 302)
}

func sendHtml(w http.ResponseWriter, text string) {
	fmt.Fprintf(w, `<html>
                                <head>
                                <title>Redirects Setup</title>
                                </head>
                                <body>`)
	fmt.Fprintf(w, text)
	fmt.Fprintf(w, `</body>
                                </html>`)
}
