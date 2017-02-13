package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/parnurzeal/gorequest"
)

type Feed struct {
	XMLName xml.Name `xml:"feed"`
	Id      string   `xml:"id"`
	Title   string   `xml:"title"`
	Entries []Entry  `xml:"entry"`
}

type Entry struct {
	XMLName  xml.Name   `xml:"entry"`
	Id       string     `xml:"id"`
	Title    string     `xml:"title"`
	Content  string     `xml:"content"`
	Author   Author     `xml:"author"`
	Category Category   `xml:"category"`
	Updated  customTime `xml:"updated"`
}

type customTime struct {
	time.Time
}

func (c *customTime) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var v string
	d.DecodeElement(&v, &start)
	parse, _ := time.Parse(time.RFC3339Nano, v)
	*c = customTime{parse}
	return nil
}

func (c *customTime) UnmarshalXMLAttr(attr xml.Attr) error {
	fmt.Printf("time:%v\n", attr.Value)
	parse, _ := time.Parse(time.RFC3339Nano, attr.Value)
	*c = customTime{parse}
	return nil
}

type Category struct {
	XMLName xml.Name `xml:"category"`
	Term    string   `xml:"term,attr"`
}

type Author struct {
	XMLName xml.Name `xml:"author"`
	Name    string   `xml:"name"`
	Email   string   `xml:"email"`
}

func (e Entry) isComment() bool {
	return e.Category.Term == "comment"
}

func GetFeed() Feed {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", os.Getenv("JIRA_URL")+"/activity?maxItems=100", nil)
	req.SetBasicAuth(os.Getenv("JIRA_USERNAME"), os.Getenv("JIRA_PASSWORD"))
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	var f Feed
	xml.Unmarshal(body, &f)
	return f
}

func HtmlToSlackMarkup(s string) string {
	//replace <a href> with the slack syntax
	rp := regexp.MustCompile(`<a[^>]*href="(?P<URL>[^"]*)"[^>]*>(?P<LinkText>[^<]*)</a>`)
	output := rp.ReplaceAllString(s, "<$URL|$LinkText>")

	//honor the <p> tags
	output = strings.Replace(output, `<p>`, `\n\n`, -1)

	//remove all the html tags
	rp2 := regexp.MustCompile(`<[^>|]*>`)
	output = rp2.ReplaceAllString(output, "")

	//clean up white space - two or more times
	rp3 := regexp.MustCompile(`\s{2,}`)
	output = rp3.ReplaceAllString(output, " ")

	//clean up white space - two or more times
	rp4 := regexp.MustCompile(`\r\n|\n|\r`)
	output = rp4.ReplaceAllString(output, `\n`)

	return output

}

func SendRichPostToSlack(title string, comment string) {
	message := `
	{
	  "attachments": [
	    {
	      "fallback":"%v",
	      "pretext":"%v",
	      "color":"#0000D0",
	      "fields":[
	        {
	          "value":"%v",
	          "short":false
	        }
	      ]
	    }
	  ]
	}`

	payload := fmt.Sprintf(message, title, title, comment)
	fmt.Println("payload:")
	fmt.Println(payload)
	SendPayload(payload)
}

func SendPayload(payload string) {
	request := gorequest.New()
	_, _, errs := request.Post(os.Getenv("SLACK_WEBHOOK")).
		Send(payload).
		End()
	if errs != nil {
		log.Fatal(errs)
	}
}

func SyncSlackMessages(anchor time.Time) {

	log.Print("Syncing Slack Messages")
	feed := GetFeed()
	for _, e := range feed.Entries {
		if e.isComment() && e.Updated.After(anchor) {
			log.Print("-----------------------")
			log.Print("Identified new message:", e.Id)
			title := HtmlToSlackMarkup(e.Title)
			content := HtmlToSlackMarkup(e.Content)
			log.Print("Sanitized title: ", title)
			log.Print("Sanitized content:", content)
			SendRichPostToSlack(title, content)
			log.Print("-----------------------")
		}
	}
}

func main() {
	//do a full sync of legacy mesages
	lastSyncTime := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	SyncSlackMessages(lastSyncTime)
	lastSyncTime = time.Now()

	//now do every 10 seconds
	c := time.Tick(10 * time.Second)
	for now := range c {
		SyncSlackMessages(lastSyncTime)
		lastSyncTime = now
	}
}
