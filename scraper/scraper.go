package scraper

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Scrape makes a basic HTTP GET request to a site and returns the response.
func Scrape(siteURL string) *http.Response {

	url := fixURL(siteURL)

	if url == "" {
		fmt.Println("Site URL invalid.")
		os.Exit(1)
	}

	fmt.Println("Getting images from " + url)
	tlsConfig := &tls.Config{ // The &thing{a: b} syntax is equivalent to
		InsecureSkipVerify: true, // new(thing(a: b)) in other languages.
	}

	transport := &http.Transport{ // And we take that tlsConfig object we instantiated
		TLSClientConfig: tlsConfig, // and use it as the value for another new object's
	}

	// Create HTTP client with timeout & ignore https
	client := &http.Client{
		Timeout:   100 * time.Second,
		Transport: transport,
	}

	// Create and modify HTTP request before sending
	request, err := http.NewRequest("GET", url, nil)

	if err != nil {
		log.Fatal(err)
	}
	request.Header.Set("User-Agent", "Not Firefox")

	// Make HTTP GET request
	response, err := client.Do(request)

	if err != nil {
		log.Fatal(err)
	}

	return response
}
func fixURL(href string) string {
	uri, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return uri.String()
}
