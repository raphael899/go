package main

import (
	"context"
	"os"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"/firts-go-server/routes"
	"/firts-go-server/controllers"
)

func main() {
	app := fiber.New()

	port := os.Getenv("PORT")

	if port == "" {
		port = "3000"
	}

	uri := "mongodb://localhost:27017/gomongodb"

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}

	coll := client.Database("gomongodb").Collection("users")

	userController := controllers.NewUserController()

	app.Use(cors.New())
	app.Static("/", "./client/dist")

	routes.SetupUserRoutes(app, userController)

	app.Listen(":" + port)
}
