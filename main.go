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

var maxRetries = 3 //retry operations only three times
var mailRecipients = strings.Split(os.Getenv("MAIL_RECIPIENT"), ",")
var base = getBase()

var successMessage = `
Hi!, Find attached your daily picture of a panda!

P.S. These messages are scheduled to go out at 10am everyday. If you receive it at any other time, something went wrong and we had to retry :)
`

var errorMessage = "Hi!, Sadly we couldn't find any picture of a panda to send to you today. We'll be back tomorrow."

func getBase() string {
	if os.Getenv("GO_ENV") == "machine" {
		return ""
	}

	return "/go/src/go-panda/"
}

func main() {
	getImage()
}

func getImage() bool {

	err := godotenv.Load(base + ".env") //full path needed cause of the cron
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
	findImage(response, 0)

	return true
}

func selectRandom(siteURLS []string) int {
	rand.Seed(time.Now().Unix())
	index := rand.Intn(len(siteURLS))
	return index
}

func findImage(response *http.Response, retryCount int) bool {
	document, err := goquery.NewDocumentFromResponse(response)
	if err != nil {
		log.Println("Error loading HTTP response body. ", err)
		if retryCount < maxRetries {
			log.Println("Retrying..")
			return findImage(response, retryCount+1)
		}
		return false
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

		fileName := base + "images/" + selectedAlt + ".png"
		downloaded := downloadImage(fileName, selectedImage, 0)

		if downloaded {
			//record image as downloaded
			fmt.Println("Downloaded image " + fileName)
			//send image as attachment
			sendMessage("opeonikuts@gmail.com", "Your daily dose of panda!", successMessage, mailRecipients, fileName)
		} else {
			//restart
			if retryCount < 3 {
				log.Println("Retrying..")
				return findImage(response, retryCount+1)
			}
			return false
		}
	} else {
		// send disappointing message. moving forward, should restart the routine and try again
		fmt.Println("No valid images found")
		sendMessage("opeonikuts@gmail.com", "Bad news, no panda dose today", errorMessage, mailRecipients, "")
	}

	return true
}

func downloadImage(fileName string, url string, retryCount int) bool {
	response, e := http.Get(url)
	if e != nil {
		//if there's no response just fail to avoid an infinite loop if the external url is down
		log.Fatal(e)
	}
	defer response.Body.Close()

	//open a file for writing
	file, err := os.Create(fileName)
	if err != nil {
		log.Println(err)
		if retryCount < maxRetries {
			log.Println("Retrying..")
			return downloadImage(fileName, url, retryCount+1)
		}
		return false
	}
	defer file.Close()

	// Use io.Copy to just dump the response body to the file. This supports huge files
	_, err = io.Copy(file, response.Body)
	if err != nil {
		log.Println(err)
		if retryCount < maxRetries {
			log.Println("Retrying..")
			return downloadImage(fileName, url, retryCount+1)
		}
		return false
	}

	return true
}

func sendMessage(sender, subject, body string, recipients []string, attachment string) bool {
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
		fmt.Println("Could not send message. Retrying..", err)
		return sendMessage(sender, subject, body, recipients, attachment)
	}

	fmt.Printf("ID: %s Resp: %s\n", id, resp)
	return true
}
