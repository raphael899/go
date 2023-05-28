package routes

import (
	"github.com/gofiber/fiber/v2"
	"firts-go-server/controllers"
)

// SetupUserRoutes configura las rutas relacionadas con los usuarios
func SetupUserRoutes(app *fiber.App, userController *controllers.UserController) {
	app.Get("/users", userController.GetUsers)
	app.Post("/users", userController.AddUser)
	app.Put("/users/:id", userController.UpdateUser)
	app.Get("/users/:id", userController.GetUser)
	app.Delete("/users/:id", userController.DeleteUser)
}
