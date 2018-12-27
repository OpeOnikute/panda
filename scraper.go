package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
	mailgun "github.com/mailgun/mailgun-go"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// select site to scrape (randomly)
	siteURLS := []string{"https://www.worldwildlife.org/species/giant-panda", "https://www.photosforclass.com/search/panda"}
	checkedSites := []string{}

	site := siteURLS[selectRandom(siteURLS)]

	checkedSites = append(checkedSites, site)

	// get image from the site
	// if no image, get another site and start again
	// if image, send email containing image as attachment to me!
	url := fixURL(site)

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
	defer response.Body.Close()

	// Create output file
	findImage(response.Body)
}

func selectRandom(siteURLS []string) int {
	rand.Seed(time.Now().Unix())
	index := rand.Intn(len(siteURLS))
	return index
}

func fixURL(href string) string {
	uri, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return uri.String()
}

func findImage(responseBody io.Reader) {
	document, err := goquery.NewDocumentFromReader(responseBody)
	if err != nil {
		log.Fatal("Error loading HTTP response body. ", err)
	}

	validImages := []string{}
	validAlts := []string{}

	document.Find("img").Each(func(index int, element *goquery.Selection) {
		imgSrc, srcExists := element.Attr("src")
		imgAlt, altExists := element.Attr("alt")
		re := regexp.MustCompile(`(?i)panda`) //case insensitive search
		pandaAlt := re.FindString(imgAlt)
		if srcExists && altExists && pandaAlt != "" {
			fmt.Println(imgSrc)
			validImages = append(validImages, imgSrc)
			validAlts = append(validAlts, imgAlt)
		}
	})

	fmt.Println(validImages)

	if len(validImages) > 0 {
		selectedIndex := selectRandom(validImages)
		selectedImage := validImages[selectedIndex]

		selectedAlt := validAlts[selectedIndex]

		fileName := selectedAlt + ".png"
		downloaded := downloadImage(fileName, selectedImage)

		if downloaded {
			//send image as attachment
			sendMessage("opeonikuts@gmail.com", "Your daily dose of panda!", "Hi!, Find attached your daily picture of a panda!", "opeyemionikute@yahoo.com", fileName)
			//record image as downloaded
		}
	} else {
		fmt.Println("No valid images found")
	}
}

func downloadImage(fileName string, url string) bool {
	response, e := http.Get(url)
	if e != nil {
		log.Fatal(e)
	}
	defer response.Body.Close()

	//open a file for writing
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Use io.Copy to just dump the response body to the file. This supports huge files
	_, err = io.Copy(file, response.Body)
	if err != nil {
		log.Fatal(err)
	}

	return true
}

func sendMessage(sender, subject, body, recipient, attachment string) {
	// Your available domain names can be found here:
	// (https://app.mailgun.com/app/domains)
	var domain = os.Getenv("MG_DOMAIN")

	// The API Keys are found in your Account Menu, under "Settings":
	// (https://app.mailgun.com/app/account/security)

	// starts with "key-"
	var privateAPIKey = os.Getenv("MG_API_KEY")
	mg := mailgun.NewMailgun(domain, privateAPIKey)

	message := mg.NewMessage(sender, subject, body, recipient)
	message.AddAttachment(attachment)
	resp, id, err := mg.Send(message)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("ID: %s Resp: %s\n", id, resp)
}
