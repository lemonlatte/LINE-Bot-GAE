package linebot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
	"io/ioutil"
	"net/http"
)

type LineEventContent struct {
	From               string        `json:"from"`
	ContentType        int           `json:"contentType"`
	ToType             int           `json:"toType"`
	Text               string        `json:"text"`
	Params             []interface{} `json:"params"`
	OriginalContentUrl string        `json:"originalContentUrl"`
	PreviewImageUrl    string        `json:"previewImageUrl"`
}

type LineEvent struct {
	From        string           `json:"from"`
	FromChannel int              `json:"fromChannel"`
	To          []string         `json:"to"`
	ToChannel   int              `json:"toChannel"`
	EventType   string           `json:"eventType"`
	Id          string           `json:"id"`
	Content     LineEventContent `json:"content"`
}

type LineCallbackBody struct {
	Result []LineEvent
}

const LINE_ENDPOINT = "https://trialbot-api.line.me"

var LINE_HEADERS = map[string]string{
	"X-Line-ChannelID":             "",
	"X-Line-ChannelSecret":         "",
	"X-Line-Trusted-User-With-ACL": "",
}

func init() {
	http.HandleFunc("/", home)
	http.HandleFunc("/callback", lineBotCallback)
}

func lineBotCallback(w http.ResponseWriter, r *http.Request) {
	var lineCBBody LineCallbackBody
	ctx := appengine.NewContext(r)
	if r.Method == "POST" {
		body, _ := ioutil.ReadAll(r.Body)
		log.Infof(ctx, string(body))
		err := json.Unmarshal(body, &lineCBBody)
		if err != nil {
			log.Errorf(ctx, err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		lineApi := &LineAPIProxy{r}

		result := lineCBBody.Result[0]
		switch result.EventType {
		case "138311609100106403":
			log.Infof(ctx, "Receive Operation %+v", result)
			to := []string{result.Content.Params[0].(string)}
			err := lineApi.sendText(to, "Welcome to Jim's Bot.")
			if err != nil {
				log.Errorf(ctx, err.Error())
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
		case "138311609000106303":
			log.Infof(ctx, "Receive Message %+v", result)
			to := []string{result.Content.From}
			err := lineApi.sendText(to, fmt.Sprintf("只會學你: %s", result.Content.Text))
			if err != nil {
				log.Errorf(ctx, err.Error())
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
		}
		fmt.Fprintf(w, "")
	} else {
		http.Error(w, "", http.StatusNotFound)
	}
}

type LineAPIProxy struct {
	*http.Request
}

func (lineApi *LineAPIProxy) sendText(to []string, message string) (err error) {
	content := LineEventContent{
		ContentType: 1,
		ToType:      1,
		Text:        message,
	}
	err = lineApi.sendEvent(to, content)
	return
}

func (lineApi *LineAPIProxy) sendEvent(to []string, content LineEventContent) (err error) {
	data := LineEvent{
		To:        to,
		ToChannel: 1383378250,
		EventType: "138311608800106203",
		Content:   content,
	}

	b, err := json.Marshal(&data)
	if err != nil {
		return
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s:%s", LINE_ENDPOINT, "/v1/events"), bytes.NewBuffer(b))
	if err != nil {
		return
	}
	req.Header.Add("Content-Type", "application/json")
	for key, val := range LINE_HEADERS {
		req.Header.Add(key, val)
	}
	ctx := appengine.NewContext(lineApi.Request)
	tr := &urlfetch.Transport{Context: ctx}
	resp, err := tr.RoundTrip(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		log.Errorf(ctx, string(body))
	}
	return
}

func home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Line Bot Test!")
}
