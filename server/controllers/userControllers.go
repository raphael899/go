package controllers

import (
	"context"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"firts-go-server/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson/primitive",
	"time"
	"golang.org/x/crypto/bcrypt"
	"github.com/dgrijalva/jwt-go"
)

// UserController es el controlador para las operaciones relacionadas con los usuarios
type UserController struct {
	coll *mongo.Collection
}

// NewUserController crea una instancia del controlador de usuarios
func NewUserController() *UserController {
	uri := "mongodb://localhost:27017/gomongodb"
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	coll := client.Database("gomongodb").Collection("users")

	return &UserController{
		coll: coll,
	}
}

// GetUsers devuelve todos los usuarios
func (uc *UserController) GetUsers(c *fiber.Ctx) error {
	var users []models.User
	results, err := uc.coll.Find(context.TODO(), bson.M{})
	if err != nil {
		panic(err)
	}

	for results.Next(context.TODO()) {
		var user models.User
		results.Decode(&user)
		users = append(users, user)
	}

	return c.JSON(&fiber.Map{
		"data": users,
	})
}

// AddUser agrega un nuevo usuario
func (uc *UserController) AddUser(c *fiber.Ctx) error {
	var user models.User
	c.BodyParser(&user)
	filter := bson.D{{Key: "email", Value: user.Email}}

	var resultQuery models.User

	if user.Name == "" {
		return c.JSON(&fiber.Map{
			"data": "add user name",
		})
	}

	if user.Email == "" {
		return c.JSON(&fiber.Map{
			"data": "add user email",
		})
	}

	err := uc.coll.FindOne(context.TODO(), filter).Decode(&resultQuery)

	if resultQuery.Email == user.Email {
		return c.JSON(&fiber.Map{
			"data":    resultQuery,
			"message": "Email already exists",
		})
	}

	result, err := uc.coll.InsertOne(
		context.TODO(),
		bson.D{
			bson.E{Key: "name", Value: user.Name},
			bson.E{Key: "email", Value: user.Email},
		},
	)

	if err != nil {
		panic(err)
	}

	return c.JSON(&fiber.Map{
		"data": result,
	})
}

// UpdateUser actualiza un usuario existente
func (uc *UserController) UpdateUser(c *fiber.Ctx) error {
	var user models.User
	c.BodyParser(&user)
	idParams := c.Params("id")

	// Make a copy
	buffer := make([]byte, len(idParams))
	copy(buffer, idParams)
	resultCopy := string(buffer)

	fmt.Println(resultCopy)

	id, _ := primitive.ObjectIDFromHex(resultCopy)
	filter := bson.D{{"_id", id}}

	update := bson.D{{"$set", bson.D{}}}
	if user.Name != "" {
		update[0].Value = append(update[0].Value.(bson.D), bson.E{Key: "name", Value: user.Name})
	}
	if user.Email != "" {
		update[0].Value = append(update[0].Value.(bson.D), bson.E{Key: "email", Value: user.Email})
	}

	result, err := uc.coll.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		panic(err)
	}

	return c.JSON(&fiber.Map{
		"data": result,
	})
}

// GetUser obtiene un usuario por su ID
func (uc *UserController) GetUser(c *fiber.Ctx) error {
	idParams := c.Params("id")

	// Make a copy
	buffer := make([]byte, len(idParams))
	copy(buffer, idParams)
	resultCopy := string(buffer)

	id, _ := primitive.ObjectIDFromHex(resultCopy)
	filter := bson.D{{"_id", id}}

	var resultQuery models.User

	err := uc.coll.FindOne(context.TODO(), filter).Decode(&resultQuery)

	if err != nil {
		panic(err)
	}

	return c.JSON(&fiber.Map{
		"data": resultQuery,
	})
}

// DeleteUser elimina un usuario por su ID
func (uc *UserController) DeleteUser(c *fiber.Ctx) error {
	idParams := c.Params("id")

	// Make a copy
	buffer := make([]byte, len(idParams))
	copy(buffer, idParams)
	resultCopy := string(buffer)

	id, _ := primitive.ObjectIDFromHex(resultCopy)
	filter := bson.D{{"_id", id}}

	result, err := uc.coll.DeleteOne(context.TODO(), filter)
	if err != nil {
		panic(err)
	}

	return c.JSON(&fiber.Map{
		"data": result,
	})
}
