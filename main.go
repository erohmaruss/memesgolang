//6400334073:AAE_vuF9PXDNttZztHiYteROadz9EEzcZuE

package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Message struct {
	Text      string
	MediaPath string
}

type Response struct {
	Text     string `json:"text"`
	File     string `json:"file"`
	FileName string `json:"fileName"`
	FileType string `json:"fileType"`
}

var messagesMap = make(map[int64]Message)
var counter int64 = 0

func main() {
	rand.Seed(time.Now().UnixNano())
	botToken := "6400334073:AAE_vuF9PXDNttZztHiYteROadz9EEzcZuE"

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}

	go func() {
		for update := range updates {
			if update.Message != nil {
				var msg Message
				msg.Text = update.Message.Text

				// Сохраняем фото
				if update.Message.Photo != nil && len(*update.Message.Photo) > 0 {
					photo := (*update.Message.Photo)[len(*update.Message.Photo)-1]
					fileURL, _ := bot.GetFileDirectURL(photo.FileID)
					mediaPath := downloadMedia(fileURL)
					msg.MediaPath = mediaPath
				}

				// Сохраняем видео
				if update.Message.Video != nil {
					video := update.Message.Video
					fileURL, err := bot.GetFileDirectURL(video.FileID)
					if err != nil {
						log.Printf("Failed to get file URL: %v", err)
						continue
					}
					mediaPath := downloadMedia(fileURL)
					msg.MediaPath = mediaPath
				}

				messagesMap[counter] = msg
				counter++
			}
		}
	}()

	http.HandleFunc("/getmessage", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Saved message. Total messages: %d", len(messagesMap))
		var randomMessage Message
		var keys []int64

		for k := range messagesMap {
			keys = append(keys, k)
		}

		if len(keys) == 0 {
			http.Error(w, "No messages found", http.StatusNotFound)
			return
		}

		randomKey := keys[rand.Intn(len(keys))]
		randomMessage = messagesMap[randomKey]

		fileData, err := ioutil.ReadFile(randomMessage.MediaPath)
		if err != nil {
			http.Error(w, "Failed to read media file", http.StatusInternalServerError)
			return
		}

		fileBase64 := base64.StdEncoding.EncodeToString(fileData)

		response := Response{
			Text:     randomMessage.Text,
			File:     fileBase64,
			FileName: filepath.Base(randomMessage.MediaPath),
			FileType: http.DetectContentType(fileData),
		}

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Failed to create JSON response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResponse)
	})

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func downloadMedia(url string) string {
	if _, err := os.Stat("media"); os.IsNotExist(err) {
		os.Mkdir("media", 0755)
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	fileName := filepath.Base(url)
	filePath := filepath.Join("media", fileName)

	out, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return filePath
}
