package main

import (
	"fmt"
	"io/ioutil"
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
var base = getBase()

var successMessage = `
Hi!, Find attached your daily picture of a panda!

P.S. These messages are scheduled to go out at 11am everyday. If you receive it at any other time, something went wrong and we had to retry :)
`

var errorMessage = "Hi!, Sadly we couldn't find any picture of a panda to send to you today. We'll be back tomorrow."

func getBase() string {
	if os.Getenv("GO_ENV") == "machine" {
		return ""
	}

	return "/go/src/go-panda/"
}

func main() {
	getImage(0)
}

func getImage(retryCount int) bool {

	err := godotenv.Load(base + ".env") //full path needed cause of the cron
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// select site to scrape (randomly)
	siteURLS := []string{"https://www.worldwildlife.org/species/giant-panda", "https://www.photosforclass.com/search/panda", "https://www.photosforclass.com/search/panda/2", "https://www.photosforclass.com/search/panda/3", "https://www.photosforclass.com/search/panda/4"}
	// checkedSites := []string{}

	site := siteURLS[selectRandom(siteURLS)]

	// checkedSites = append(checkedSites, site)

	// get image from the site
	// if no image, get another site and start again
	// if image, send email containing image as attachment to me!
	response := scraper.Scrape(site)
	// Create output file
	imageSent := findImage(response, 0)

	if !imageSent && retryCount < 3 {
		return getImage(retryCount + 1)
	}
	return true
}

func selectRandom(siteURLS []string) int {
	rand.Seed(time.Now().Unix())
	index := rand.Intn(len(siteURLS))
	return index
}

func findImage(response *http.Response, retryCount int) bool {

	var mailRecipients = strings.Split(os.Getenv("MAIL_RECIPIENT"), ",")

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

		downloaded, contentType := downloadImage(selectedImage, 0)

		if len(downloaded) > 0 {

			fileExt := "." + strings.Replace(contentType, "image/", "", 1)
			fileName := selectedAlt + fileExt

			//record image as downloaded
			fmt.Println("Downloaded image " + fileName)
			//send image as attachment
			sendMessage("opeonikuts@gmail.com", "Your daily dose of panda!", successMessage, mailRecipients, fileName, downloaded, 0)
			return true
		}

		//retry
		if retryCount < 3 {
			log.Println("Retrying..")
			return findImage(response, retryCount+1)
		}
		return false

	}

	// send disappointing message. moving forward, should restart the routine and try again
	fmt.Println("No valid images found")
	sendMessage("opeonikuts@gmail.com", "Bad news, no panda dose today", errorMessage, mailRecipients, "", nil, 0)
	return false
}

func downloadImage(url string, retryCount int) ([]byte, string) {
	response, e := http.Get(url)
	if e != nil {
		//if there's no response just fail to avoid an infinite loop if the external url is down
		log.Fatal(e)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		if retryCount < maxRetries {
			log.Println("Retrying..")
			return downloadImage(url, retryCount+1)
		}
		return nil, ""
	}

	contentType := response.Header["Content-Type"][0]
	return body, contentType
}

func sendMessage(sender, subject, body string, recipients []string, fileName string, attachment []byte, retryCount int) bool {
	var domain = os.Getenv("MG_DOMAIN")
	var privateAPIKey = os.Getenv("MG_API_KEY")

	mg := mailgun.NewMailgun(domain, privateAPIKey)

	message := mg.NewMessage(sender, subject, body)

	for i := 0; i < len(recipients); i++ {
		message.AddRecipient(recipients[i])
	}

	//send as byte array to prevent trying to save and read from disk, which is buggy.
	if fileName != "" && len(attachment) > 0 {
		message.AddBufferAttachment(fileName, attachment)
	}
	resp, id, err := mg.Send(message)

	if err != nil {
		fmt.Println("Could not send message.", err)
		if retryCount < 3 {
			log.Println("Retrying..")
			return sendMessage(sender, subject, body, recipients, fileName, attachment, retryCount+1)
		}
		log.Fatal(err)
	}

	fmt.Printf("ID: %s Resp: %s\n", id, resp)
	return true
}
