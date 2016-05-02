package linebot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
	"io/ioutil"
	"net/http"
	"opendata"
	"strconv"
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

var emojiMask = "\xf3\xbe\x8c\xae"

const LINE_ENDPOINT = "https://trialbot-api.line.me"

var LINE_HEADERS = map[string]string{
	"X-Line-ChannelID":             "",
	"X-Line-ChannelSecret":         "",
	"X-Line-Trusted-User-With-ACL": "",
}

func init() {
	http.HandleFunc("/", home)
	http.HandleFunc("/update/airState", updateAirState)
	http.HandleFunc("/callback", lineBotCallback)
}

func getAirStateString(airState map[string]float64) string {
	var psiReport, pm2_5Report string
	psi, ok := airState["PSI"]
	if ok {
		var level string
		if psi <= 50 {
			level = "Green"
		} else if psi <= 100 {
			level = "Yellow"
		} else if psi <= 199 {
			level = "Red"
		} else if psi <= 299 {
			level = "Purple"
		} else {
			level = "Brown"
		}
		psiReport = fmt.Sprintf("PSI: %.0f (%s)", psi, level)
	}
	pm2_5, ok := airState["PM2.5"]
	if ok {
		var level string
		if pm2_5 <= 35 {
			level = "Green"
		} else if pm2_5 <= 53 {
			level = "Yellow"
		} else if pm2_5 <= 70 {
			level = "Red"
		} else {
			level = "Purple"
		}
		pm2_5Report = fmt.Sprintf("PM2.5: %.0f (%s)", pm2_5, level)
	}
	return fmt.Sprintf("%s\n%s", psiReport, pm2_5Report)
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
			// log.Infof(ctx, "Receive Operation %+v", result)
			to := []string{result.Content.Params[0].(string)}
			err := lineApi.sendText(to, "Welcome to Jim's Bot.")
			if err != nil {
				log.Errorf(ctx, err.Error())
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
		case "138311609000106303":
			var err error
			// log.Infof(ctx, "Receive Message %+v", result)
			to := []string{result.Content.From}
			if result.Content.ContentType == 1 {
				text := result.Content.Text
				log.Infof(ctx, "Text Buffer: %+v", []byte(text))
				switch text {
				case "Air", "air":
					airState, err := getAirState(ctx)
					if err == nil {
						err = lineApi.sendText(to, getAirStateString(airState))
					}
				default:
					err = lineApi.sendText(to, fmt.Sprintf("%s", text))
				}
			} else {
				err = lineApi.sendText(to, fmt.Sprintf("%s", "喔喔喔，看不懂，請用文字跟我溝通"))
			}
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

func updateAirState(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	asList, err := opendata.GetAirState(ctx)

	if err != nil {
		log.Errorf(ctx, err.Error())
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	asKeys := []*datastore.Key{}
	for _, as := range asList {
		keyName := fmt.Sprintf("%s|%s|%s", as.County, as.SiteName,
			as.PublishTime.Time.Format("2006-01-02T15:04"))
		asKeys = append(asKeys, datastore.NewKey(ctx, "AirState", keyName, 0, nil))
	}
	_, err = datastore.PutMulti(ctx, asKeys, asList)
	if err != nil {
		log.Errorf(ctx, err.Error())
	}
	fmt.Fprintf(w, "")
	return
}

func strToFloat(s string) (f float64, err error) {
	f, err = strconv.ParseFloat(s, 64)
	return
}

func getAirState(ctx context.Context) (map[string]float64, error) {
	q := datastore.NewQuery("AirState").Filter("County =", `臺北市`).Filter("SiteName =", `大同`).Order("-PublishTime.Time")
	var as opendata.AirState
	for t := q.Run(ctx); ; {
		key, err := t.Next(&as)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if key != nil {
			break
		}
	}
	psi, err := strToFloat(as.PSI)
	if err != nil {
		return nil, err
	}
	pm2_5, err := strToFloat(as.PM2_5)
	if err != nil {
		return nil, err
	}
	return map[string]float64{
		"PSI":   psi,
		"PM2.5": pm2_5,
	}, nil
}

func home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Line Bot Test!")
}
