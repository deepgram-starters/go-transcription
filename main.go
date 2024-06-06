package main

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/deepgram-devs/deepgram-go-sdk/deepgram"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type TranscriptionResponse struct {
	Model            string                                   `json:"model,omitempty"`
	Version          string                                   `json:"version,omitempty"`
	Tier             string                                   `json:"tier,omitempty"`
	DeepgramFeatures deepgram.PreRecordedTranscriptionOptions `json:"dgFeatures,omitempty"`
	Transcription    deepgram.PreRecordedResponse             `json:"transcription,omitempty"`
}

func main() {
	godotenv.Load()
	dg := deepgram.NewClient(os.Getenv("deepgram_api_key"))

	r := gin.Default()
	r.Use(cors.Default())
	r.Static("/", "./static")

	r.POST("/api", transcribe(dg))

	r.Run("localhost:" + os.Getenv("port"))
}

func transcribe(dg *deepgram.Client) gin.HandlerFunc {

	fn := func(c *gin.Context) {
		url := c.PostForm("url")
		model := c.PostForm("model")
		version := c.PostForm("version")
		tier := c.PostForm("tier")
		features := c.PostForm("features")

		var dgFeatures deepgram.PreRecordedTranscriptionOptions
		err := json.Unmarshal([]byte(features), &dgFeatures)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
		}

		dgFeatures.Model = model
		if len(version) > 0 {
			dgFeatures.Version = version
		}

		if model != "whisper" {
			dgFeatures.Tier = tier
		}

		if strings.HasPrefix(url, "https://res.cloudinary.com/deepgram") {
			transcription, err := dg.PreRecordedFromURL(deepgram.UrlSource{Url: url}, dgFeatures)
			if err != nil {
				panic(err)
			}

			res := TranscriptionResponse{
				Model:            model,
				Version:          version,
				Tier:             tier,
				DeepgramFeatures: dgFeatures,
				Transcription:    transcription,
			}

			c.JSON(http.StatusOK, res)
		} else {
			file, err := c.FormFile("file")
			if err != nil {
				c.AbortWithError(http.StatusBadRequest, errors.New("you need to choose a file to transcribe your own audio"))
				return
			}

			uploadedFile, err := file.Open()
			if err != nil {
				c.AbortWithError(http.StatusBadRequest, errors.New("cannot open file"))
			}
			defer uploadedFile.Close()

			stream := uploadedFile.(io.ReadCloser)
			mime := mime.TypeByExtension(filepath.Ext(file.Filename))

			transcription, err := dg.PreRecordedFromStream(
				deepgram.ReadStreamSource{Stream: stream, Mimetype: mime},
				dgFeatures)
			if err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
			}

			res := TranscriptionResponse{
				Model:            model,
				Version:          version,
				Tier:             tier,
				DeepgramFeatures: dgFeatures,
				Transcription:    *transcription,
			}

			c.JSON(http.StatusOK, res)

		}
	}

	return gin.HandlerFunc(fn)
}
