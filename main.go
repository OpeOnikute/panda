package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"go-panda/scraper"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
	mailgun "github.com/mailgun/mailgun-go"
)

var mailRecipients = strings.Split(os.Getenv("MAIL_RECIPIENT"), ",")

func main() {

	err := godotenv.Load("/go/src/go-panda/.env") //full path needed cause of the cron
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// select site to scrape (randomly)
	siteURLS := []string{"https://www.worldwildlife.org/species/giant-panda", "https://www.photosforclass.com/search/panda"}
	// checkedSites := []string{}

	site := siteURLS[selectRandom(siteURLS)]

	// checkedSites = append(checkedSites, site)

	// get image from the site
	// if no image, get another site and start again
	// if image, send email containing image as attachment to me!
	response := scraper.Scrape(site)
	// Create output file
	findImage(response)
}

func selectRandom(siteURLS []string) int {
	rand.Seed(time.Now().Unix())
	index := rand.Intn(len(siteURLS))
	return index
}

func findImage(response *http.Response) {
	document, err := goquery.NewDocumentFromResponse(response)
	if err != nil {
		log.Fatal("Error loading HTTP response body. ", err)
	}

	validImages := []string{}
	validAlts := []string{}

	document.Find("img").Each(func(index int, element *goquery.Selection) {
		parent := element.Parent()
		parentTitle, parentTitleExists := parent.Attr("title") //enabling parsing on photclass.com
		imgSrc, srcExists := element.Attr("src")
		imgAlt, altExists := element.Attr("alt")
		re := regexp.MustCompile(`(?i)panda`) //case insensitive search
		pandaAlt := re.FindString(imgAlt)
		if srcExists && altExists && pandaAlt != "" {
			validImages = append(validImages, imgSrc)
			validAlts = append(validAlts, imgAlt)
		} else if parentTitleExists {
			validImages = append(validImages, imgSrc)
			validAlts = append(validAlts, parentTitle)
		}
	})

	if len(validImages) > 0 {
		selectedIndex := selectRandom(validImages)
		selectedImage := validImages[selectedIndex]

		selectedAlt := validAlts[selectedIndex]

		fileName := "/go/src/go-panda/images/" + selectedAlt + ".png"
		downloaded := downloadImage(fileName, selectedImage)

		if downloaded {
			//record image as downloaded
			fmt.Println("Downloaded image " + fileName)
			//send image as attachment
			sendMessage("opeonikuts@gmail.com", "Your daily dose of panda!", "Hi!, Find attached your daily picture of a panda!", mailRecipients, fileName)
		}
	} else {
		// send disappointing message. moving forward, should restart the routine and try again
		fmt.Println("No valid images found")
		sendMessage("opeonikuts@gmail.com", "Bad news, no panda dose today", "Hi!, Sadly we couldn't find any picture of a panda to send to you today. We'll be back tomorrow.", mailRecipients, "")
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

func sendMessage(sender, subject, body string, recipients []string, attachment string) {
	// Your available domain names can be found here:
	// (https://app.mailgun.com/app/domains)
	var domain = os.Getenv("MG_DOMAIN")

	// The API Keys are found in your Account Menu, under "Settings":
	// (https://app.mailgun.com/app/account/security)

	// starts with "key-"
	var privateAPIKey = os.Getenv("MG_API_KEY")
	mg := mailgun.NewMailgun(domain, privateAPIKey)

	message := mg.NewMessage(sender, subject, body)

	for i := 0; i < len(recipients); i++ {
		message.AddRecipient(recipients[i])
	}

	if attachment != "" {
		message.AddAttachment(attachment)
	}
	resp, id, err := mg.Send(message)

	if err != nil {
		//TODO: Just log failed emails in a file. No need for fatality.
		log.Fatal(err)
	}

	fmt.Printf("ID: %s Resp: %s\n", id, resp)
}
