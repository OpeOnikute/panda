package panda

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	mailgun "github.com/mailgun/mailgun-go/v4"
	"github.com/opeonikute/panda/scraper"
)

var maxRetries = 1 //retry operations only three times

var successMessage = `
Hello panda lover! 

Attached, you will find today's panda.

You can also see him/her on the cover of today's magazine at https://daily-panda.site 🤩🤩

P.S. These messages are scheduled to go out at 8am GMT everyday. If you receive it at any other time, something went wrong and we had to retry :)
`

var errorMessage = "Hi!, Sadly we couldn't find any picture of a panda to send to you today. We'll be back tomorrow."

const emailSender = "no-reply@opeonikute.dev"

type cdUploadResponse struct {
	PublicId     string `json:"public_id"`
	Version      uint   `json:"version"`
	ResourceType string `json:"resource_type"`
	Format       string `json:"format"`
	Size         int    `json:"bytes"`
	URL          string `json:"url"`
	SecureURL    string `json:"secure_url"`
}

// GoPanda ...
type GoPanda struct {
	Config     Settings
	DB         db
	SourceSite string
}

// ErrPandaNotFound is thrown when we don't get a panda of the day for a specific date
var ErrPandaNotFound = errors.New("no panda was found")

// Run exposes the main functionality of the package
func (g *GoPanda) Run(retryCount int) bool {

	// select site to scrape (randomly)
	siteURLS := []string{"https://www.worldwildlife.org/species/giant-panda", "https://www.photosforclass.com/search/panda", "https://www.photosforclass.com/search/panda/2", "https://www.photosforclass.com/search/panda/3", "https://www.photosforclass.com/search/panda/4"}
	// checkedSites := []string{}

	site := siteURLS[selectRandom(siteURLS)]

	// ideally would want to check the database for existing panda of the day.
	// but don't want to make a database query for this every time.
	// instead we can make the panda of the day idempotent.

	// get image from the site
	// if no image, get another site and start again
	// if image, send email containing image as attachment to me!
	response := scraper.Scrape(site)
	g.SourceSite = site

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

			fmt.Println("Downloaded image " + fileName)

			if g.Config.SendMail {
				// cast config interface to string and split into array
				var mailRecipients = strings.Split(g.Config.MailRecipients, ",")

				_, err = g.sendMessage(emailSender, "Your daily dose of panda!", successMessage, mailRecipients, fileName, downloaded)

				if err != nil {
					fmt.Println("Couldn't send email.")
					fmt.Println(err)
				}
			}

			_, err = g.saveImage(fileName, downloaded)

			if err != nil {
				fmt.Println("Couldn't save image.")
				fmt.Println(err)
			}

			return true
		}

		//retry
		if retryCount < 3 {
			log.Println("Retrying..")
			return g.findImage(response, retryCount+1)
		}

		return false
	}

	// TODO: Add some sort of monitoring, if this is important enough.
	fmt.Println("No valid images found. Doing nothing...")
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

func (g *GoPanda) sendMessage(sender, subject, body string, recipients []string, fileName string, attachment []byte) (bool, error) {

	domain := g.Config.MgDomain
	privateAPIKey := g.Config.MgKey

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

	resp, _, err := mg.Send(ctx, message)

	if err != nil {
		fmt.Println("Could not send message.", err)
		return false, err
	}

	fmt.Printf("Sent email to recipients. Resp: %s\n", resp)

	return true, nil
}

func (g *GoPanda) saveImage(fileName string, attachment []byte) (bool, error) {

	url, err := g.uploadImageToCloudinary(fileName, attachment)

	if err != nil {
		return false, err
	}

	// save to database
	_, err = g.savePOD(fileName, g.SourceSite, url)

	if err != nil {
		return false, err
	}

	return true, nil
}

// GetPOD fetches the panda of a specific day.
// tm - allows you specify the specific date so you can call time.Now() and pass that in.
func (g *GoPanda) GetPOD(tm time.Time) (Entry, error) {
	// connect to database
	var mongoURL = g.Config.MongoURL
	var mongoDB = g.Config.MongoDB
	var en Entry

	err := g.DB.Connect(mongoURL, mongoDB)
	if err != nil {
		return en, err
	}

	dt := GetDate(tm)
	en, err = g.DB.FindPOD(dt)

	if err != nil {
		return en, err
	}

	// confirm a panda was actually found.
	if en.Source == "" && en.FileName == "" && en.URL == "" {
		return en, ErrPandaNotFound
	}

	return en, nil
}

func (g *GoPanda) savePOD(fileName, source, url string) (Entry, error) {
	// connect to database
	var mongoURL = g.Config.MongoURL
	var mongoDB = g.Config.MongoDB

	en := Entry{
		FileName: fileName,
		Source:   source,
		URL:      url,
	}

	err := g.DB.Connect(mongoURL, mongoDB)
	// TODO: close the db connection
	if err != nil {
		return en, err
	}

	// save new entry
	_, err = g.DB.InsertPOD(en)
	if err != nil {
		return en, err
	}

	fmt.Println("Saved image to database.")
	return en, err
}

func (g *GoPanda) generateCDSignature(params map[string]interface{}) string {

	// Instructions for generating the signature can be found here:
	// https://cloudinary.com/documentation/upload_images#generating_authentication_signatures
	secret := g.Config.CdAPISecret
	arr := []string{}

	// Separate the parameter names from their values with an = and
	// join the parameter/value pairs together with an &.
	for key, elem := range params {
		arr = append(arr, fmt.Sprintf("%s=%s", key, elem))
	}

	// Sort the elements alphabetically
	sort.Strings(arr)

	str := strings.Join(arr, "&") + secret

	// create the sha1 hash and return as a string
	sha := sha1.New()
	sha.Write([]byte(str))

	return hex.EncodeToString(sha.Sum(nil))
}

// CreateGif makes a gif on the Cloudinary account with all images
// with the tag 'panda_image'
func (g *GoPanda) CreateGif() (string, error) {
	cloudName := g.Config.CdCloudName
	apiKey := g.Config.CdAPIKey

	tag := "panda_image"
	timestamp := int32(time.Now().Unix())
	transf := "dl_800,w_400,h_400,c_fill_pad,g_auto"

	signParams := make(map[string]interface{})
	signParams["transformation"] = transf
	signParams["tag"] = tag
	signParams["timestamp"] = fmt.Sprintf("%d", timestamp)

	sign := g.generateCDSignature(signParams)

	cloudinaryURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/multi", cloudName)

	payloadStr := `{"tag":"%s", "api_key": "%s", "transformation": "%s", "signature":"%s", "timestamp":"%d"}`

	// send POST request with JSON payload
	payload := []byte(fmt.Sprintf(payloadStr, tag, apiKey, transf, sign, timestamp))
	req, err := http.NewRequest("POST", cloudinaryURL, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	fmt.Println("Response Status:", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)
	res := string(body)
	return res, nil
}

func (g *GoPanda) uploadImageToCloudinary(imgName string, imgBody []byte) (string, error) {

	cloudName := g.Config.CdCloudName
	preset := g.Config.CdUploadPreset

	cloudinaryURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", cloudName)

	params := map[string]string{
		"upload_preset": preset,
	}

	// TODO: Set a timeout parameter on this
	client := &http.Client{}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", imgName)
	if err != nil {
		return "", err
	}
	part.Write(imgBody)

	// add other fields to form
	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", cloudinaryURL, body)

	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := client.Do(req)

	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	// decode response
	dec := json.NewDecoder(res.Body)
	upInfo := new(cdUploadResponse)

	if err := dec.Decode(upInfo); err != nil {
		log.Println(err)
		return "", err
	}

	fmt.Println("Uploaded image to cloudinary.")
	return upInfo.SecureURL, nil
}
