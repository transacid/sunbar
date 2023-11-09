//usr/bin/env go run $0 $@; exit
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
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

// get your API key for free at https://ipgeolocation.io/
const API_KEY = "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"

func main() {
	data := getData()
	sunrise := data.Sunrise
	sunset := data.Sunset
	now := time.Now()

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

func getData() Data {
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
		var d Data
		psr, pss := parseDates(data)
		d.Sunrise = psr
		d.Sunset = pss
		return d
	} else {
		return getSunData()
	}
}

func getSunData() Data {
	var data SunData
	req, err := http.NewRequest("GET", "https://api.ipgeolocation.io/astronomy", nil)
	if err != nil {
		panic(err)
	}
	q := req.URL.Query()
	q.Add("apiKey", API_KEY)
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
	var d Data
	psr, pss := parseDates(data)
	d.Sunrise = psr
	d.Sunset = pss
	return d
}

func parseDates(body SunData) (time.Time, time.Time) {
	var sunrise HourMinute
	var sunset HourMinute
	srs := strings.Split(body.Sunrise, ":")
	sts := strings.Split(body.Sunset, ":")
	srhi, _ := strconv.ParseInt(srs[0], 10, 0)
	srmi, _ := strconv.ParseInt(srs[1], 10, 0)
	sshi, _ := strconv.ParseInt(sts[0], 10, 0)
	ssmi, _ := strconv.ParseInt(sts[1], 10, 0)
	sunrise.h = int(srhi)
	sunrise.m = int(srmi)
	sunset.h = int(sshi)
	sunset.m = int(ssmi)
	n := time.Now()
	sunr := time.Date(n.Year(), n.Month(), n.Day(), sunrise.h, sunrise.m, 0, 0, time.Local)
	suns := time.Date(n.Year(), n.Month(), n.Day(), sunset.h, sunset.m, 0, 0, time.Local)
	return sunr, suns
}

func eventDurationFromNow(event time.Time) string {
	d := time.Until(event)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	if h == 0 {
		return fmt.Sprintf("%dm", m)
	} else {
		return fmt.Sprintf("%dh %dm", h, m)
	}
}

type SunData struct {
	Sunrise string `json:"sunrise"`
	Sunset  string `json:"sunset"`
}

type HourMinute struct {
	h int
	m int
}

type Data struct {
	Sunrise time.Time
	Sunset  time.Time
}
