package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
)

type Request struct {
	Prompt string
	N      int
	Size   string
	Model  string
}

type Output struct {
	Output      []string `json:"output"`
	Status      string   `json:"status"`
	FetchResult string   `json:"fetch_result"`
}

type ImageItem struct {
	Url string `json:"url"`
}

var STABLEDIFFUSION_API_KEY string
var OPENAI_API_KEY string

func main() {
	godotenv.Load()

	port := ":8080"
	if os.Getenv("PORT") != "" {
		port = ":" + os.Getenv("PORT")
	}

	STABLEDIFFUSION_API_KEY = os.Getenv("STABLEDIFFUSION_API_KEY")
	OPENAI_API_KEY = os.Getenv("OPENAI_API_KEY")
	app := fiber.New(fiber.Config{
		Network: fiber.NetworkTCP6,
	})
	app.Use(cors.New(cors.Config{
		AllowOrigins: "https://ttimage.vercel.app, http://localhost:3000",
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Working fine")
	})
	app.Post("/image", func(c *fiber.Ctx) error {
		var request Request

		err := c.BodyParser(&request)

		if err != nil {
			return c.Status(403).JSON(fiber.Map{
				"status":  false,
				"message": "Failed to parse json",
			})
		}

		if request.Model == "dalle" {
			client := openai.NewClient(OPENAI_API_KEY)

			reqUrl := openai.ImageRequest{
				Prompt:         request.Prompt,
				Size:           openai.CreateImageSize512x512,
				ResponseFormat: openai.CreateImageResponseFormatURL,
				N:              request.N,
			}

			respUrl, err := client.CreateImage(context.Background(), reqUrl)

			if err != nil {
				fmt.Printf("Image creation error: %v\n", err)
				return c.Status(403).JSON(fiber.Map{
					"status":  false,
					"message": err,
				})
			}
			fmt.Println(respUrl.Data)
			return c.Status(200).JSON(fiber.Map{
				"status": "true",
				"data": fiber.Map{
					"output": respUrl.Data,
				},
			})

		} else {
			res, err := getImage(request.Prompt, request.Model, request.N)

			if err != nil {
				return c.Status(403).JSON(fiber.Map{
					"status":  false,
					"message": err,
				})
			}

			return c.Status(200).JSON(fiber.Map{
				"status": true,
				"data":   res,
			})
		}

	})

	app.Listen(port)
}

func getImage(prompt string, model string, n int) (respose Output, err error) {
	client := &http.Client{}
	text := fmt.Sprintf(`{
		"key": "%v",
		"model_id": "%v",
		"prompt": "%v",
		"negative_prompt": "",
		"width": "512",
		"height": "512",
		"samples": "%v",
		"num_inference_steps": "30",
		"safety_checker": "no",
		"enhance_prompt": "no",
		"seed": "null",
		"guidance_scale": 7.5,
		"webhook": null,
		"track_id": null
	  }`, STABLEDIFFUSION_API_KEY, model, prompt, n)
	var data = strings.NewReader(text)
	req, err := http.NewRequest("POST", "https://stablediffusionapi.com/api/v4/dreambooth", data)
	if err != nil {
		return Output{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	// req.Header.Set("accept-encoding", "gzip")
	resp, err := client.Do(req)
	if err != nil {
		return Output{}, err
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return Output{}, err
	}

	var result Output
	json.Unmarshal(bodyText, &result)

	return result, nil
}
