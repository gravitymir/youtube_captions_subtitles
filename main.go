package youtube_captions_subtitles

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type SubtitlesXML []struct {
	Text  string `xml:",chardata"`
	Start string `xml:"start,attr"`
	Dur   string `xml:"dur,attr"`
}

type CaptionTracks []track

type track struct {
	BaseURL string `json:"baseUrl"`
	Name    struct {
		SimpleText string `json:"simpleText"`
	} `json:"name"`
	VssID          string `json:"vssId"`
	LanguageCode   string `json:"languageCode"`
	Kind           string `json:"kind"`
	IsTranslatable bool   `json:"isTranslatable"`
}

func requestToYouTybe(videoIDorURL string) ([]byte, error) {
	regular, err := regexp.Compile(`([a-zA-Z0-9-_]{11})`)
	if err != nil {
		return nil, err
	}
	if regular.Match([]byte(videoIDorURL)) != true {
		return nil, errors.New(fmt.Sprintf("error parse args requestToYouTybe: \"%s\"", videoIDorURL))
	}
	ID := regular.Find([]byte(videoIDorURL))

	res, err := http.Get(fmt.Sprintf("https://youtube.com/watch?v=%s", ID))
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(res.Body)

	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		errStr := fmt.Sprintf("http StatusCode is %d", res.StatusCode)
		return nil, errors.New(errStr)
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("data convert to string: %s\n", "OK")
		return data, err
	}
	return data, err
}

func getSubtitlesAllLanguages(ID string) (tracks CaptionTracks, err error) {
	data, err := requestToYouTybe(ID)
	if err != nil {
		return nil, err
	}

	regular, err := regexp.Compile(`("captionTracks":.*isTranslatable":(true|false)}])`)
	if err != nil {
		return nil, err
	}
	if regular.Match(data) != true {
		return nil, errors.New(fmt.Sprintf("captions not found on video: \"%s\"", ID))
	}
	result := regular.Find(data)

	jsonByteArray := append([]byte{123}, result...) //add {
	jsonByteArray = append(jsonByteArray, 125)      //add }

	var cptn struct {
		CaptionTracks `json:"captionTracks"`
	}
	if err = json.Unmarshal(jsonByteArray, &cptn); err != nil {
		return nil, err
	}

	return cptn.CaptionTracks, err
}

func getSubtitles(ID, LanguageCode string) (subtitles SubtitlesXML, err error) {
	var tr track

	trackArray, err := getSubtitlesAllLanguages(ID)
	if err != nil {
		return nil, err
	}
	if LanguageCode != "" {
		for _, el := range trackArray {
			if el.LanguageCode == LanguageCode {
				tr = el
			}
		}
	} else {
		tr = trackArray[0]
	}

	if tr.BaseURL == "" {
		return subtitles, errors.New(fmt.Sprintf("subtitles ID: \"%s\" LanguageCode: \"%s\" not found", ID, LanguageCode))
	}

	resp, err := http.Get(tr.BaseURL)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)

	if err != nil {
		return subtitles, err
	}

	bodyByte, err := io.ReadAll(resp.Body)
	if err != nil {
		return subtitles, err
	}

	//bytes.Replace()
	var trnscrpt struct {
		SubtitlesXML `xml:"text"`
	}
	if err = xml.Unmarshal(bodyByte, &trnscrpt); err != nil {
		return subtitles, err
	}
	return trnscrpt.SubtitlesXML, err
}

// GetInfo show all available subtitle language codes
// args YouTybe video ID string len(11) "CLkkj3aka4g"
// or full url "https://www.youtube.com/watch?v=CLkkj3aka4g"
// return string JSON array of languages[Name:LanguageCode]
func GetInfo(ID string) (string, error) {
	if tracks, err := getSubtitlesAllLanguages(ID); err != nil {
		return "", err
	} else {
		if langInfoArray, err := json.MarshalIndent(tracks, "", "    "); err != nil {
			return "", err
		} else {
			str := strings.Replace(string(langInfoArray), `\u003c`, "<", -1)
			str = strings.Replace(str, `\u003e`, `>`, -1)
			return strings.Replace(str, `\u0026`, `&`, -1), nil
		}
	}
}

// GetStructSlice get subtitles into Golang []struct{Text, Start, Dur}
// it ready to use in your Golang code
func GetStructSlice(ID, languageCode string) (subtitles SubtitlesXML, err error) {
	if subtitles, err = getSubtitles(ID, languageCode); err != nil {
		return nil, err
	} else {
		return subtitles, err
	}
}

// GetJson subtitles in string JSON
func GetJson(ID, languageCode string) (string, error) {
	if subtitles, err := getSubtitles(ID, languageCode); err != nil {
		return "", err
	} else {
		subtitlesJson, err := json.Marshal(subtitles)
		return string(subtitlesJson), err
	}
}

// GetJsonPretty return subtitles in string JSON with 4 spaces for better humans reading
// pretty style JSON print
func GetJsonPretty(ID, languageCode string) (string, error) {
	if subtitles, err := getSubtitles(ID, languageCode); err != nil {
		return "", err
	} else {
		subtitlesJson, err := json.MarshalIndent(subtitles, "", "    ")
		return string(subtitlesJson), err
	}
}
