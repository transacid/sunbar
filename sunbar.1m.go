// usr/bin/env go run $0 $@; exit
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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

// set to true when you want to automatically set darkmode on sunset
const DARKMODE_SWITCH = false

func main() {
	data, err := getData()
	if err != nil {
		panic(err)
	}
	sunrise := data.Sunrise
	sunset := data.Sunset
	now := time.Now()
	var onOff string

	switch {
	case sunrise.After(now) && sunset.After(now):
		fmt.Print(Printer("sunrise", eventDurationFromNow(sunrise), "sunset", eventDurationFromNow(sunset)))
		onOff = "true"
	case sunrise.Before(now) && sunset.After(now):
		fmt.Print(Printer("sunset", eventDurationFromNow(sunset), "sunrise", eventDurationFromNow(sunrise.Add(time.Hour*24))))
		onOff = "false"
	case sunrise.Before(now) && sunset.Before(now):
		fmt.Print(Printer("sunrise", eventDurationFromNow(sunrise.Add(time.Hour*24)), "sunset", eventDurationFromNow(sunset.Add(time.Hour*24))))
		onOff = "true"
	}

	if DARKMODE_SWITCH {
		dMcmd := exec.Command("osascript", "-e", fmt.Sprintf("tell app \"System Events\" to tell appearance preferences to set dark mode to %s", onOff))
		dMcmd.Run()
		nScmd := exec.Command("shortcuts", "run", fmt.Sprintf("nightshift %s", onOff))
		nScmd.Run()
	}
}

func Printer(nowEvent, nowDuration, nextEvent, nextDuration string) string {
	return fmt.Sprintf(":%s: %s\n---\n:%s: %s | [symbolize = true]", nowEvent, nowDuration, nextEvent, nextDuration)
}

func getData() (sunData, error) {
	var sdata sunData
	var data map[string]any
	home, err := os.UserHomeDir()
	if err != nil {
		return sdata, err
	}
	f, err := os.Open(fmt.Sprintf("%s/.sun.json", home))
	if err != nil {
		return sdata, err
	}
	stat, err := f.Stat()
	if err != nil {
		return sdata, err
	}
	mtime := stat.ModTime()
	now := time.Now().Truncate(time.Hour * 24)

	if !mtime.Before(now) {
		res := make([]byte, stat.Size())
		f.Read(res)
		err = json.Unmarshal(res, &data)
		if err != nil {
			return sdata, err
		}

		parsedSunrise, parsedSunset, err := parseDates(data)
		if err != nil {
			return sdata, err
		}
		sdata.Sunrise = parsedSunrise
		sdata.Sunset = parsedSunset
		return sdata, nil
	} else {
		sdata, err = getSunData()
		if err != nil {
			return sdata, err
		}
		return sdata, nil
	}
}

func getSunData() (sunData, error) {
	lat, long, err := getLocation()
	if err != nil {
		return sunData{}, err
	}
	var sdata sunData
	req, err := http.NewRequest("GET", "https://api.ipgeolocation.io/astronomy", nil)
	if err != nil {
		return sdata, err
	}
	queryString := req.URL.Query()
	queryString.Add("apiKey", API_KEY)
	queryString.Add("lat", lat)
	queryString.Add("long", long)
	req.URL.RawQuery = queryString.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return sdata, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return sdata, err
	}
	if resp.StatusCode != http.StatusOK {
		return sdata, fmt.Errorf("%s\n%s", resp.Status, body)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return sdata, err
	}
	err = os.WriteFile(fmt.Sprintf("%s/.sun.json", home), body, 0600)
	if err != nil {
		return sdata, err
	}
	var rdata map[string]any
	err = json.Unmarshal(body, &rdata)
	if err != nil {
		return sdata, err
	}

	sunrise, sunset, err := parseDates(rdata)
	if err != nil {
		return sdata, err
	}
	sdata.Sunrise = sunrise
	sdata.Sunset = sunset
	return sdata, nil
}

func parseDates(dates map[string]any) (time.Time, time.Time, error) {
	sunRiseSplit := strings.Split(dates["sunrise"].(string), ":")
	sunSetSplit := strings.Split(dates["sunset"].(string), ":")
	sunRiseHour, err := strconv.Atoi(sunRiseSplit[0])
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	SunRiseMinute, err := strconv.Atoi(sunRiseSplit[1])
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	sunSetHour, err := strconv.Atoi(sunSetSplit[0])
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	sunSetMinute, err := strconv.Atoi(sunSetSplit[1])
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	now := time.Now()
	sunrise := time.Date(now.Year(), now.Month(), now.Day(), sunRiseHour, SunRiseMinute, 0, 0, time.Local)
	sunset := time.Date(now.Year(), now.Month(), now.Day(), sunSetHour, sunSetMinute, 0, 0, time.Local)
	return sunrise, sunset, nil
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

type sunData struct {
	Sunrise time.Time
	Sunset  time.Time
}

func getLocation() (string, string, error) {
	resp, err := http.Get("https://ifconfig.co/json")
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", "", err
	}

	lat := strconv.FormatFloat(data["latitude"].(float64), 'f', 2, 64)
	long := strconv.FormatFloat(data["longitude"].(float64), 'f', 2, 64)

	return lat, long, nil
}
