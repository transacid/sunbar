// usr/bin/env go run $0 $@; exit
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

// <bitbar.title>Sunrise / Sunset countdown</bitbar.title>
// <bitbar.version>v1.0.0</bitbar.version>
// <bitbar.author>Boris Petersen</bitbar.author>
// <bitbar.author.github>transacid</bitbar.author.github>
// <bitbar.desc>Displays the time till next sunset or sunrise.</bitbar.desc>
// <bitbar.image>https://raw.githubusercontent.com/transacid/sunbar/main/screenshot.png</bitbar.image>
// <bitbar.dependencies>go</bitbar.dependencies>
// <bitbar.abouturl>https://github.com/transacid/sunbar</bitbar.abouturl>

func main() {
	data := getData()
	sunset := data.Results.Sunset
	sunrise := data.Results.Sunrise
	now := adjustedNow()

	switch {
	case sunrise.After(now) && sunset.After(now):
		fmt.Print(Printer("sunrise", eventDurationFromNow(sunrise), "sunset", eventDurationFromNow(sunset)))
	case sunrise.Before(now) && sunset.After(now):
		fmt.Print(Printer("sunset", eventDurationFromNow(sunset), "sunrise", eventDurationFromNow(sunrise.Add(time.Hour*24))))
	case sunrise.Before(now) && sunset.Before(now):
		fmt.Print(Printer("sunrise", eventDurationFromNow(sunrise.Add(time.Hour*24)), "sunset", eventDurationFromNow(sunset.Add(time.Hour*24))))
	}
}

func Printer(nowEvent, nowDuration, nextEvent, nextDuration string) string {
	return fmt.Sprintf(":%s: %s\n---\n:%s: %s | [symbolize = true]", nowEvent, nowDuration, nextEvent, nextDuration)
}

func getData() SunData {
	var data SunData
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	f, err := os.Open(fmt.Sprintf("%s/.sun.json", home))
	if err != nil {
		panic(err)
	}
	stat, err := f.Stat()
	if err != nil {
		panic(err)
	}
	mtime := stat.ModTime()
	now := time.Now().Truncate(time.Hour * 24)

	if !mtime.Before(now) {
		res := make([]byte, stat.Size())
		f.Read(res)
		err = json.Unmarshal(res, &data)
		if err != nil {
			panic(err)
		}
		return data
	} else {
		return getSunData()
	}
}

func getSunData() SunData {
	var data SunData
	loc := getLocData()
	req, err := http.NewRequest("GET", "https://api.sunrise-sunset.org/json", nil)
	if err != nil {
		panic(err)
	}
	q := req.URL.Query()
	q.Add("lat", strconv.FormatFloat(loc.Latitude, 'f', -1, 64))
	q.Add("long", strconv.FormatFloat(loc.Latitude, 'f', -1, 64))
	q.Add("formatted", "0")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(fmt.Sprintf("%s/.sun.json", home), body, 0600)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		panic(err)
	}
	return data
}

func getLocData() LocData {
	var data LocData
	resp, err := http.Get("https://ifconfig.co/json")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		panic(err)
	}
	return data
}

// dirty hack because the API return a weird non-dst delta
func adjustedNow() time.Time {
	now := time.Now()
	if now.IsDST() {
		now = now.Add(time.Hour * 1)
	}
	return now
}

func eventDurationFromNow(event time.Time) string {
	d := event.Sub(adjustedNow())
	d = d.Abs().Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	return fmt.Sprintf("%02dh %02dm", h, m)
}

type SunData struct {
	Results struct {
		Sunrise time.Time `json:"sunrise"`
		Sunset  time.Time `json:"sunset"`
	} `json:"results"`
}

type LocData struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
