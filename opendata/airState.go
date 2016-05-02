package opendata

import (
	"encoding/json"
	"golang.org/x/net/context"
	"google.golang.org/appengine/urlfetch"
	"io/ioutil"
	"net/http"
	"time"
)

const DATA_URI = "http://opendata2.epa.gov.tw/AQX.json?format=json"

type PublishTime struct {
	Time time.Time
}

func (pt *PublishTime) MarshalJSON() ([]byte, error) {
	b := pt.Time.Format("2006-01-02 15:04")
	return []byte(b), nil
}

func (pt *PublishTime) UnmarshalJSON(b []byte) (err error) {
	loc, _ := time.LoadLocation("Asia/Taipei")
	t, err := time.ParseInLocation("2006-01-02 15:04", string(b[1:len(b)-1]), loc)
	if err == nil {
		*pt = PublishTime{Time: t}
	}
	return
}

type AirState struct {
	County      string
	SiteName    string
	PSI         string
	PM10        string
	PM2_5       string `json:"PM2.5"`
	O3          string
	CO          string
	NO2         string
	SO2         string
	PublishTime PublishTime
}

func GetAirState(ctx context.Context) (asList []AirState, err error) {
	req, err := http.NewRequest("GET", DATA_URI, nil)
	if err != nil {
		return
	}
	req.Header.Add("Content-Type", "application/json")

	tr := &urlfetch.Transport{Context: ctx}
	resp, err := tr.RoundTrip(req)

	if err != nil {
		return
	}

	asList = []AirState{}
	b, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(b, &asList)
	return
}
