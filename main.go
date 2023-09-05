package main

import (
	"context"
	"fmt"
	"log"
	"time"
	 "html/template"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Employee struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Phone    string `json:"phone"`
	Id       string `json:"id" bson:"_id,omitempty"`
}

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

var mg MongoInstance

func Connect() error {
	const dbName = "employee" // Substitua pelo nome do seu banco de dados
	const mongoURI = "mongodb+srv://Andre:113113@cluster0.46fvged.mongodb.net/" + dbName + "?retryWrites=true&w=majority"

	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		return err
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return err
	}

	databases, err := client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		return err
	}
	fmt.Println(databases)

	mg = MongoInstance{
		Client: client,
		Db:     client.Database(dbName),
	}
	return nil
}

func main() {
	if err := Connect(); err != nil {
		log.Fatal(err)
	}
	app := fiber.New()

	app.Get("/employee", func(c *fiber.Ctx) error {
		query := bson.D{{}}
		cursor, err := mg.Db.Collection("employee").Find(c.Context(), query)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		var employees []Employee

		if err := cursor.All(c.Context(), &employees); err != nil {
			return c.Status(500).SendString(err.Error())
		}
		
		c.JSON(employees)

        // Renderizar o template HTML
        tmpl, err := template.ParseFiles("employee_list.html")
        if err != nil {
            return c.Status(500).SendString(err.Error())
        }

        // Enviar a resposta HTML renderizada
        c.Set("Content-Type", "text/html")
        return tmpl.Execute(c.Response().BodyWriter(), employees)
	})

	app.Post("/employee", func(c *fiber.Ctx) error {
		collection := mg.Db.Collection("employee")
		employee := new(Employee)

		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		employee.Id = ""

		insertionResult, err := collection.InsertOne(c.Context(), employee)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}
		createdRecord := collection.FindOne(c.Context(), filter)

		createdEmployee := &Employee{}
		createdRecord.Decode(createdEmployee)

		return c.Status(201).JSON(createdEmployee)
	})

	app.Put("/employee/:id", func(c *fiber.Ctx) error {
		idParam := c.Params("id")
		employeeID, err := primitive.ObjectIDFromHex(idParam)

		if err != nil {
			return c.Status(400).SendString("Invalid ID format")
		}

		employee := new(Employee)

		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		query := bson.D{{Key: "_id", Value: employeeID}}
		update := bson.D{
			{Key: "$set",
				Value: bson.D{
					{Key: "name", Value: employee.Name},
					// Adicione outros campos que deseja atualizar aqui
				},
			},
		}

		_, err = mg.Db.Collection("employee").UpdateOne(c.Context(), query, update)

		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		employee.Id = idParam

		return c.Status(200).JSON(employee)
	})

	app.Delete("/employee/:id", func(c *fiber.Ctx) error {
		idParam := c.Params("id")
		employeeID, err := primitive.ObjectIDFromHex(idParam)

		if err != nil {
			return c.Status(400).SendString("Invalid ID format")
		}

		query := bson.D{{Key: "_id", Value: employeeID}}
		result, err := mg.Db.Collection("employee").DeleteOne(c.Context(), query)

		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		if result.DeletedCount < 1 {
			return c.Status(404).SendString("No such employee found")
		}

		return c.Status(200).JSON("record deleted")
	})

	// Define a porta em que o servidor Fiber será executado
	app.Listen(":3000")
}
