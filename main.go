package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-lambda-go/lambda"
	mailgun "github.com/mailgun/mailgun-go/v4"
	"github.com/opeonikute/go-panda/scraper"
)

var maxRetries = 1 //retry operations only three times

var successMessage = `
Hi!, Find attached your daily picture of a panda!

P.S. These messages are scheduled to go out at 11am everyday. If you receive it at any other time, something went wrong and we had to retry :)
`

var errorMessage = "Hi!, Sadly we couldn't find any picture of a panda to send to you today. We'll be back tomorrow."

const emailSender = "no-reply@opeonikute.dev"

func main() {
	lambda.Start(handleRequest)
}

func handleRequest(ctx context.Context) (bool, error) {
	goPanda := GoPanda{
		config: Config{
			MgDomain:       os.Getenv("MG_DOMAIN"),
			MgKey:          os.Getenv("MG_API_KEY"),
			MailRecipients: os.Getenv("MAIL_RECIPIENT"),
		},
	}
	result := goPanda.Run(0)
	if result != true {
		return false, errors.New("Email was not sent. Image function returned false")
	}
	return true, nil
}

// GoPanda ...
type GoPanda struct {
	config Config
}

// Run exposes the main functionality of the package
func (g *GoPanda) Run(retryCount int) bool {

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
	imageSent := g.findImage(response, 0)

	if !imageSent && retryCount < 3 {
		return g.Run(retryCount + 1)
	}
	return true
}

func selectRandom(siteURLS []string) int {
	rand.Seed(time.Now().Unix())
	index := rand.Intn(len(siteURLS))
	return index
}

func (g *GoPanda) findImage(response *http.Response, retryCount int) bool {

	// cast config interface to string and split into array
	var mailRecipients = strings.Split(g.config.MailRecipients, ",")

	document, err := goquery.NewDocumentFromResponse(response)
	if err != nil {
		log.Println("Error loading HTTP response body. ", err)
		if retryCount < maxRetries {
			log.Println("Retrying..")
			return g.findImage(response, retryCount+1)
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

		if selectedAlt == "" {
			selectedAlt = "panda"
		}

		downloaded, contentType := g.downloadImage(selectedImage, 0)

		if len(downloaded) > 0 {

			fileExt := "." + strings.Replace(contentType, "image/", "", 1)
			fileName := selectedAlt + fileExt

			//record image as downloaded
			fmt.Println("Downloaded image " + fileName)
			//send image as attachment
			g.sendMessage(emailSender, "Your daily dose of panda!", successMessage, mailRecipients, fileName, downloaded, 0)
			return true
		}

		//retry
		if retryCount < 3 {
			log.Println("Retrying..")
			return g.findImage(response, retryCount+1)
		}
		return false

	}

	// send disappointing message. moving forward, should restart the routine and try again
	fmt.Println("No valid images found")
	g.sendMessage(emailSender, "Bad news, no panda dose today", errorMessage, mailRecipients, "", nil, 0)
	return false
}

func (g *GoPanda) downloadImage(url string, retryCount int) ([]byte, string) {
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
			return g.downloadImage(url, retryCount+1)
		}
		return nil, ""
	}

	contentType := response.Header["Content-Type"][0]
	return body, contentType
}

func (g *GoPanda) sendMessage(sender, subject, body string, recipients []string, fileName string, attachment []byte, retryCount int) bool {

	domain := g.config.MgDomain
	privateAPIKey := g.config.MgKey

	mg := mailgun.NewMailgun(domain, privateAPIKey)
	mg.SetAPIBase(mailgun.APIBaseEU)

	message := mg.NewMessage(sender, subject, body)

	for i := 0; i < len(recipients); i++ {
		message.AddRecipient(recipients[i])
	}

	//send as byte array to prevent trying to save and read from disk, which is buggy.
	if fileName != "" && len(attachment) > 0 {
		message.AddBufferAttachment(fileName, attachment)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	resp, id, err := mg.Send(ctx, message)

	if err != nil {
		fmt.Println("Could not send message.", err)
		if retryCount < 3 {
			log.Println("Retrying..")
			return g.sendMessage(sender, subject, body, recipients, fileName, attachment, retryCount+1)
		}
		log.Fatal(err)
	}

	fmt.Printf("ID: %s Resp: %s\n", id, resp)
	return true
}
